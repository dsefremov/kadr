package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dsefremov/kadr/internal/media"
	"github.com/dsefremov/kadr/internal/session"
	"github.com/dsefremov/kadr/internal/upload"
)

func (s *Server) registerUploadRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/uploads", s.postUpload)
	mux.HandleFunc("HEAD /api/uploads/{id}", s.headUpload)
	mux.HandleFunc("PATCH /api/uploads/{id}", s.patchUpload)
	mux.HandleFunc("DELETE /api/uploads/{id}", s.deleteUpload)
	mux.HandleFunc("GET /api/me/uploads", s.listMyUploads)
}

func (s *Server) postUpload(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	if r.Header.Get("Tus-Resumable") != upload.TusVersion {
		http.Error(w, "tus version required", http.StatusPreconditionFailed)
		return
	}
	length, err := strconv.ParseInt(r.Header.Get("Upload-Length"), 10, 64)
	if err != nil || length <= 0 {
		http.Error(w, "Upload-Length required", http.StatusBadRequest)
		return
	}
	kind := upload.ParseKindFromMetadata(r.Header.Get("Upload-Metadata"))
	row, err := s.uploads.Create(r.Context(), sess.UserID, uint64(length), kind)
	if errors.Is(err, upload.ErrLength) {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}
	if errors.Is(err, upload.ErrQuota) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota exceeded"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	loc := s.uploadLocation(row.PublicID)
	w.Header().Set("Location", loc)
	w.Header().Set("Tus-Resumable", upload.TusVersion)
	w.Header().Set("Upload-Offset", "0")
	w.Header().Set("Tus-Max-Size", strconv.FormatInt(int64(maxSize(kind)), 10))
	w.WriteHeader(http.StatusCreated)
}

func maxSize(kind string) int64 {
	if kind == "video" {
		return 2_000_000_000
	}
	return 50_000_000
}

func (s *Server) uploadLocation(publicID string) string {
	// Relative URL keeps tus PATCH on the page origin (vite preview, nginx, etc.).
	return "/api/uploads/" + publicID
}

func (s *Server) headUpload(w http.ResponseWriter, r *http.Request) {
	sess := s.currentSession(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id := r.PathValue("id")
	row, err := s.uploads.Get(r.Context(), id, sess.UserID)
	if errors.Is(err, upload.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	w.Header().Set("Tus-Resumable", upload.TusVersion)
	w.Header().Set("Upload-Offset", strconv.FormatUint(row.Offset, 10))
	w.Header().Set("Upload-Length", strconv.FormatUint(row.UploadLength, 10))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) patchUpload(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	if r.Header.Get("Tus-Resumable") != upload.TusVersion {
		http.Error(w, "tus version required", http.StatusPreconditionFailed)
		return
	}
	off, err := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
	if err != nil {
		http.Error(w, "bad offset", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	row, err := s.uploads.Patch(r.Context(), id, sess.UserID, off, r.Body)
	var dup *upload.DuplicateMediaError
	if errors.As(err, &dup) {
		ph := media.ProfileHashV1SDR
		errCode := "duplicate"
		if dup.InTrash {
			errCode = "duplicate_in_trash"
		}
		body := map[string]any{
			"error":        errCode,
			"media_id":     dup.MediaPublicID,
			"media_state":  dup.MediaState,
			"preview_url":  fmt.Sprintf("/m/%s/preview-640.%s", dup.MediaPublicID, ph),
			"timeline_url": fmt.Sprintf("/api/feeds/%s/timeline", s.feedPublicIDForUser(r.Context(), sess.UserID)),
		}
		if dup.InTrash {
			body["restore_url"] = fmt.Sprintf("/api/media/%s/restore", dup.MediaPublicID)
		}
		writeJSON(w, http.StatusConflict, body)
		w.Header().Set("Tus-Resumable", upload.TusVersion)
		return
	}
	if errors.Is(err, upload.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, upload.ErrOffset) {
		w.Header().Set("Tus-Resumable", upload.TusVersion)
		http.Error(w, "offset conflict", http.StatusConflict)
		return
	}
	if errors.Is(err, upload.ErrLength) {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	w.Header().Set("Tus-Resumable", upload.TusVersion)
	w.Header().Set("Upload-Offset", strconv.FormatUint(row.Offset, 10))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteUpload(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	err := s.uploads.Delete(r.Context(), id, sess.UserID)
	if errors.Is(err, upload.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMyUploads(w http.ResponseWriter, r *http.Request) {
	sess := s.requireVerified(w, r)
	if sess == nil {
		return
	}
	rows, err := s.uploads.ListActive(r.Context(), sess.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	type item struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Length   uint64 `json:"length"`
		Offset   uint64 `json:"offset"`
		State    string `json:"state"`
	}
	out := make([]item, 0, len(rows))
	for _, row := range rows {
		out = append(out, item{
			ID: row.PublicID, Kind: row.Kind, Length: row.UploadLength, Offset: row.Offset, State: row.State,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"uploads": out})
}

func (s *Server) feedPublicIDForUser(ctx context.Context, userID int64) string {
	var pub string
	_ = s.db.QueryRowContext(ctx, `SELECT public_id FROM feeds WHERE owner_id = ?`, userID).Scan(&pub)
	return pub
}

func (s *Server) requireVerified(w http.ResponseWriter, r *http.Request) *session.Data {
	sess := s.currentSession(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return nil
	}
	if !sess.Verified {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "email not verified"})
		return nil
	}
	return sess
}
