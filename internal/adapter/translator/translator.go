package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"pinnAutomizer/internal/domain"
	"time"

	"github.com/rs/zerolog"
)

type Config struct {
	Host    string        `env:"TRANSLATOR_HOST" required:"true"`
	Port    string        `env:"TRANSLATOR_PORT" required:"true"`
	Timeout time.Duration `env:"TRANSLATOR_TIMEOUT" required:"true"`
}

type Translator struct {
	client  *http.Client
	baseUrl string
	log     zerolog.Logger
}

func New(c Config, log zerolog.Logger) *Translator {
	client := &http.Client{
		Timeout: c.Timeout,
	}

	return &Translator{
		client:  client,
		baseUrl: fmt.Sprintf("%s:%s", c.Host, c.Port),
		log:     log.With().Str("component", "translator").Logger(),
	}
}

func (t *Translator) Translate(ctx context.Context, translate *domain.ToTranslate) error {
	url := fmt.Sprintf("http://%s/api/v1/scripts/to-translate/%s", t.baseUrl, translate.ID)

	body, err := json.Marshal(&struct {
		Path string `json:"path"`
	}{
		Path: translate.Path,
	})

	t.log.Info().
		Str("body", string(body)).
		Msg("requesting translator to translate")

	if err != nil {
		return err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/json")

	response, err := t.client.Do(r)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("tranlate status code: %d", response.StatusCode)
	}

	return r.Body.Close()
}
