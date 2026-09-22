package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	authhttp "github.com/simple-erp/backend/internal/auth/adapter/http"
	authpostgres "github.com/simple-erp/backend/internal/auth/adapter/postgres"
	"github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/auth/domain"
	masterhttp "github.com/simple-erp/backend/internal/masterdata/adapter/http"
	masterpostgres "github.com/simple-erp/backend/internal/masterdata/adapter/postgres"
	masterapp "github.com/simple-erp/backend/internal/masterdata/application"
	"github.com/simple-erp/backend/internal/platform/config"
	"github.com/simple-erp/backend/internal/platform/httpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}
	login := application.NewLoginService(authpostgres.LoginStore{Pool: pool}, domain.NewPasswordHasher(), []byte(cfg.SessionPepper))
	authHandler := authhttp.NewHandler(login)
	lifecycleService := application.NewLifecycleService(authpostgres.LifecycleStore{Pool: pool}, domain.NewPasswordHasher(), []byte(cfg.SessionPepper))
	lifecycle := authhttp.LifecycleHandler{Service: lifecycleService}
	sessionHandler := authhttp.SessionHandler{Service: application.NewSessionService(authpostgres.SessionStore{Pool: pool}, []byte(cfg.SessionPepper))}
	authMiddleware := authhttp.NewMiddleware(authpostgres.Authenticator{Pool: pool, Pepper: []byte(cfg.SessionPepper)}, cfg.TrustedOrigin)
	masterHandler := masterhttp.Handler{Service: masterapp.NewService(masterpostgres.Store{Pool: pool})}
	usersHandler := authhttp.UsersHandler{Service: application.NewUserAdminService(authpostgres.UserAdminStore{Pool: pool}), Lifecycle: lifecycleService}
	settingsHandler := masterhttp.SettingsHandler{Service: masterapp.NewSettingsService(masterpostgres.SettingsStore{Pool: pool})}
	api := chi.NewRouter()
	api.Handle("/auth/login", authHandler.Routes())
	api.Handle("/auth/activate-invitation", lifecycle.PublicRoutes())
	api.Handle("/auth/reset-password", lifecycle.PublicRoutes())
	api.Post("/auth/refresh", sessionHandler.Refresh)
	api.With(func(next http.Handler) http.Handler { return authMiddleware.RequireAuth(next) }).Post("/auth/logout", sessionHandler.Logout)
	api.Handle("/user-invitations", authMiddleware.RequireRole("ADMIN", lifecycle.AdminRoutes()))
	api.Handle("/user-invitations/*", authMiddleware.RequireRole("ADMIN", lifecycle.AdminRoutes()))
	api.Handle("/password-resets/*", authMiddleware.RequireRole("ADMIN", lifecycle.AdminRoutes()))
	api.Mount("/users", authMiddleware.RequireRole("ADMIN", usersHandler.Routes()))
	api.Mount("/master-data", authMiddleware.RequireRole("ADMIN", masterHandler.Routes()))
	api.Handle("/settings/organization", authMiddleware.RequireRole("ADMIN", settingsHandler))
	s := &http.Server{
		Addr: cfg.HTTPAddress, Handler: httpserver.NewRouter(api),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	log.Printf("api listening on %s", cfg.HTTPAddress)
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
