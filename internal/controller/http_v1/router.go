package http_v1

import (
	"net/http"
	"pinnAutomizer/internal/middleware/cors"
	"pinnAutomizer/internal/usecases/auth/login"
	"pinnAutomizer/internal/usecases/auth/logout"
	"pinnAutomizer/internal/usecases/auth/me"
	"pinnAutomizer/internal/usecases/auth/refresh"
	"pinnAutomizer/internal/usecases/auth/register"
	"pinnAutomizer/internal/usecases/task/create_task"
	"pinnAutomizer/internal/usecases/task/get_tasks"
	"pinnAutomizer/internal/usecases/task/solve_task"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type AuthMiddleware interface {
	Authenticate(http.Handler) http.Handler
}

type V1Handler func(w http.ResponseWriter, r *http.Request, log zerolog.Logger)

func Router(authMiddleware AuthMiddleware, log zerolog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.Recoverer,
		cors.NewChiCORS(cors.DefaultCORSConfig()),
		authMiddleware.Authenticate,
	)

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			//auth
			r.Post("/auth/login", login.HttpV1Handler(log))
			r.Post("/auth/logout", logout.HttpV1Handler(log))
			r.Get("/auth/me", me.HttpV1Handler(log))
			r.Post("/auth/refresh", refresh.HttpV1Handler(log))
			r.Post("/auth/register", register.HttpV1Handler(log))

			//tasks
			r.Post("/tasks", create_task.HttpV1Handler(log))
			r.Get("/tasks", get_tasks.HttpV1Handler(log))
			r.Post("/tasks/solve", solve_task.HttpV1Handler(log))
		})
	})

	return r
}
