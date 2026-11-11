package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsefremov/kadr/internal/feed"
	"github.com/dsefremov/kadr/internal/media"
)

func (s *Server) registerMediaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /m/{mediaId}/{file}", s.serveMedia)
	mux.HandleFunc("GET /api/me/media", s.listMyMedia)
	mux.HandleFunc("GET /api/media/{id}", s.getMediaMeta)
	mux.HandleFunc("PATCH /api/media/{id}/taken-date", s.patchMediaTakenDate)
}

type patchTakenDateBody struct {
	TakenDate string `json:"taken_date"`
}

func (s *Server) patchMediaTakenDate(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	var body patchTakenDateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TakenDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	id := r.PathValue("id")
	err := media.SetTakenDate(r.Context(), s.db, sess.UserID, id, body.TakenDate)
	if errors.Is(err, media.ErrBadDate) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_date"})
		return
	}
	if errors.Is(err, media.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getMediaMeta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row, takenDate, err := media.DetailByPublicID(r.Context(), s.db, id)
	if errors.Is(err, media.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	var visibility string
	var ownerID int64
	err = s.db.QueryRowContext(r.Context(), `SELECT visibility, owner_id FROM feeds WHERE id = ?`, row.FeedID).
		Scan(&visibility, &ownerID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sess := s.currentSession(r)
	owner := sess != nil && sess.UserID == ownerID
	if row.State != "ready" {
		http.NotFound(w, r)
		return
	}
	if visibility != "public" && !owner {
		http.NotFound(w, r)
		return
	}
	urls := feed.BuildURLsForKind(row.PublicID, row.Kind)
	if visibility == "public" {
		w.Header().Set("X-Robots-Tag", "noindex")
	}
	out := map[string]any{
		"id": row.PublicID, "kind": row.Kind,
		"taken_date": takenDate, "taken_sort": row.TakenSort.UTC().Format("2006-01-02T15:04:05.000000Z"),
		"urls": urls, "owner": owner,
	}
	if row.Width.Valid {
		out["width"] = row.Width.Int64
	}
	if row.Height.Valid {
		out["height"] = row.Height.Int64
	}
	if row.DurationMs.Valid {
		out["duration_ms"] = row.DurationMs.Int64
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := r.PathValue("mediaId")
	file := r.PathValue("file")
	dot := strings.LastIndex(file, ".")
	if dot <= 0 {
		http.NotFound(w, r)
		return
	}
	variant := file[:dot]
	profileHash := file[dot+1:]
	if variant == "" || profileHash == "" {
		http.NotFound(w, r)
		return
	}
	row, err := media.ByPublicID(r.Context(), s.db, mediaID)
	if errors.Is(err, media.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	var visibility string
	var ownerID int64
	err = s.db.QueryRowContext(r.Context(), `SELECT visibility, owner_id FROM feeds WHERE id = ?`, row.FeedID).
		Scan(&visibility, &ownerID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var trashed sql.NullTime
	_ = s.db.QueryRowContext(r.Context(), `SELECT trashed_at FROM media WHERE public_id = ?`, mediaID).Scan(&trashed)
	sess := s.currentSession(r)
	owner := sess != nil && sess.UserID == ownerID
	if row.State != "ready" {
		http.NotFound(w, r)
		return
	}
	if trashed.Valid {
		if !owner {
			http.NotFound(w, r)
			return
		}
	} else if visibility != "public" && !owner {
		http.NotFound(w, r)
		return
	}
	storageKey, mime, err := media.VariantPath(r.Context(), s.db, mediaID, variant, profileHash)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.opts.MediaDir, filepath.FromSlash(storageKey))
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	stat, _ := f.Stat()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, filepath.Base(path), stat.ModTime(), f)
}

func (s *Server) listMyMedia(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	state := r.URL.Query().Get("status")
	rows, err := media.ListByOwner(r.Context(), s.db, sess.UserID, state)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	type item struct {
		ID    string `json:"id"`
		Kind  string `json:"kind"`
		State string `json:"state"`
		Error string `json:"error,omitempty"`
	}
	out := make([]item, 0, len(rows))
	for _, row := range rows {
		it := item{ID: row.PublicID, Kind: row.Kind, State: row.State}
		if row.ErrorMessage.Valid {
			it.Error = row.ErrorMessage.String
		}
		out = append(out, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{"media": out})
}
