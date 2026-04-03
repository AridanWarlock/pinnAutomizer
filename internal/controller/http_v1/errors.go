package http_v1

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
