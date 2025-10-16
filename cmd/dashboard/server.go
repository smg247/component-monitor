package main

import (
	"net/http"
	"ship-status-dash/pkg/types"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Server represents the HTTP server for the dashboard API.
type Server struct {
	logger   *logrus.Logger
	config   *types.Config
	handlers *Handlers
	db       *gorm.DB
}

// NewServer creates a new Server instance with the provided configuration, database connection, and logger.
func NewServer(config *types.Config, db *gorm.DB, logger *logrus.Logger) *Server {
	handlers := NewHandlers(logger, config, db)

	return &Server{
		logger:   logger,
		config:   config,
		handlers: handlers,
		db:       db,
	}
}

func (s *Server) setupRoutes() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/health", s.handlers.Health).Methods("GET")
	router.HandleFunc("/api/components", s.handlers.GetComponents).Methods("GET")
	router.HandleFunc("/api/components/{componentName}/{subComponentName}/outages/{outageId:[0-9]+}", s.handlers.UpdateOutage).Methods("PATCH")
	router.HandleFunc("/api/components/{componentName}/{subComponentName}/outages/{outageId:[0-9]+}", s.handlers.DeleteOutage).Methods("DELETE")
	router.HandleFunc("/api/components/{componentName}/{subComponentName}/outages", s.handlers.CreateOutage).Methods("POST")
	router.HandleFunc("/api/components/{componentName}/{subComponentName}/outages", s.handlers.GetSubComponentOutages).Methods("GET")
	router.HandleFunc("/api/components/{componentName}/outages", s.handlers.GetOutages).Methods("GET")

	router.Use(s.loggingMiddleware)

	return router
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		s.logger.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start),
		}).Info("Request processed")
	})
}

// Start begins listening for HTTP requests on the specified address.
func (s *Server) Start(addr string) error {
	handler := s.setupRoutes()
	s.logger.Infof("Starting dashboard server on %s", addr)
	return http.ListenAndServe(addr, handler)
}
