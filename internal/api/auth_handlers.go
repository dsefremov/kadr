package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dsefremov/kadr/internal/auth"
	"github.com/dsefremov/kadr/internal/feed"
	"github.com/dsefremov/kadr/internal/httpx"
	"github.com/dsefremov/kadr/internal/ratelimit"
	"github.com/dsefremov/kadr/internal/session"
)

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", s.postRegister)
	mux.HandleFunc("POST /api/auth/verify", s.postVerify)
	mux.HandleFunc("POST /api/auth/verify/resend", s.postResend)
	mux.HandleFunc("POST /api/auth/login", s.postLogin)
	mux.HandleFunc("POST /api/auth/logout", s.postLogout)
	mux.HandleFunc("POST /api/auth/password/reset", s.postPasswordReset)
	mux.HandleFunc("GET /api/me", s.getMe)
}

type registerBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) postRegister(w http.ResponseWriter, r *http.Request) {
	var body registerBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	err := s.auth.Register(r.Context(), auth.RegisterInput{
		Email: body.Email, Password: body.Password,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Проверьте почту — мы отправили письмо со ссылкой для подтверждения.",
	})
}

type tokenBody struct {
	Token string `json:"token"`
}

func (s *Server) postVerify(w http.ResponseWriter, r *http.Request) {
	var body tokenBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
		return
	}
	if err := s.auth.Verify(r.Context(), body.Token); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

type resendBody struct {
	Email string `json:"email"`
}

func (s *Server) postResend(w http.ResponseWriter, r *http.Request) {
	var body resendBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	ip := httpx.ClientIP(r)
	allowed, err := ratelimit.ResendIPAllowed(r.Context(), s.db, ip)
	if err != nil || !allowed {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Если адрес зарегистрирован, мы отправили новое письмо.",
		})
		return
	}
	err = s.auth.ResendVerify(r.Context(), body.Email)
	if err == nil {
		_ = ratelimit.RecordResend(r.Context(), s.db, strings.ToLower(strings.TrimSpace(body.Email)), ip)
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Если адрес зарегистрирован, мы отправили новое письмо.",
	})
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	ip := httpx.ClientIP(r)
	if err := ratelimit.LoginAllowed(r.Context(), s.db, email); errors.Is(err, ratelimit.ErrLoginLocked) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Слишком много попыток. Подождите 15 минут."})
		return
	}
	userID, err := s.auth.Login(r.Context(), body.Email, body.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		_ = ratelimit.RecordLoginFailure(r.Context(), s.db, email, ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Неверный email или пароль"})
		return
	}
	if errors.Is(err, auth.ErrBlocked) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Аккаунт заблокирован"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	ratelimit.ClearLoginFailures(r.Context(), s.db, email)
	sid, err := s.sessions.Create(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	s.setSessionCookie(w, sid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(session.CookieName); err == nil {
		_ = s.sessions.Revoke(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.opts.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type passwordResetBody struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (s *Server) postPasswordReset(w http.ResponseWriter, r *http.Request) {
	var body passwordResetBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	if body.Token != "" {
		if err := s.auth.ResetPassword(r.Context(), body.Token, body.Password); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
		return
	}
	_ = s.auth.RequestPasswordReset(r.Context(), body.Email)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Если адрес зарегистрирован, мы отправили письмо со ссылкой.",
	})
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	sess := s.currentSession(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var used, trash, reserved, limit uint64
	_ = s.db.QueryRowContext(r.Context(), `
		SELECT used_bytes, trash_bytes, reserved_bytes, quota_bytes FROM users WHERE id = ?`, sess.UserID).
		Scan(&used, &trash, &reserved, &limit)
	feedMeta, _ := feed.ForOwner(r.Context(), s.db, sess.UserID)
	out := map[string]any{
		"email": sess.Email, "email_verified": sess.Verified,
		"quota_limit": limit, "quota_used": used, "quota_trash": trash, "quota_reserved": reserved,
	}
	if feedMeta != nil {
		out["handle"] = feedMeta.Handle
		out["feed_id"] = feedMeta.PublicID
		out["feed_visibility"] = feedMeta.Visibility
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) currentSession(r *http.Request) *session.Data {
	c, err := r.Cookie(session.CookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	sess, err := s.sessions.Load(r.Context(), c.Value)
	if err != nil {
		return nil
	}
	return sess
}

func (s *Server) setSessionCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(session.AbsoluteTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.opts.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
