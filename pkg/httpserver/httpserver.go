package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type Config struct {
	Port        string        `env:"HTTP_PORT" env-required:"true"`
	Timeout     time.Duration `env:"HTTP_TIMEOUT" env-required:"true"`
	IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT" env-required:"true"`
}

type Server struct {
	server *http.Server

	log zerolog.Logger
}

func New(handler http.Handler, c Config, log zerolog.Logger) *Server {
	server := &http.Server{
		Addr:         fmt.Sprint("127.0.0.1:$s", c.Port),
		Handler:      handler,
		ReadTimeout:  c.Timeout,
		WriteTimeout: c.Timeout,
		IdleTimeout:  c.IdleTimeout,
	}

	return &Server{
		server: server,

		log: log,
	}
}

func (s *Server) ListenAndServer() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) {
	_ = s.server.Shutdown(ctx)
}
