package kafka

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func TestProducerBatchAcrossTopicsIntegration(t *testing.T) {
	addr := os.Getenv("PINN_TEST_KAFKA_ADDR")
	if addr == "" {
		t.Skip("set PINN_TEST_KAFKA_ADDR for a disposable Kafka broker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := kafkago.DialContext(ctx, "tcp", addr)
	require.NoError(t, err)
	defer connection.Close()
	controller, err := connection.Controller()
	require.NoError(t, err)
	admin, err := kafkago.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	require.NoError(t, err)
	defer admin.Close()
	deadline, _ := ctx.Deadline()
	require.NoError(t, admin.SetDeadline(deadline))
	topics := []string{"pinn-test-" + uuid.NewString(), "pinn-test-" + uuid.NewString()}
	for _, topic := range topics {
		require.NoError(t, admin.CreateTopics(kafkago.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}))
	}
	t.Cleanup(func() {
		cleanupContext, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		cleanup, err := kafkago.DialContext(cleanupContext, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
		if err == nil {
			defer cleanup.Close()
			_ = cleanup.SetDeadline(time.Now().Add(5 * time.Second))
			_ = cleanup.DeleteTopics(topics...)
		}
	})
	producer := NewProducer(ProducerConfig{Broker: addr}, zerolog.Nop())
	defer producer.Close()
	var messages []core.KafkaMessage
	for _, topic := range topics {
		key, err := core.NewIdempotencyKey(uuid.NewString())
		require.NoError(t, err)
		message, err := core.NewProduceKafkaMessage(topic, []byte("task"), []byte("result"), nil)
		require.NoError(t, err)
		message, err = message.WithIdKey(key)
		require.NoError(t, err)
		messages = append(messages, message)
	}
	require.NoError(t, producer.WriteMessages(ctx, messages...))
	for _, expected := range messages {
		reader := kafkago.NewReader(kafkago.ReaderConfig{Brokers: []string{addr}, Topic: expected.Topic, Partition: 0, MaxBytes: 1e6})
		t.Cleanup(func() {
			_ = reader.Close()
		})
		actual, err := reader.ReadMessage(ctx)
		closeErr := reader.Close()
		require.NoError(t, err)
		require.NoError(t, closeErr)
		require.Equal(t, expected.Key, actual.Key)
		require.Equal(t, expected.Value, actual.Value)
		var headers core.KafkaHeaders = make(core.KafkaHeaders)
		for _, header := range actual.Headers {
			headers[header.Key] = string(header.Value)
		}
		require.Equal(t, expected.Headers, headers)
	}
}
