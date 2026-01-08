package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

const (
	maxBytes = 1024 * 1024 // 1MB
)

func MustParse(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	log zerolog.Logger,
) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		var msg string
		var code int

		switch {
		case errors.As(err, &syntaxError):
			msg = fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			code = http.StatusBadRequest
		case errors.Is(err, io.ErrUnexpectedEOF):
			msg = "Request body contains badly-formed JSON"
			code = http.StatusBadRequest
		case errors.As(err, &unmarshalTypeError):
			msg = fmt.Sprintf(
				"Request body contains an invalid value for the %q field (at position %d)",
				unmarshalTypeError.Field,
				unmarshalTypeError.Offset,
			)
			code = http.StatusBadRequest
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg = fmt.Sprintf("Request body contains unknown field %s", fieldName)
			code = http.StatusBadRequest
		case errors.Is(err, io.EOF):
			msg = "Request body must not be empty"
			code = http.StatusBadRequest
		case errors.As(err, &maxBytesError):
			msg = fmt.Sprintf("Request body must not be larger than %d bytes", maxBytesError.Limit)
			code = http.StatusRequestEntityTooLarge
		default:
			msg = http.StatusText(http.StatusInternalServerError)
			code = http.StatusInternalServerError

			log.Error().
				Err(err).
				Msg("Unexpected error on parsing request body")
		}

		http.Error(w, msg, code)
		return false
	}

	if decoder.More() {
		msg := "Request body must only contain a single JSON object"
		code := http.StatusBadRequest

		http.Error(w, msg, code)
		return false
	}

	return true
}
