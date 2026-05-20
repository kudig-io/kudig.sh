// Package web provides the HTTP API server for kudig.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/client-go/kubernetes"
)

// Server holds the HTTP server state.
type Server struct {
	clientset *kubernetes.Clientset
	router    *mux.Router
	port      int
}

// NewServer creates a new Server with the given Kubernetes clientset.
func NewServer(clientset *kubernetes.Clientset) *Server {
	s := &Server{
		clientset: clientset,
		router:    mux.NewRouter(),
	}
	s.SetupRoutes()
	return s
}

// SetupRoutes registers all REST API routes.
func (s *Server) SetupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.enableCORS)

	// Clusters
	api.HandleFunc("/clusters", s.handleListClusters).Methods("GET")
	api.HandleFunc("/clusters/{cluster}", s.handleGetCluster).Methods("GET")
	api.HandleFunc("/clusters/{cluster}/status", s.handleGetClusterStatus).Methods("GET")

	// Pods
	api.HandleFunc("/namespaces/{namespace}/pods", s.handleListPods).Methods("GET")
	api.HandleFunc("/pods", s.handleListAllPods).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/pods/{pod}", s.handleGetPod).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/pods/{pod}/logs", s.handleGetPodLogs).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/pods/{pod}", s.handleDeletePod).Methods("DELETE")

	// Nodes
	api.HandleFunc("/nodes", s.handleListNodes).Methods("GET")
	api.HandleFunc("/nodes/{node}", s.handleGetNode).Methods("GET")
	api.HandleFunc("/nodes/metrics", s.handleGetNodeMetrics).Methods("GET")

	// Deployments
	api.HandleFunc("/namespaces/{namespace}/deployments", s.handleListDeployments).Methods("GET")
	api.HandleFunc("/deployments", s.handleListAllDeployments).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/deployments/{deployment}", s.handleGetDeployment).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/deployments/{deployment}/scale", s.handleScaleDeployment).Methods("POST")
	api.HandleFunc("/namespaces/{namespace}/deployments/{deployment}/restart", s.handleRestartDeployment).Methods("POST")
	api.HandleFunc("/namespaces/{namespace}/deployments/{deployment}/pods", s.handleGetDeploymentPods).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/deployments/{deployment}/status", s.handleGetDeploymentStatus).Methods("GET")

	// Services
	api.HandleFunc("/namespaces/{namespace}/services", s.handleListServices).Methods("GET")
	api.HandleFunc("/services", s.handleListAllServices).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/services/{service}", s.handleGetService).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/services/{service}/endpoints", s.handleGetServiceEndpoints).Methods("GET")

	// Events
	api.HandleFunc("/namespaces/{namespace}/events", s.handleGetEvents).Methods("GET")
	api.HandleFunc("/events", s.handleGetEvents).Methods("GET")

	// Monitoring
	api.HandleFunc("/monitor/status", s.handleGetMonitorStatus).Methods("GET")
	api.HandleFunc("/monitor/alerts", s.handleGetMonitorAlerts).Methods("GET")
	api.HandleFunc("/monitor/metrics", s.handleGetMetricsHistory).Methods("GET")

	// SPA frontend fallback
	spaDir := os.Getenv("KUDIG_WEB_DIR")
	if spaDir == "" {
		spaDir = "./web/build"
	}
	s.router.PathPrefix("/").HandlerFunc(s.serveSPA(spaDir))
}

// Start begins listening on the specified port with graceful shutdown.
func (s *Server) Start(port int) error {
	s.port = port
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("web server listening on :%d", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down web server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// enableCORS adds cross-origin headers to every response.
func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// serveSPA serves the React frontend, falling back to index.html for client routes.
func (s *Server) serveSPA(dir string) http.HandlerFunc {
	fs := http.Dir(dir)
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		full := filepath.Join(dir, path)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		http.FileServer(fs).ServeHTTP(w, r)
	}
}

// respondJSON writes v as JSON with the given status code.
func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("respondJSON encode error: %v", err)
	}
}

// respondError writes a JSON error message with the given status code.
func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
