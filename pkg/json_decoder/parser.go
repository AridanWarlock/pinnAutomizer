package json_decoder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxBytes = 1024 * 1024 // 1MB
)

var (
	ErrBadJson               = errors.New("bad json")
	ErrEntityToLarge         = errors.New("entity to large")
	ErrUnexpectedDecodeError = errors.New("unexpected decode error")
)

func ParseRequestBody(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		err = analyseDecodeError(err)

		return fmt.Errorf("decode request body: %w", err)
	}

	if decoder.More() {
		msg := "request body must only contain a single JSON object"
		return fmt.Errorf("decode request body: %w: %s", ErrBadJson, msg)
	}

	return nil
}

func analyseDecodeError(err error) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var maxBytesError *http.MaxBytesError

	finalErr := ErrBadJson
	var msg string

	switch {
	case errors.As(err, &syntaxError):
		msg = fmt.Sprintf("request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
	case errors.Is(err, io.ErrUnexpectedEOF):
		msg = "request body contains badly-formed JSON"
	case errors.As(err, &unmarshalTypeError):
		msg = fmt.Sprintf(
			"request body contains an invalid value for the %q field (at position %d)",
			unmarshalTypeError.Field,
			unmarshalTypeError.Offset,
		)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		msg = fmt.Sprintf("request body contains unknown field %s", fieldName)
	case errors.Is(err, io.EOF):
		msg = "request body must not be empty"
	case errors.As(err, &maxBytesError):
		msg = fmt.Sprintf("request body must not be larger than %d bytes", maxBytesError.Limit)
		finalErr = ErrEntityToLarge
	default:
		return fmt.Errorf("%w: %v", ErrUnexpectedDecodeError, err)
	}

	return fmt.Errorf("%w: %s", finalErr, msg)
}
