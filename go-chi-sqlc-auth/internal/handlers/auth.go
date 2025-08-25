package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"dev.mfr/go-chi-sqlc-auth/internal/auth"
	"dev.mfr/go-chi-sqlc-auth/internal/httpx"
	"dev.mfr/go-chi-sqlc-auth/internal/middleware"
	"dev.mfr/go-chi-sqlc-auth/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	Pool   *pgxpool.Pool
	Issuer auth.JWTIssuer
}

func NewAuthHandler(pool *pgxpool.Pool, issuer auth.JWTIssuer) *AuthHandler {
	return &AuthHandler{Pool: pool, Issuer: issuer}
}

func (h *AuthHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Group(func(pr chi.Router) {
		pr.Use(middleware.JWT(h.Issuer))
		pr.Get("/me", h.Me)
	})
	return r
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		httpx.Error(w, http.StatusBadRequest, "password required")
		return
	}
	ph, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	role := models.RoleUser
	if req.Role != nil {
		role = *req.Role
	}

	row := h.Pool.QueryRow(r.Context(),
		`INSERT INTO users (username, email, password_hash, first_name, last_name, phone_number, address, role)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
         RETURNING id, created_at, updated_at`,
		req.Username, req.Email, ph, req.FirstName, req.LastName, req.PhoneNumber, req.Address, role,
	)
	var id string
	var createdAt, updatedAt time.Time
	if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
		httpx.Error(w, http.StatusBadRequest, parsePGError(err))
		return
	}
	token, err := h.Issuer.Issue(id, role)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	httpx.JSON(w, http.StatusCreated, models.AuthResponse{Token: token, Role: role})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var (
		id   string
		role string
		hash string
	)
	err := h.Pool.QueryRow(r.Context(), "SELECT id, role, password_hash FROM users WHERE email=$1", req.Email).Scan(&id, &role, &hash)
	if err == pgx.ErrNoRows {
		httpx.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "query error")
		return
	}
	if err := auth.CheckPassword(hash, req.Password); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := h.Issuer.Issue(id, models.Role(role))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	httpx.JSON(w, http.StatusOK, models.AuthResponse{Token: token, Role: models.Role(role)})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, _ := r.Context().Value(middleware.CtxUserID).(string)
	if _, err := uuid.Parse(uid); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var resp struct {
		ID    string      `json:"id"`
		Email string      `json:"email"`
		Role  models.Role `json:"role"`
	}
	err := h.Pool.QueryRow(r.Context(), "SELECT id, email, role FROM users WHERE id=$1", uid).Scan(&resp.ID, &resp.Email, &resp.Role)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "user not found")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func decodeJSON(r *http.Request, v interface{}) error { return json.NewDecoder(r.Body).Decode(v) }

// parsePGError trims common pgx errors to a simple message
func parsePGError(err error) string {
	return err.Error()
}
