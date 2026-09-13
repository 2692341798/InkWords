package rabbitmq

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTextbookEnvelopesRemainDecodableBySharedWorkerContracts(t *testing.T) {
	workspaceID := uuid.New()
	generationRaw, err := json.Marshal(TextbookGenerationRequestedMessage{
		TaskID: uuid.New(), Kind: "textbook_sample_generate", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	var generation GenerationRequestedMessage
	require.NoError(t, json.Unmarshal(generationRaw, &generation))
	require.Equal(t, workspaceID, *generation.WorkspaceID)
	require.Equal(t, GenerationRequestedMessage{}.RoutingKey(), TextbookGenerationRequestedMessage{}.RoutingKey())

	parseRaw, err := json.Marshal(TextbookParseRequestedMessage{
		TaskID: uuid.New(), Kind: "textbook_source_import", WorkspaceID: &workspaceID, Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	var parse ParseRequestedMessage
	require.NoError(t, json.Unmarshal(parseRaw, &parse))
	require.Equal(t, workspaceID, *parse.WorkspaceID)
	require.Equal(t, ParseRequestedMessage{}.RoutingKey(), TextbookParseRequestedMessage{}.RoutingKey())
}

func TestLegacyUserIdentityIsIgnoredAndNotRepublished(t *testing.T) {
	workspaceID := uuid.New()
	raw := []byte(`{"task_id":"11111111-1111-1111-1111-111111111111","kind":"generate_single","user_id":"22222222-2222-2222-2222-222222222222","workspace_id":"` + workspaceID.String() + `","payload":{}}`)
	var message GenerationRequestedMessage
	require.NoError(t, json.Unmarshal(raw, &message))
	require.Equal(t, workspaceID, *message.WorkspaceID)

	republished, err := json.Marshal(message)
	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(republished, &envelope))
	require.NotContains(t, envelope, "user_id")
}
