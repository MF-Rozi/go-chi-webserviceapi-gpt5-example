package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"dev.mfr/go-chi-sqlc-auth/internal/auth"
	"dev.mfr/go-chi-sqlc-auth/internal/config"
	"dev.mfr/go-chi-sqlc-auth/internal/database"
	"dev.mfr/go-chi-sqlc-auth/internal/handlers"
	mw "dev.mfr/go-chi-sqlc-auth/internal/middleware"
	"dev.mfr/go-chi-sqlc-auth/internal/models"
	"github.com/go-chi/chi/v5"
	middleware2 "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := database.NewPool(cfg.DB)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	issuer := auth.JWTIssuer{Secret: []byte(cfg.JWT.Secret), Expires: time.Duration(cfg.JWT.ExpiresInHours) * time.Hour}

	// Seed admin and demo user if not exists
	if err := seedUsers(pool); err != nil {
		log.Printf("seed warning: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware2.Logger)
	r.Use(middleware2.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	authH := handlers.NewAuthHandler(pool, issuer)
	r.Mount("/auth", authH.Routes())

	usersH := handlers.NewUsersHandler(pool)
	// protect users routes
	r.Group(func(pr chi.Router) {
		pr.Use(mw.JWT(issuer))
		pr.Mount("/users", usersH.Routes())
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func seedUsers(pool *pgxpool.Pool) error {
	ctx := context.Background()
	// admin
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(1) FROM users WHERE email=$1", "admin@example.com").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		pw, _ := auth.HashPassword("AdminPass123!")
		_, err := pool.Exec(ctx, `INSERT INTO users (username, email, password_hash, first_name, last_name, role) VALUES ($1,$2,$3,$4,$5,$6)`,
			"admin", "admin@example.com", pw, "Admin", "User", models.RoleAdmin)
		if err != nil {
			return err
		}
	}
	// demo
	if err := pool.QueryRow(ctx, "SELECT COUNT(1) FROM users WHERE email=$1", "demo@example.com").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		pw, _ := auth.HashPassword("DemoPass123!")
		_, err := pool.Exec(ctx, `INSERT INTO users (username, email, password_hash, first_name, last_name, role) VALUES ($1,$2,$3,$4,$5,$6)`,
			"demo", "demo@example.com", pw, "Demo", "User", models.RoleUser)
		if err != nil {
			return err
		}
	}
	return nil
}
