package middleware

import (
	"context"
	"net/http"
	"strings"

	"dev.mfr/go-chi-sqlc-auth/internal/auth"
	"dev.mfr/go-chi-sqlc-auth/internal/models"
)

type ctxKey string

const (
	CtxUserID ctxKey = "uid"
	CtxRole   ctxKey = "role"
)

func JWT(issuer auth.JWTIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ah := r.Header.Get("Authorization")
			if ah == "" || !strings.HasPrefix(ah, "Bearer ") {
				http.Error(w, "missing or invalid auth header", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(ah, "Bearer ")
			claims, err := issuer.Parse(token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRoles(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := map[models.Role]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(CtxRole).(models.Role)
			if _, ok := allowed[role]; !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
