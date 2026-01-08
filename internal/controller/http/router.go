package http

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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type AuthMiddleware interface {
	Authenticate(http.Handler) http.Handler
}

func Router(authMiddleware AuthMiddleware) http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.Recoverer,
		cors.CORS(cors.DefaultCORSConfig()),
		authMiddleware.Authenticate,
	)

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			//auth
			r.Post("/auth/login", login.HTTPv1)
			r.Post("/auth/logout", logout.HTTPv1)
			r.Get("/auth/me", me.HTTPv1)
			r.Post("/auth/refresh", refresh.HTTPv1)
			r.Post("/auth/register", register.HTTPv1)

			//scripts
			r.Post("/scripts", create_script.HTTPv1)
			r.Get("/scripts", search_scripts.HTTPv1)
			r.Post("/scripts/from-translate/{id}", update_script_after_translate.HTTPv1)
		})
	})

	return r
}
