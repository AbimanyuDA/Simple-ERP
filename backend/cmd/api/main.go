package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	authhttp "github.com/simple-erp/backend/internal/auth/adapter/http"
	authpostgres "github.com/simple-erp/backend/internal/auth/adapter/postgres"
	"github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/auth/domain"
	financehttp "github.com/simple-erp/backend/internal/finance/adapter/http"
	financepostgres "github.com/simple-erp/backend/internal/finance/adapter/postgres"
	financeapp "github.com/simple-erp/backend/internal/finance/application"
	inventoryhttp "github.com/simple-erp/backend/internal/inventory/adapter/http"
	inventorypostgres "github.com/simple-erp/backend/internal/inventory/adapter/postgres"
	inventoryapp "github.com/simple-erp/backend/internal/inventory/application"
	masterhttp "github.com/simple-erp/backend/internal/masterdata/adapter/http"
	masterpostgres "github.com/simple-erp/backend/internal/masterdata/adapter/postgres"
	masterapp "github.com/simple-erp/backend/internal/masterdata/application"
	"github.com/simple-erp/backend/internal/platform/config"
	"github.com/simple-erp/backend/internal/platform/httpserver"
	"github.com/simple-erp/backend/internal/platform/idempotency"
	purchasehttp "github.com/simple-erp/backend/internal/purchasing/adapter/http"
	purchasepostgres "github.com/simple-erp/backend/internal/purchasing/adapter/postgres"
	purchaseapp "github.com/simple-erp/backend/internal/purchasing/application"
	reportinghttp "github.com/simple-erp/backend/internal/reporting/adapter/http"
	reportingpostgres "github.com/simple-erp/backend/internal/reporting/adapter/postgres"
	reportingapp "github.com/simple-erp/backend/internal/reporting/application"
	saleshttp "github.com/simple-erp/backend/internal/sales/adapter/http"
	salespostgres "github.com/simple-erp/backend/internal/sales/adapter/postgres"
	salesapp "github.com/simple-erp/backend/internal/sales/application"
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
	purchaseHandler := purchasehttp.Handler{Service: purchaseapp.NewService(purchasepostgres.Store{Pool: pool})}
	salesHandler := saleshttp.Handler{Service: salesapp.NewService(salespostgres.Store{Pool: pool})}
	inventoryHandler := inventoryhttp.Handler{Service: inventoryapp.NewService(inventorypostgres.Store{Pool: pool})}
	financeHandler := financehttp.Handler{Service: financeapp.NewService(financepostgres.Store{Pool: pool})}
	reportingHandler := reportinghttp.Handler{Service: reportingapp.NewService(reportingpostgres.Store{Pool: pool})}
	api := chi.NewRouter()
	api.Handle("/auth/login", authHandler.Routes())
	api.Handle("/auth/activate-invitation", lifecycle.PublicRoutes())
	api.Handle("/auth/reset-password", lifecycle.PublicRoutes())
	api.Post("/auth/refresh", sessionHandler.Refresh)
	api.With(func(next http.Handler) http.Handler { return authMiddleware.RequireAuth(next) }).Post("/auth/logout", sessionHandler.Logout)
	api.With(authMiddleware.RequireAuth).Get("/auth/session", func(w http.ResponseWriter, r *http.Request) {
		principal, _ := application.PrincipalFrom(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"userId": principal.UserID, "email": principal.Email, "displayName": principal.DisplayName, "role": principal.Role})
	})
	api.Handle("/user-invitations", authMiddleware.RequireRole(domain.RoleSuperAdmin, lifecycle.AdminRoutes()))
	api.Handle("/user-invitations/*", authMiddleware.RequireRole(domain.RoleSuperAdmin, lifecycle.AdminRoutes()))
	api.Handle("/password-resets/*", authMiddleware.RequireRole(domain.RoleSuperAdmin, lifecycle.AdminRoutes()))
	api.Mount("/users", authMiddleware.RequireRole(domain.RoleSuperAdmin, usersHandler.Routes()))
	// Authenticated users may read master data for operational selection.
	// Mutations require Supervisor or Super Admin in the Master Data handler.
	api.Mount("/master-data", authMiddleware.RequireAuth(masterHandler.Routes()))
	api.Handle("/settings/organization", authMiddleware.RequireRole(domain.RoleSuperAdmin, settingsHandler))
	api.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Use(idempotency.Middleware(idempotency.PostgresStore{Pool: pool}))
		purchaseHandler.Register(r)
		// AP is shared: Staff drafts supplier invoices/payments, Supervisor verifies/posts.
		financeHandler.RegisterSupplier(r)
		// AR follows the same maker/checker split; handlers enforce action-level roles.
		financeHandler.RegisterSales(r)
		salesHandler.Register(r)
		inventoryHandler.Register(r)
		reportingHandler.RegisterDashboard(r)
		r.Group(func(admin chi.Router) {
			admin.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					principal, _ := application.PrincipalFrom(request.Context())
					if !domain.CanApprove(principal.Role) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusForbidden)
						_ = json.NewEncoder(w).Encode(map[string]string{"message": "Akses ditolak."})
						return
					}
					next.ServeHTTP(w, request)
				})
			})
			reportingHandler.RegisterAdmin(admin)
		})
	})
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
