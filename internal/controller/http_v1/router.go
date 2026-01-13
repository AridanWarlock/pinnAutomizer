package http_v1

import (
	"net/http"
	"pinnAutomizer/internal/auth/login"
	"pinnAutomizer/internal/auth/logout"
	"pinnAutomizer/internal/auth/me"
	"pinnAutomizer/internal/auth/refresh"
	"pinnAutomizer/internal/auth/register"
	"pinnAutomizer/internal/middleware/cors"
	"pinnAutomizer/internal/script/create_script"
	"pinnAutomizer/internal/script/search_scripts"
	"pinnAutomizer/internal/script/update_script_after_translate"
	"pinnAutomizer/internal/task/create_task"
	"pinnAutomizer/internal/task/get_tasks"

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

			//scripts
			r.Post("/scripts", create_script.HttpV1Handler(log))
			r.Get("/scripts", search_scripts.HttpV1Handler(log))
			r.Post("/scripts/from-translate/{id}", update_script_after_translate.HttpV1Handler(log))

			//tasks
			r.Post("/tasks", create_task.HttpV1Handler(log))
			r.Get("/tasks", get_tasks.HttpV1Handler(log))
		})
	})

	return r
}
