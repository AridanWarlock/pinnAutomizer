package httpout

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
} // @name ErrorResponse

func NewErrorResponse(err error, msg string) ErrorResponse {
	return ErrorResponse{
		Error:   err.Error(),
		Message: msg,
	}
}
