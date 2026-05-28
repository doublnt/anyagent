package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/anyagent/anyagent/backend/internal/db"
	"github.com/anyagent/anyagent/backend/internal/handler"
	"github.com/anyagent/anyagent/backend/internal/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	if err := db.Init(); err != nil {
		log.Printf("Warning: Database not available: %v", err)
		log.Println("Running without database (memory/trace features disabled)")
	} else {
		defer db.Close()
		log.Println("Database connected")
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Auth routes (public)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)
	mux.HandleFunc("POST /api/v1/auth/register", handler.Register)

	// Auth routes (protected)
	mux.Handle("GET /api/v1/auth/me", middleware.Auth(http.HandlerFunc(handler.GetCurrentUser)))

	// Agent Registry (public read, protected publish with scope=manage)
	mux.HandleFunc("GET /api/v1/agents", handler.ListAgents)
	mux.HandleFunc("GET /api/v1/agents/{name}", handler.GetAgent)
	mux.HandleFunc("GET /api/v1/agents/{name}/versions", handler.ListAgentVersions)
	mux.Handle("POST /api/v1/agents",
		middleware.Chain(
			middleware.Auth(http.HandlerFunc(handler.PublishAgent)),
			middleware.RequireScope("manage"),
		))
	mux.HandleFunc("GET /api/v1/agents/{name}/{version}/download", handler.DownloadAgent)

	// Hosted agent execution (internal, Use scope)
	mux.Handle("POST /api/v1/agents/{name}/run",
		middleware.Chain(
			middleware.Auth(http.HandlerFunc(handler.RunAgent)),
			middleware.RequireScope("use"),
		))

	// Entitlements / subscriptions
	mux.HandleFunc("GET /api/v1/entitlements/check", handler.CheckEntitlement)
	mux.Handle("POST /api/v1/entitlements",
		middleware.Chain(
			middleware.Auth(http.HandlerFunc(handler.Subscribe)),
			middleware.RequireScope("read"),
		))
	mux.Handle("POST /api/v1/entitlements/{id}/token",
		middleware.Chain(
			middleware.Auth(http.HandlerFunc(handler.MintToken)),
			middleware.RequireScope("use"),
		))

	// Usage metering
	mux.Handle("POST /api/v1/usage",
		middleware.Auth(http.HandlerFunc(handler.RecordUsage)))

	// Subscription view (replaces stub)
	mux.HandleFunc("GET /api/v1/subscription", handler.GetSubscription)
	mux.HandleFunc("GET /api/v1/license/verify", handler.VerifyLicense)

	// Projects (protected)
	mux.Handle("GET /api/v1/projects", middleware.Auth(http.HandlerFunc(handler.ListProjects)))
	mux.Handle("POST /api/v1/projects", middleware.Auth(http.HandlerFunc(handler.CreateProject)))
	mux.Handle("GET /api/v1/projects/{id}", middleware.Auth(http.HandlerFunc(handler.GetProject)))

	// Memory (protected)
	mux.Handle("GET /api/v1/projects/{id}/memories", middleware.Auth(http.HandlerFunc(handler.ListMemories)))
	mux.Handle("POST /api/v1/projects/{id}/memories", middleware.Auth(http.HandlerFunc(handler.CreateMemory)))
	mux.Handle("POST /api/v1/projects/{id}/memories/search", middleware.Auth(http.HandlerFunc(handler.SearchMemories)))

	// Traces (protected)
	mux.Handle("GET /api/v1/projects/{id}/traces", middleware.Auth(http.HandlerFunc(handler.ListTraces)))
	mux.Handle("POST /api/v1/projects/{id}/traces", middleware.Auth(http.HandlerFunc(handler.CreateTrace)))
	mux.Handle("GET /api/v1/traces/{id}", middleware.Auth(http.HandlerFunc(handler.GetTrace)))
	mux.Handle("POST /api/v1/traces/{id}/spans", middleware.Auth(http.HandlerFunc(handler.AddTraceSpan)))
	mux.Handle("POST /api/v1/traces/{id}/complete", middleware.Auth(http.HandlerFunc(handler.CompleteTrace)))

	// Apply global middleware
	h := middleware.Chain(mux, middleware.Logger, middleware.CORS, middleware.Recovery)

	log.Printf("AnyAgent API server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, h); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
