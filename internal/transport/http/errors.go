package core_http

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
