package cors

import (
	"net/http"

	"github.com/go-chi/cors"
)

type Config struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() Config {
	return Config{
		AllowedOrigins: []string{
			"http://localhost:3031",
			"http://127.0.0.1:3031",
			"http://0.0.0.0:3031",
			"http://10.8.1.1:3031",
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
			"Origin",
			"X-Requested-With",
			"X-CSRF-Token",

			"Content-Disposition",
			"Content-Length",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Set-Cookie",
		},
		AllowCredentials: true,
		MaxAge:           300,
	}
}

func NewChiCORS(config Config) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   config.AllowedMethods,
		AllowedHeaders:   config.AllowedHeaders,
		ExposedHeaders:   config.ExposedHeaders,
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	})
}
