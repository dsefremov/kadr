package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dsefremov/kadr/internal/api"
	"github.com/dsefremov/kadr/internal/config"
	"github.com/dsefremov/kadr/internal/db"
	"github.com/dsefremov/kadr/internal/logx"
	"github.com/dsefremov/kadr/internal/mail"
	"github.com/dsefremov/kadr/internal/migrate"
	"github.com/dsefremov/kadr/internal/worker"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kadr <api|worker|migrate>")
		os.Exit(2)
	}
	cmd := os.Args[1]
	cfg, err := config.Load(cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log := logx.New(cfg.LogLevel, cfg.LogFormat)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var database *sql.DB
	for attempt := 1; attempt <= 30; attempt++ {
		database, err = db.Open(cfg.MySQLDSN)
		if err == nil {
			break
		}
		log.Warn("database not ready", "attempt", attempt, "err", err)
		time.Sleep(2 * time.Second)
	}
	if database == nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	switch cmd {
	case "migrate":
		if err := migrate.Up(ctx, database); err != nil {
			log.Error("migrate", "err", err)
			os.Exit(1)
		}
		log.Info("migrations applied")
		return
	case "api":
		srv := api.New(log, database, api.Options{
			PublicURL: cfg.PublicURL, CookieSecure: cfg.CookieSecure, MailFrom: cfg.MailFrom,
		})
		httpSrv := &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           srv.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			log.Info("api listening", "addr", cfg.HTTPAddr)
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("api", "err", err)
				stop()
			}
		}()
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdown)
	case "worker":
		smtp := &mail.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.MailFrom}
		w := worker.New(log, database, cfg.WorkerPoll, smtp, cfg.PublicURL)
		if err := w.Run(ctx); err != nil && err != context.Canceled {
			log.Error("worker", "err", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		os.Exit(2)
	}
}
