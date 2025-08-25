package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"dev.mfr/go-chi-sqlc-auth/internal/httpx"
	"dev.mfr/go-chi-sqlc-auth/internal/middleware"
	"dev.mfr/go-chi-sqlc-auth/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	Pool *pgxpool.Pool
}

func NewUsersHandler(pool *pgxpool.Pool) *UsersHandler {
	return &UsersHandler{Pool: pool}
}

func (h *UsersHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/password", h.UpdatePassword)
	return r
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	// Only admin
	role, _ := r.Context().Value(middleware.CtxRole).(models.Role)
	if role != models.RoleAdmin {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	limit := 50
	offset := 0
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			limit = v
		}
	}
	if q := r.URL.Query().Get("offset"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			offset = v
		}
	}
	rows, err := h.Pool.Query(r.Context(), "SELECT id, username, email, first_name, last_name, phone_number, address, role, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	type user struct {
		ID          string      `json:"id"`
		Username    string      `json:"username"`
		Email       string      `json:"email"`
		FirstName   string      `json:"first_name"`
		LastName    string      `json:"last_name"`
		PhoneNumber *string     `json:"phone_number"`
		Address     *string     `json:"address"`
		Role        models.Role `json:"role"`
		CreatedAt   time.Time   `json:"created_at"`
		UpdatedAt   time.Time   `json:"updated_at"`
	}
	var resp []user
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName, &u.PhoneNumber, &u.Address, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp = append(resp, u)
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *UsersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Allow admin or self
	role, _ := r.Context().Value(middleware.CtxRole).(models.Role)
	uid, _ := r.Context().Value(middleware.CtxUserID).(string)
	if role != models.RoleAdmin && uid != id {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var u struct {
		ID          string      `json:"id"`
		Username    string      `json:"username"`
		Email       string      `json:"email"`
		FirstName   string      `json:"first_name"`
		LastName    string      `json:"last_name"`
		PhoneNumber *string     `json:"phone_number"`
		Address     *string     `json:"address"`
		Role        models.Role `json:"role"`
		CreatedAt   time.Time   `json:"created_at"`
		UpdatedAt   time.Time   `json:"updated_at"`
	}
	err := h.Pool.QueryRow(r.Context(), "SELECT id, username, email, first_name, last_name, phone_number, address, role, created_at, updated_at FROM users WHERE id=$1", id).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName, &u.PhoneNumber, &u.Address, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Allow admin or self
	role, _ := r.Context().Value(middleware.CtxRole).(models.Role)
	uid, _ := r.Context().Value(middleware.CtxUserID).(string)
	if role != models.RoleAdmin && uid != id {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Only admin can change role; for non-admin keep existing role
	var roleToSet *models.Role
	if role == models.RoleAdmin {
		roleToSet = req.Role
	}
	// Build update; if roleToSet is nil, keep current role
	var err error
	if roleToSet != nil {
		err = h.Pool.QueryRow(r.Context(), `UPDATE users SET username=$2, email=$3, first_name=$4, last_name=$5, phone_number=$6, address=$7, role=$8, updated_at=now() WHERE id=$1 RETURNING id`, id, req.Username, req.Email, req.FirstName, req.LastName, req.PhoneNumber, req.Address, *roleToSet).Scan(&id)
	} else {
		err = h.Pool.QueryRow(r.Context(), `UPDATE users SET username=$2, email=$3, first_name=$4, last_name=$5, phone_number=$6, address=$7, updated_at=now() WHERE id=$1 RETURNING id`, id, req.Username, req.Email, req.FirstName, req.LastName, req.PhoneNumber, req.Address).Scan(&id)
	}
	if err == pgx.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"id": id})
}

func (h *UsersHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Allow admin or self
	role, _ := r.Context().Value(middleware.CtxRole).(models.Role)
	uid, _ := r.Context().Value(middleware.CtxUserID).(string)
	if role != models.RoleAdmin && uid != id {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var req models.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "hash error")
		return
	}
	err = h.Pool.QueryRow(r.Context(), `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1 RETURNING id`, id, string(hash)).Scan(&id)
	if err == pgx.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"id": id})
}

func (h *UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Only admin
	role, _ := r.Context().Value(middleware.CtxRole).(models.Role)
	if role != models.RoleAdmin {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	id := chi.URLParam(r, "id")
	ct, err := h.Pool.Exec(r.Context(), "DELETE FROM users WHERE id=$1", id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ct.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": 1})
}
