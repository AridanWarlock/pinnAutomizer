package update_task_status_after_train

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	Addr  []string `env:"KAFKA_CONSUMER_ADDR"`
	Topic string   `env:"KAFKA_CONSUMER_AFTER_TRAIN_TOPIC"`
	Group string   `env:"KAFKA_CONSUMER_AFTER_TRAIN_GROUP"`
}

type Consumer struct {
	reader *kafka.Reader

	stop context.CancelFunc
	done chan struct{}

	log zerolog.Logger
}

func NewConsumer(cfg Config, log zerolog.Logger) *Consumer {
	ctx, stop := context.WithCancel(context.Background())

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          cfg.Addr,
		Topic:            cfg.Topic,
		GroupID:          cfg.Group,
		ReadBatchTimeout: time.Second,
		CommitInterval:   time.Second,
	})

	c := &Consumer{
		reader: reader,
		stop:   stop,
		done:   make(chan struct{}),

		log: log.With().Str("component", "consumer: task.UpdateTaskStatusAfterTrain").Logger(),
	}

	go c.run(ctx)

	return c
}

type Message struct {
	TaskID uuid.UUID `json:"taskID"`
}

func (c *Consumer) run(ctx context.Context) {
	log := c.log.With().Ctx(ctx).Logger()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			log.Error().Err(err).Msg("kafka consumer: FetchMessage")

			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				break
			}
		}

		var message Message
		if err = json.Unmarshal(m.Value, &message); err != nil {
			log.Error().Err(err).Msg("kafka consumer: Unmarshal message")
			continue
		}

		err = usecase.UpdateTaskStatusAfterTrain(ctx, Input{ID: message.TaskID})
		if err != nil {
			log.Error().Err(err).Msg("kafka consumer: UpdateTaskStatusAfterTrain")
			continue
		}

		err = c.reader.CommitMessages(ctx, m)
		if err != nil {
			log.Error().Err(err).Msg("kafka consumer: CommitMessages")
			return
		}
	}

	close(c.done)
}

func (c *Consumer) Close() {
	c.log.Info().Msg("kafka consumer: closing")

	c.stop()

	if err := c.reader.Close(); err != nil {
		c.log.Error().Err(err).Msg("kafka consumer: reader.Close")
	}

	<-c.done
	c.log.Info().Msg("kafka consumer: closed")
}
