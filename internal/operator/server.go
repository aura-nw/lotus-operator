package operator

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aura-nw/lotus-operator/config"
)

type Server struct {
	ctx    context.Context
	logger *slog.Logger
	info   config.ServerInfo

	srv *http.Server
}

func NewServer(ctx context.Context, logger *slog.Logger, info config.ServerInfo) (*Server, error) {
	s := &Server{
		logger: logger,
		info:   info,
	}

	mux := http.NewServeMux()

	s.registerHandlers(mux)

	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%s", info.HttpPort),
		Handler: mux,
	}
	return s, nil
}

func (s *Server) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`OK`)); err != nil {
			s.logger.Error("write data to client error", "err", err)
		}
	})
}

func (s *Server) Start() {
	if err := s.srv.ListenAndServe(); err != nil {
		panic(err)
	}
}

func (s *Server) Stop() {
	if err := s.srv.Shutdown(s.ctx); err != nil {
		s.logger.Error("shutdown http server error", "err", err)
	}
}
