package mq

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

type recordingPublishChannel struct{ published amqp.Publishing }

func (channel *recordingPublishChannel) PublishWithContext(_ context.Context, _ string, _ string, _ bool, _ bool, publishing amqp.Publishing) error {
	channel.published = publishing
	return nil
}

func TestPublisherPreservesTextbookWorkspaceIdentity(t *testing.T) {
	workspaceID := uuid.New()
	tests := []struct {
		name    string
		publish func(*Publisher) error
	}{
		{name: "generation", publish: func(publisher *Publisher) error {
			return publisher.PublishTextbookGenerationRequested(t.Context(), sharedrabbitmq.TextbookGenerationRequestedMessage{TaskID: uuid.New(), Kind: "textbook_sample_generate", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`)})
		}},
		{name: "parse", publish: func(publisher *Publisher) error {
			return publisher.PublishTextbookParseRequested(t.Context(), sharedrabbitmq.TextbookParseRequestedMessage{TaskID: uuid.New(), Kind: "textbook_source_import", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`)})
		}},
		{name: "verification", publish: func(publisher *Publisher) error {
			return publisher.PublishTextbookVerificationRequested(t.Context(), sharedrabbitmq.TextbookVerificationRequestedMessage{TaskID: uuid.New(), Kind: "textbook_teaching_artifact_verify", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`)})
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &recordingPublishChannel{}
			publisher := &Publisher{channel: channel, exchange: "test"}
			require.NoError(t, test.publish(publisher))
			var envelope struct {
				WorkspaceID *uuid.UUID `json:"workspace_id"`
			}
			require.NoError(t, json.Unmarshal(channel.published.Body, &envelope))
			require.NotNil(t, envelope.WorkspaceID)
			require.Equal(t, workspaceID, *envelope.WorkspaceID)
		})
	}
}

func TestTextbookVerificationEnvelopeHasNoLegacyUserIdentity(t *testing.T) {
	workspaceID := uuid.New()
	channel := &recordingPublishChannel{}
	publisher := &Publisher{channel: channel, exchange: "test"}
	require.NoError(t, publisher.PublishTextbookVerificationRequested(t.Context(), sharedrabbitmq.TextbookVerificationRequestedMessage{
		TaskID: uuid.New(), Kind: "textbook_teaching_artifact_verify", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
	}))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(channel.published.Body, &envelope))
	require.NotContains(t, envelope, "user_id")
}

func TestTextbookGenerationAndParseEnvelopesHaveNoLegacyUserIdentity(t *testing.T) {
	workspaceID := uuid.New()
	tests := []struct {
		name    string
		publish func(*Publisher) error
	}{
		{name: "generation", publish: func(publisher *Publisher) error {
			return publisher.PublishTextbookGenerationRequested(t.Context(), sharedrabbitmq.TextbookGenerationRequestedMessage{
				TaskID: uuid.New(), Kind: "textbook_sample_generate", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
			})
		}},
		{name: "parse", publish: func(publisher *Publisher) error {
			return publisher.PublishTextbookParseRequested(t.Context(), sharedrabbitmq.TextbookParseRequestedMessage{
				TaskID: uuid.New(), Kind: "textbook_source_import", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
			})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &recordingPublishChannel{}
			publisher := &Publisher{channel: channel, exchange: "test"}
			require.NoError(t, test.publish(publisher))
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(channel.published.Body, &envelope))
			require.NotContains(t, envelope, "user_id")
		})
	}
}

func TestGenerationParseAndExportEnvelopesUseWorkspaceIdentityOnly(t *testing.T) {
	workspaceID := uuid.New()
	tests := []struct {
		name    string
		publish func(*Publisher) error
	}{
		{name: "generation", publish: func(publisher *Publisher) error {
			return publisher.PublishGenerationRequested(t.Context(), sharedrabbitmq.GenerationRequestedMessage{
				TaskID: uuid.New(), Kind: "generate_single", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
			})
		}},
		{name: "parse", publish: func(publisher *Publisher) error {
			return publisher.PublishParseRequested(t.Context(), sharedrabbitmq.ParseRequestedMessage{
				TaskID: uuid.New(), Kind: "parse_file", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
			})
		}},
		{name: "export", publish: func(publisher *Publisher) error {
			return publisher.PublishExportRequested(t.Context(), sharedrabbitmq.ExportRequestedMessage{
				TaskID: uuid.New(), Kind: "export_pdf", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
			})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			channel := &recordingPublishChannel{}
			publisher := &Publisher{channel: channel, exchange: "test"}
			require.NoError(t, test.publish(publisher))
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(channel.published.Body, &envelope))
			require.Equal(t, workspaceID.String(), envelope["workspace_id"])
			require.NotContains(t, envelope, "user_id")
		})
	}
}
