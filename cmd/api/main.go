package main

import (
    "net/http"
    "os"
    "time"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "github.com/rs/cors"
    "isp-saas.com/platform/internal/handlers"
    "isp-saas.com/platform/internal/middleware"
    "isp-saas.com/platform/pkg/database"
    "isp-saas.com/platform/pkg/logger"
)

func main() {
    godotenv.Load()

    log := logger.New()
    log.Info("Starting ISP SaaS Platform API v1.1.0...")

    db, err := database.Connect()
    if err != nil {
        log.Fatal("Failed to connect to database", "error", err)
    }
    defer db.Close()
    log.Info("Database connected successfully")

    if err := db.RunMigrations("./migrations"); err != nil {
        log.Fatal("Failed to run migrations", "error", err)
    }
    log.Info("Migrations completed")

    h := handlers.New(db, log)
    r := mux.NewRouter()

    // ============== PUBLIC ROUTES ==============
    r.HandleFunc("/api/health", h.HealthCheck).Methods("GET")
    r.HandleFunc("/api/auth/login", h.Login).Methods("POST")
    r.HandleFunc("/api/auth/register", h.Register).Methods("POST")
    r.HandleFunc("/api/plans", h.GetPlans).Methods("GET")
    r.HandleFunc("/api/plans/{id}", h.GetPlan).Methods("GET")

    // Agent routes (public for agents)
    r.HandleFunc("/api/licenses/validate", h.ValidateLicense).Methods("POST")
    r.HandleFunc("/api/telemetry", h.SubmitTelemetry).Methods("POST")
    r.HandleFunc("/api/logs", h.CreateSystemLog).Methods("POST")
    r.HandleFunc("/api/sites/report", h.ReportCachedSite).Methods("POST")

    // ============== PROTECTED ROUTES ==============
    api := r.PathPrefix("/api").Subrouter()
    api.Use(middleware.AuthMiddleware)

    // Auth
    api.HandleFunc("/auth/refresh", h.RefreshToken).Methods("POST")

    // Dashboard (role-aware)
    api.HandleFunc("/dashboard/stats", h.GetDashboardStats).Methods("GET")

    // Top Sites & Apps (NEW)
    api.HandleFunc("/sites/top", h.GetTopSites).Methods("GET")
    api.HandleFunc("/apps/top", h.GetTopApps).Methods("GET")
    api.HandleFunc("/apps/categories", h.GetAppCategories).Methods("GET")

    // Users
    api.HandleFunc("/users", h.GetUsers).Methods("GET")
    api.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
    api.HandleFunc("/users/{id}", h.UpdateUser).Methods("PUT")
    api.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE")

    // Distributors
    api.HandleFunc("/distributors", h.GetDistributors).Methods("GET")
    api.HandleFunc("/distributors", h.CreateDistributor).Methods("POST")
    api.HandleFunc("/distributors/{id}", h.GetDistributor).Methods("GET")
    api.HandleFunc("/distributors/{id}", h.UpdateDistributor).Methods("PUT")
    api.HandleFunc("/distributors/{id}/isps", h.GetDistributorISPs).Methods("GET")

    // ISPs
    api.HandleFunc("/isps", h.GetISPs).Methods("GET")
    api.HandleFunc("/isps", h.CreateISP).Methods("POST")
    api.HandleFunc("/isps/{id}", h.GetISP).Methods("GET")
    api.HandleFunc("/isps/{id}", h.UpdateISP).Methods("PUT")
    api.HandleFunc("/isps/{id}", h.DeleteISP).Methods("DELETE")
    api.HandleFunc("/isps/{id}/suspend", h.SuspendISP).Methods("POST")
    api.HandleFunc("/isps/{id}/activate", h.ActivateISP).Methods("POST")
    api.HandleFunc("/isps/{id}/telemetry", h.GetISPTelemetry).Methods("GET")
    api.HandleFunc("/isps/{id}/dashboard", h.GetISPDashboard).Methods("GET")

    // Licenses
    api.HandleFunc("/licenses", h.GetLicenses).Methods("GET")
    api.HandleFunc("/licenses", h.CreateLicense).Methods("POST")
    api.HandleFunc("/licenses/{id}", h.GetLicense).Methods("GET")
    api.HandleFunc("/licenses/{id}/revoke", h.RevokeLicense).Methods("POST")

    // Telemetry
    api.HandleFunc("/telemetry/stats", h.GetTelemetryStats).Methods("GET")

    // Billing
    api.HandleFunc("/invoices", h.GetInvoices).Methods("GET")
    api.HandleFunc("/invoices", h.CreateInvoice).Methods("POST")
    api.HandleFunc("/invoices/{id}/pay", h.MarkInvoicePaid).Methods("POST")
    api.HandleFunc("/invoices/check-overdue", h.CheckOverdueInvoices).Methods("POST")

    // System Logs
    api.HandleFunc("/logs", h.GetSystemLogs).Methods("GET")
    api.HandleFunc("/logs/stats", h.GetLogStats).Methods("GET")
    api.HandleFunc("/logs/cleanup", h.DeleteOldLogs).Methods("DELETE")

    // Settings
    api.HandleFunc("/settings", h.GetSettings).Methods("GET")
    api.HandleFunc("/settings/get", h.GetSetting).Methods("GET")
    api.HandleFunc("/settings/update", h.UpdateSetting).Methods("PUT")

    // CORS
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"*"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type"},
        AllowCredentials: true,
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    srv := &http.Server{
        Handler:      c.Handler(r),
        Addr:         ":" + port,
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Info("Server starting", "port", port)
    log.Info("New endpoints: /api/sites/top, /api/apps/top, /api/isps/{id}/dashboard")

    if err := srv.ListenAndServe(); err != nil {
        log.Fatal("Server failed", "error", err)
    }
}
