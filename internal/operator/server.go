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
		w.Write([]byte(`OK`))
	})
}

func (s *Server) Start() {
	s.srv.ListenAndServe()
}

func (s *Server) Stop() {
	s.srv.Shutdown(s.ctx)
}
