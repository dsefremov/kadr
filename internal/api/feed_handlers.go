package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/dsefremov/kadr/internal/feed"
)

func (s *Server) registerFeedRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/u/{handle}/feed", s.getPublicFeed)
	mux.HandleFunc("GET /api/feeds/{feedId}/timeline", s.getTimeline)
	mux.HandleFunc("PATCH /api/me/feed", s.patchMyFeed)
}

func (s *Server) getPublicFeed(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("handle")
	meta, err := feed.ByHandle(r.Context(), s.db, handle)
	if errors.Is(err, feed.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	sess := s.currentSession(r)
	if meta.Visibility != "public" && (sess == nil || sess.UserID != meta.OwnerID) {
		http.NotFound(w, r)
		return
	}
	if meta.Visibility == "public" {
		w.Header().Set("X-Robots-Tag", "noindex")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": meta.PublicID, "handle": meta.Handle, "visibility": meta.Visibility,
	})
}

func (s *Server) getTimeline(w http.ResponseWriter, r *http.Request) {
	feedID := r.PathValue("feedId")
	meta, err := feed.ByPublicID(r.Context(), s.db, feedID)
	if errors.Is(err, feed.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	sess := s.currentSession(r)
	if meta.Visibility != "public" && (sess == nil || sess.UserID != meta.OwnerID) {
		http.NotFound(w, r)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := feed.Timeline(r.Context(), s.db, feedID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type patchFeedBody struct {
	Visibility string `json:"visibility"`
}

func (s *Server) patchMyFeed(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	var body patchFeedBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	if err := feed.SetVisibility(r.Context(), s.db, sess.UserID, body.Visibility); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
