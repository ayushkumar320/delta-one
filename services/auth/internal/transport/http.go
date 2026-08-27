// Package transport exposes the auth service over HTTP.
package transport

import (
	"net/http"
	"time"

	"github.com/ayush/delta-one/services/auth/internal/domain"
	"github.com/ayush/delta-one/services/auth/internal/service"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
)

// Handler serves the auth HTTP API.
type Handler struct {
	auth *service.Auth
}

// NewHandler returns a Handler for the given service.
func NewHandler(auth *service.Auth) *Handler { return &Handler{auth: auth} }

// Routes returns the service's router with shared middleware applied.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth"})
	})
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.Handle("GET /auth/me", middleware.RequireUser(http.HandlerFunc(h.me)))

	mw := append(middleware.Default("auth"), middleware.Identity)
	return middleware.Chain(mux, mw...)
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	Role     string `json:"role,omitempty"`
}

type userView struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func viewUser(u domain.User) userView {
	return userView{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt}
}

type sessionView struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      userView  `json:"user"`
}

func viewSession(s service.Session) sessionView {
	return sessionView{Token: s.Token, ExpiresAt: s.ExpiresAt, User: viewUser(s.User)}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	session, err := h.auth.Register(r.Context(), in.Email, in.Password, in.Name, in.Role)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, viewSession(session))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	session, err := h.auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, viewSession(session))
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.Profile(r.Context(), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, viewUser(user))
}
