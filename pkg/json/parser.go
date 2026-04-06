package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

const (
	maxBytes = 1024 * 1024 // 1MB
)

func Parse(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		msg, code := analyseDecodeError(err)

		http.Error(w, msg, code)
		return fmt.Errorf("decode request body: %w", err)
	}

	if decoder.More() {
		msg := "Request body must only contain a single JSON object"
		code := http.StatusBadRequest

		http.Error(w, msg, code)
		return fmt.Errorf("decode request body: %w", domain.ErrMultipleRequestJsons)
	}

	return nil
}

func analyseDecodeError(err error) (string, int) {
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
	}

	return msg, code
}
