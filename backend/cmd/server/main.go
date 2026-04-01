package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/ccf/check-in/backend/internal/auth"
	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/ccf/check-in/backend/internal/handlers"
	"github.com/ccf/check-in/backend/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file found, using environment variables")
	}

	logger.Init()

	ctClient := ct.NewClient(
		mustEnv("CT_BASE_URL"),
		mustEnv("CT_API_TOKEN"),
	)

	jwtSecret := []byte(mustEnv("JWT_SECRET"))
	frontendBase := mustEnv("FRONTEND_BASE_URL")

	h := handlers.New(ctClient, jwtSecret, frontendBase)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendBase},
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Admin routes (require admin token via Authorization header)
	r.Group(func(r chi.Router) {
		r.Use(auth.AdminMiddleware(jwtSecret))
		r.Get("/api/admin/children", h.ListChildren)
		r.Get("/api/admin/children/{id}/parent", h.GetParentDetail)
		r.Post("/api/admin/children/{id}/qr", h.GenerateQR)
	})

	// Parent-facing routes (require parent token embedded in URL path)
	r.Get("/api/parent/{token}", h.GetParentCheckinPage)
	r.Post("/api/parent/{token}/checkin/{childId}", h.CheckIn)
	r.Post("/api/parent/{token}/checkout/{childId}", h.CheckOut)

	// Admin auth — exchange a known admin password for a short-lived JWT
	r.Post("/api/auth/admin", h.AdminLogin)

	port := getEnv("PORT", "8080")
	slog.Info("server listening", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
