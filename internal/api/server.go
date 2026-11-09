package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/dsefremov/kadr/internal/auth"
	"github.com/dsefremov/kadr/internal/db"
	"github.com/dsefremov/kadr/internal/session"
	"github.com/dsefremov/kadr/internal/version"
)

type Server struct {
	log      *slog.Logger
	db       *sql.DB
	auth     *auth.Service
	sessions *session.Store
	opts     Options
}

func New(log *slog.Logger, database *sql.DB, opts Options) *Server {
	return &Server{
		log:      log,
		db:       database,
		auth:     &auth.Service{DB: database, PublicPath: opts.PublicURL},
		sessions: &session.Store{DB: database},
		opts:     opts,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /api/health", s.apiHealth)
	s.registerAuthRoutes(mux)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := db.Ping(ctx, s.db); err != nil {
		s.log.Warn("readiness failed", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": "database"})
		return
	}
	applied, err := db.MigrationApplied(ctx, s.db)
	if err != nil || !applied {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": "migrations"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) apiHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	dbOK := db.Ping(ctx, s.db) == nil
	migOK := false
	if dbOK {
		migOK, _ = db.MigrationApplied(ctx, s.db)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"db":     dbOK,
		"schema": migOK,
		"build":  version.Info(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
