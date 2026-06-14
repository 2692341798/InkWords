package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	coretask "inkwords-backend/services/core-api/domain/task"
)

type stubTaskPublisher struct{}

func (stubTaskPublisher) PublishGenerationRequested(context.Context, coretask.GenerationRequestedMessage) error {
	return nil
}

func (stubTaskPublisher) PublishParseRequested(context.Context, coretask.ParseRequestedMessage) error {
	return nil
}

func (stubTaskPublisher) PublishExportRequested(context.Context, coretask.ExportRequestedMessage) error {
	return nil
}

func TestInitTaskPublisherFromEnv_UsesConfiguredExchange(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	t.Setenv("RABBITMQ_EXCHANGE", "custom.exchange")

	expectedPublisher := stubTaskPublisher{}
	cleanupCalled := false

	publisher, cleanup, err := initTaskPublisherFromEnv(func(url string, exchange string) (coretask.Publisher, func(), error) {
		require.Equal(t, "amqp://guest:guest@rabbitmq:5672/", url)
		require.Equal(t, "custom.exchange", exchange)
		return expectedPublisher, func() {
			cleanupCalled = true
		}, nil
	})
	require.NoError(t, err)
	require.Equal(t, expectedPublisher, publisher)

	cleanup()
	require.True(t, cleanupCalled)
}

func TestInitTaskPublisherFromEnv_UsesDefaultExchangeWhenUnset(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	t.Setenv("RABBITMQ_EXCHANGE", "")

	publisher, cleanup, err := initTaskPublisherFromEnv(func(url string, exchange string) (coretask.Publisher, func(), error) {
		require.Equal(t, "amqp://guest:guest@rabbitmq:5672/", url)
		require.Equal(t, "inkwords.events", exchange)
		return stubTaskPublisher{}, func() {}, nil
	})
	require.NoError(t, err)
	require.NotNil(t, publisher)
	cleanup()
}

func TestInitTaskPublisherFromEnv_RequiresRabbitMQURL(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "")

	_, _, err := initTaskPublisherFromEnv(func(string, string) (coretask.Publisher, func(), error) {
		t.Fatal("factory should not be called when url is missing")
		return nil, nil, nil
	})
	require.ErrorContains(t, err, "RABBITMQ_URL")
}
