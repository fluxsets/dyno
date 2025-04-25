package http

import (
	"context"
	"github.com/fluxsets/fleet"
	"gocloud.dev/server"
	"log/slog"
	"net/http"
)

func NewRouter() *http.ServeMux {
	return http.NewServeMux()
}

func NewServer(addr string, h http.Handler, healthCheck fleet.HealthCheckerRetriever, logger *slog.Logger) *Server {
	return &Server{
		addr:        addr,
		handler:     h,
		healthCheck: healthCheck,
		logger:      logger,
	}
}

type Server struct {
	*server.Server
	addr        string
	logger      *slog.Logger
	handler     http.Handler
	healthCheck fleet.HealthCheckerRetriever
}

func (s *Server) Name() string {
	return "http"
}

func (s *Server) Init(ft fleet.Fleet) error {
	s.logger = ft.Logger("component", s.Name())
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server, listening on " + s.addr)
	hs := server.New(s.handler, &server.Options{
		HealthChecks: s.healthCheck(),
		RequestLogger: NewRequestLogger(s.logger, func(err error) {
		}),
		Driver: server.NewDefaultDriver(),
	})
	s.Server = hs
	return s.Server.ListenAndServe(s.addr)
}

func (s *Server) Stop(ctx context.Context) {
	s.logger.Info("Stopping HTTP server")
	if err := s.Server.Shutdown(ctx); err != nil {
		s.logger.Warn("Error shutting down http server", "error", err)
	}
}

var _ fleet.Component = (*Server)(nil)
