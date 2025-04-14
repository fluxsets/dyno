package http

import (
	"context"
	"github.com/fluxsets/dyno"
	"gocloud.dev/server"
	"log/slog"
	"net/http"
)

func NewRouter() *http.ServeMux {
	return http.NewServeMux()
}

func NewServer(addr string, h http.HandlerFunc) *Server {
	hs := server.New(h, &server.Options{
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

func (s *Server) Name() string {
	return "http"
}

func (s *Server) Init(do dyno.Dyno) error {
	s.logger = do.Logger("deployment", s.Name())
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

var _ dyno.ServerLike = (*Server)(nil)
