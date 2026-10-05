package textbookartifact

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	textbookdomain "inkwords-backend/services/core-api/domain/textbook"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type recordingSampleWriter struct {
	called bool
	err    error
}

func (writer *recordingSampleWriter) PersistTextbookSampleResult(context.Context, uuid.UUID, map[string]any) error {
	writer.called = true
	return writer.err
}

type fakeRevisionResolver struct {
	context *textbookdomain.GeneratedRevisionContext
	err     error
}

func (resolver fakeRevisionResolver) GetGeneratedRevisionContext(context.Context, uuid.UUID) (*textbookdomain.GeneratedRevisionContext, error) {
	return resolver.context, resolver.err
}

type recordingProjectionStager struct {
	input Input
	err   error
}

func (stager *recordingProjectionStager) StageAndRegister(_ context.Context, _ uuid.UUID, input Input) (*textbookdomain.CodeArtifactRow, error) {
	stager.input = input
	return nil, stager.err
}

func TestSampleProjectionPersisterStagesGinArtifactWithoutChangingCandidateOutcome(t *testing.T) {
	writer := &recordingSampleWriter{}
	stager := &recordingProjectionStager{err: errors.New("artifact disk is full")}
	revisionID := uuid.New()
	markdown := projectionManuscript("candidate-specific")
	persister := NewSampleProjectionPersister(writer, fakeRevisionResolver{context: &textbookdomain.GeneratedRevisionContext{Revision: textbookdomain.ChapterRevision{ID: revisionID, Markdown: markdown, ContentHash: fmt.Sprintf("%x", sha256.Sum256([]byte(markdown))), Kind: sharedtextbook.RevisionKindCandidate, CreatedBy: textbookdomain.RevisionCreatorGeneration}, WorkspaceID: uuid.New()}}, stager)
	require.NoError(t, persister.PersistTextbookSampleResult(context.Background(), uuid.New(), map[string]any{"task_subtype": sharedtextbook.TextbookSampleGenerationTaskSubtype}))
	require.True(t, writer.called)
	require.Equal(t, revisionID, stager.input.RevisionID)
	require.NotEqual(t, uuid.Nil, stager.input.ArtifactID)
	require.Len(t, stager.input.Files, 3)
	require.Contains(t, string(stager.input.Files[1].Content), "candidate-specific")
	require.Equal(t, []sharedtextbook.VerificationCommand{{Kind: "go_test"}}, stager.input.Commands)
}

func projectionManuscript(value string) string {
	return "**代码来源：教学实现（main.go）。** 非生产用途；省略并发处理。\n\n```go\npackage main\n\nfunc main() { println(\"" + value + "\") }\n```\n\n**代码来源：教学实现测试（main_test.go）。** 只验证教学代码。\n\n```go\npackage main\nimport \"testing\"\nfunc TestMainOutput(t *testing.T) { main() }\n```\n"
}

func TestSampleProjectionPersisterPropagatesAuthoritativeCandidateFailure(t *testing.T) {
	writer := &recordingSampleWriter{err: errors.New("candidate write failed")}
	persister := NewSampleProjectionPersister(writer, nil, nil)
	err := persister.PersistTextbookSampleResult(context.Background(), uuid.New(), nil)
	require.ErrorContains(t, err, "candidate write failed")
}
