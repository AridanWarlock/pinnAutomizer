package http_server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/docs"
	http_middleware "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Server struct {
	mux         *http.ServeMux
	middlewares []http_middleware.Middleware

	cfg Config
	log zerolog.Logger
}

func New(
	cfg Config,
	log zerolog.Logger,
	middlewares ...http_middleware.Middleware,
) *Server {
	return &Server{
		mux:         http.NewServeMux(),
		middlewares: middlewares,

		cfg: cfg,
		log: log,
	}
}

func (s *Server) RegisterApiRouters(routers ...*ApiVersionRouter) {
	for _, router := range routers {
		prefix := "/api/" + string(router.apiVersion)

		s.mux.Handle(prefix+"/", http.StripPrefix(prefix, router))
	}
}

func (s *Server) RegisterSwagger() {
	s.mux.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(docs.SwaggerInfo.ReadDoc()))
	})

	s.mux.Handle(
		"/swagger/",
		httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		),
	)
}

func (s *Server) Run(ctx context.Context) error {
	mux := http_middleware.ChainMiddleware(s.mux, s.middlewares...)

	server := &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      mux,
		ReadTimeout:  s.cfg.Timeout,
		WriteTimeout: s.cfg.Timeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}

	ch := make(chan error)

	go func() {
		defer close(ch)

		s.log.Warn().Str("addr", s.cfg.Addr).Msg("start HTTP server")

		err := server.ListenAndServe()

		if !errors.Is(err, http.ErrServerClosed) {
			ch <- err
		}
	}()

	select {
	case err := <-ch:
		if err != nil {
			return fmt.Errorf("listen and serve HTTP: %w", err)
		}
	case <-ctx.Done():
		s.log.Warn().Msg("shutdown HTTP server...")

		shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		s.log.Warn().Msg("HTTP server shutdown gracefully")
	}

	return nil
}
