package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dsefremov/kadr/internal/media"
)

func (s *Server) registerTrashRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/media/{id}/trash", s.postMediaTrash)
	mux.HandleFunc("POST /api/media/{id}/restore", s.postMediaRestore)
	mux.HandleFunc("GET /api/me/trash", s.getMyTrash)
	mux.HandleFunc("DELETE /api/me/trash", s.deleteMyTrash)
	mux.HandleFunc("DELETE /api/media/{id}/trash", s.deleteMediaTrashForever)
}

func (s *Server) deleteMediaTrashForever(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	err := media.DeleteTrashedByPublicID(r.Context(), s.db, s.opts.MediaDir, sess.UserID, id)
	if errors.Is(err, media.ErrNotFound) || errors.Is(err, media.ErrTrashOwner) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, media.ErrNotTrashed) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "not_in_trash"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "purged"})
}

func (s *Server) postMediaTrash(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	err := media.TrashByPublicID(r.Context(), s.db, sess.UserID, id)
	if errors.Is(err, media.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, media.ErrTrashOwner) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, media.ErrTrashState) || errors.Is(err, media.ErrAlreadyTrashed) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "trashed"})
}

func (s *Server) postMediaRestore(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	err := media.RestoreByPublicID(r.Context(), s.db, sess.UserID, id)
	if errors.Is(err, media.ErrNotFound) || errors.Is(err, media.ErrTrashOwner) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, media.ErrNotTrashed) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "not_in_trash"})
		return
	}
	if errors.Is(err, media.ErrWaitingQuota) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota_exceeded"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func (s *Server) getMyTrash(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	items, err := media.ListTrash(r.Context(), s.db, sess.UserID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	ph := media.ProfileHashV1SDR
	type row struct {
		ID          string `json:"id"`
		Kind        string `json:"kind"`
		TrashedAt   string `json:"trashed_at"`
		PurgeAfter  string `json:"purge_after"`
		DaysLeft    int    `json:"days_left"`
		PreviewURL  string `json:"preview_url"`
		StoredBytes uint64 `json:"stored_bytes"`
	}
	out := make([]row, 0, len(items))
	now := time.Now().UTC()
	for _, it := range items {
		days := int(it.PurgeAfter.Sub(now).Hours() / 24)
		if days < 0 {
			days = 0
		}
		out = append(out, row{
			ID:          it.PublicID,
			Kind:        it.Kind,
			TrashedAt:   it.TrashedAt.UTC().Format(time.RFC3339),
			PurgeAfter:  it.PurgeAfter.UTC().Format(time.RFC3339),
			DaysLeft:    days,
			PreviewURL:  fmt.Sprintf("/m/%s/preview-640.%s", it.PublicID, ph),
			StoredBytes: it.StoredBytes,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) deleteMyTrash(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	n, err := media.EmptyTrash(r.Context(), s.db, s.opts.MediaDir, sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
}
