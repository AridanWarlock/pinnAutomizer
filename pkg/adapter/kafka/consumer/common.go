package consumer

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/kafka/segmentio"
)

type Handler = func(ctx context.Context, msg segmentio.Message) error
