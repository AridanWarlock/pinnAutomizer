package create_script

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/pkg/render"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Response struct {
	Filename string    `json:"filename"`
	Path     string    `json:"path"`
	UserID   uuid.UUID `json:"userID"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_V1: script.CreateScript").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		httpV1(w, r, log)
	}
}

func httpV1(w http.ResponseWriter, r *http.Request, log zerolog.Logger) {
	log = log.With().Ctx(r.Context()).Logger()

	userID := r.Context().Value(auth.UserIDKey).(uuid.UUID)

	path, filename, err := uploadScriptsFile(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("upload script file err: %s", err.Error()), http.StatusBadRequest)
		return
	}

	input := Input{
		Filename: filename,
		Path:     path,
		UserID:   userID,
	}

	out, err := usecase.CreateScript(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	render.JSON(w, out, http.StatusOK)
}

func uploadScriptsFile(r *http.Request) (string, string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return "", "", errors.New("creating multipart reader")
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", errors.New("failed to read part")
		}

		if part.FormName() == "file" {
			filename := part.FileName()

			path, err := uploadScript(part, filename)
			defer part.Close()

			if err != nil {
				return "", "", err
			}

			return path, filename, nil
		}
		_ = part.Close()
	}

	return "", "", errors.New("not found file in request data")
}

func uploadScript(part io.Reader, filename string) (string, error) {
	abs, err := generateScriptFilepath(filename)
	if err != nil {
		return "", err
	}

	dst, err := os.Create(abs)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	buffer := make([]byte, 1024*1024) // 1MB buffer
	_, err = io.CopyBuffer(dst, part, buffer)
	if err != nil {
		return "", err
	}

	return abs, nil
}

func generateScriptFilepath(original string) (string, error) {
	dir := fmt.Sprintf("/tmp/%s", uuid.NewString())
	err := os.Mkdir(dir, os.ModeDir)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", dir, original), nil
}
