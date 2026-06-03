package mq

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	taskdomain "inkwords-backend/internal/domain/task"
)

func TestGenerationRequestedMessage_RoutingKey(t *testing.T) {
	msg := GenerationRequestedMessage{
		TaskID:  uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:    "generate_series",
		UserID:  uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{"series_title":"Go 并发"}`),
	}

	require.Equal(t, "generation.requested", msg.RoutingKey())
}

func TestMarshalMessage_ContainsTaskID(t *testing.T) {
	msg := GenerationRequestedMessage{
		TaskID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:   "generate_single",
	}

	body, err := json.Marshal(msg)
	require.NoError(t, err)
	require.Contains(t, string(body), `"task_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`)
}

func TestPublisher_PublishGenerationRequested_UsesRoutingKeyAndPersistentJSON(t *testing.T) {
	fakeChannel := &stubChannel{}
	publisher := &Publisher{
		channel:  fakeChannel,
		exchange: "inkwords.events",
	}

	message := taskdomain.GenerationRequestedMessage{
		TaskID:  uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Kind:    "generate_series",
		UserID:  uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload: json.RawMessage(`{"series_title":"Go 并发"}`),
	}

	err := publisher.PublishGenerationRequested(context.Background(), message)
	require.NoError(t, err)
	require.Equal(t, "inkwords.events", fakeChannel.exchange)
	require.Equal(t, "generation.requested", fakeChannel.key)
	require.Equal(t, "application/json", fakeChannel.message.ContentType)
	require.Equal(t, uint8(amqp.Persistent), fakeChannel.message.DeliveryMode)
	require.Contains(t, string(fakeChannel.message.Body), `"task_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"`)
}

type stubChannel struct {
	exchange  string
	key       string
	message   amqp.Publishing
	mandatory bool
	immediate bool
}

func (s *stubChannel) PublishWithContext(
	_ context.Context,
	exchange string,
	key string,
	mandatory bool,
	immediate bool,
	message amqp.Publishing,
) error {
	s.exchange = exchange
	s.key = key
	s.mandatory = mandatory
	s.immediate = immediate
	s.message = message
	return nil
}
