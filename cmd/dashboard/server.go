package main

import (
	"component-monitor/pkg/types"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Server struct {
	logger   *logrus.Logger
	config   *types.Config
	handlers *Handlers
	db       *gorm.DB
}

func NewServer(config *types.Config, db *gorm.DB) *Server {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

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
	router.HandleFunc("/api/components/{componentName}/outages", s.handlers.GetOutages).Methods("GET")
	router.HandleFunc("/api/components/{componentName}/outages", s.handlers.CreateOutage).Methods("POST")

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

func (s *Server) Start(addr string) error {
	handler := s.setupRoutes()
	s.logger.Infof("Starting dashboard server on %s", addr)
	return http.ListenAndServe(addr, handler)
}
