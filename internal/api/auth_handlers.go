package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dsefremov/kadr/internal/auth"
	"github.com/dsefremov/kadr/internal/session"
	"github.com/dsefremov/kadr/internal/units"
)

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", s.postRegister)
	mux.HandleFunc("POST /api/auth/verify", s.postVerify)
	mux.HandleFunc("POST /api/auth/verify/resend", s.postResend)
	mux.HandleFunc("POST /api/auth/login", s.postLogin)
	mux.HandleFunc("POST /api/auth/logout", s.postLogout)
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
	_ = s.auth.ResendVerify(r.Context(), body.Email)
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
	userID, err := s.auth.Login(r.Context(), body.Email, body.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
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

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	sess := s.currentSession(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":            sess.Email,
		"email_verified":   sess.Verified,
		"quota_limit":      units.DefaultQuotaBytes,
		"quota_used":       0,
		"quota_trash":      0,
		"quota_reserved":   0,
	})
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
