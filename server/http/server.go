package http

import (
	"context"
	"github.com/fluxsets/orbit"
	"gocloud.dev/server"
	"gocloud.dev/server/health"
	"log/slog"
	"net/http"
)

func NewRouter() *http.ServeMux {
	return http.NewServeMux()
}

func NewServer(addr string, h http.HandlerFunc, healthChecks []health.Checker, logger *slog.Logger) *Server {
	hs := server.New(h, &server.Options{
		HealthChecks: healthChecks,
		RequestLogger: NewRequestLogger(logger, func(err error) {
		}),
		Driver: server.NewDefaultDriver(),
	})
	return &Server{
		Server: hs,
		addr:   addr,
	}
}

type Server struct {
	*server.Server
	addr   string
	logger *slog.Logger
}

func (s *Server) CheckHealth() error {
	return nil
}

func (s *Server) Name() string {
	return "http"
}

func (s *Server) Init(ob orbit.Orbit) error {
	s.logger = ob.Logger("deployment", s.Name())
	return nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server, listening on " + s.addr)
	return s.Server.ListenAndServe(s.addr)
}

func (s *Server) Stop(ctx context.Context) {
	s.logger.Info("Stopping HTTP server")
	if err := s.Server.Shutdown(ctx); err != nil {
		s.logger.Warn("Error shutting down http server", "error", err)
	}
}

var _ orbit.ServerLike = (*Server)(nil)
