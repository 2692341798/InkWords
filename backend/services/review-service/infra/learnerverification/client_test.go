package learnerverification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestClientReadsCapabilityAndSendsOnlyTheReference(t *testing.T) {
	runner := sharedtextbook.LearnerRunnerIdentity{ImageDigest: digest('a'), ToolchainVersion: "go1.25.4", SandboxProfileDigest: sharedtextbook.LearnerSandboxProfileDigest}
	capability := sharedtextbook.LearnerVerificationCapability{Format: sharedtextbook.LearnerVerificationCapabilityFormat, Accepted: true, Available: true, Profile: sharedtextbook.LearnerGoTestProfile, Runner: &runner}
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: uuid.NewString(), WorkspaceID: uuid.NewString(), ObjectiveID: uuid.NewString(), AttemptID: uuid.NewString(), InputHash: digest('b'), ClaimToken: strings.Repeat("A", 43)}
	now := time.Now().UTC()
	report := sharedtextbook.LearnerVerificationReport{Format: sharedtextbook.LearnerVerificationReportFormat, RunID: reference.RunID, InputHash: reference.InputHash, SnapshotHash: digest('c'), Runner: runner, Profile: sharedtextbook.LearnerGoTestProfile, Status: sharedtextbook.LearnerVerificationUnavailable, CompletedAt: now, Reason: "preflight"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodGet {
			require.Equal(t, "/internal/v1/learner-verification/capability", request.URL.Path)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": capability})
			return
		}
		require.Equal(t, "/internal/v1/learner-verification/runs", request.URL.Path)
		var received map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&received))
		require.Equal(t, 7, len(received))
		require.NotContains(t, received, "files")
		require.NotContains(t, received, "command")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": report})
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	require.NoError(t, err)
	gotCapability, err := client.Capability(context.Background())
	require.NoError(t, err)
	require.Equal(t, capability, gotCapability)
	gotReport, err := client.Execute(context.Background(), reference)
	require.NoError(t, err)
	require.Equal(t, report, gotReport)
}

func digest(character byte) string { return "sha256:" + strings.Repeat(string(character), 64) }
