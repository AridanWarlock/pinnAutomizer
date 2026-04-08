package httpResponse

import "net/http"

const StatusCodeUninitialized = -1

type ResponseWriter struct {
	w http.ResponseWriter

	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		w:          w,
		statusCode: StatusCodeUninitialized,
	}
}

func (rw *ResponseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == StatusCodeUninitialized {
		rw.statusCode = http.StatusOK
	}
	return rw.w.Write(b)
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.w.WriteHeader(statusCode)
	rw.statusCode = statusCode
}

func (rw *ResponseWriter) GetStatusCode() int {
	if rw.statusCode == StatusCodeUninitialized {
		return http.StatusOK
	}

	return rw.statusCode
}
