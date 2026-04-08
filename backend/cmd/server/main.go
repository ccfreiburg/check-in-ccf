package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ccf/check-in/backend/internal/auth"
	"github.com/ccf/check-in/backend/internal/ct"
	"github.com/ccf/check-in/backend/internal/ctsync"
	localdb "github.com/ccf/check-in/backend/internal/db"
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

	dbPath := getEnv("DB_PATH", "checkin.db")
	database, err := localdb.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "path", dbPath, "err", err)
		os.Exit(1)
	}

	// Parse configured child group IDs from CT_CHILD_GROUP_IDS (comma-separated).
	var syncGroups []ctsync.GroupConfig
	for _, raw := range strings.Split(getEnv("CT_CHILD_GROUP_IDS", ""), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		gid, err := strconv.Atoi(raw)
		if err != nil {
			slog.Error("invalid CT_CHILD_GROUP_IDS entry", "value", raw)
			os.Exit(1)
		}
		name := raw // fallback: use ID as name
		if g, err := ctClient.GetGroup(gid); err == nil && g.Name != "" {
			name = g.Name
		}
		syncGroups = append(syncGroups, ctsync.GroupConfig{ID: gid, Name: name})
	}

	syncSvc := ctsync.New(ctClient, database, syncGroups)

	// Auto-sync on startup if data is stale (> 12 h old or never synced).
	if syncSvc.IsStale() {
		go func() {
			slog.Info("startup CT sync: data is stale, syncing in background")
			if err := syncSvc.Run(context.Background()); err != nil {
				slog.Error("startup CT sync failed", "err", err)
			}
		}()
	} else {
		slog.Info("startup CT sync: data is fresh", "lastSync", syncSvc.LastSync().Format("2006-01-02 15:04"))
	}

	jwtSecret := []byte(mustEnv("JWT_SECRET"))
	frontendBase := mustEnv("FRONTEND_BASE_URL")

	h := handlers.New(ctClient, database, syncSvc, jwtSecret, frontendBase)

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
		// Check-in management (2-step flow)
		r.Get("/api/admin/groups", h.ListGroups)
		r.Get("/api/admin/parents", h.ListParents)
		r.Get("/api/admin/parents/{id}", h.GetParentDetailByParentID)
		r.Get("/api/admin/checkins", h.ListCheckins)
		r.Post("/api/admin/checkins/{id}/confirm", h.ConfirmTagHandout)
		r.Post("/api/admin/checkins/{id}/checkin", h.CheckInAtGroup)
		r.Post("/api/admin/sync", h.SyncCT)
	})

	// Super-admin routes (require super_admin token)
	r.Group(func(r chi.Router) {
		r.Use(auth.SuperAdminMiddleware(jwtSecret))
		r.Post("/api/admin/checkins/{id}/set-status", h.SetCheckInStatus)
	})

	// Parent-facing routes (require parent token embedded in URL path)
	r.Get("/api/parent/{token}", h.GetParentCheckinPage)
	r.Get("/api/parent/{token}/qr", h.GetParentQR)
	r.Post("/api/parent/{token}/register/{childId}", h.RegisterChild)

	// Admin auth — exchange a known admin password for a short-lived JWT
	r.Post("/api/auth/admin", h.AdminLogin)

	port := getEnv("BACKEND_API_PORT", "8080")
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
