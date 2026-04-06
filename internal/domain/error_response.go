package domain

type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func NewErrorMessage(message string, err error) ErrorResponse {
	return ErrorResponse{
		Message: message,
		Error:   err.Error(),
	}
}
