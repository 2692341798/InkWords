package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fixedLegacyNoteSource struct{ notes []ReviewNote }

func (source fixedLegacyNoteSource) ListEligibleNotes(context.Context) ([]ReviewNote, error) {
	return source.notes, nil
}

type recordingLegacyMasteryAdapter struct {
	note    ReviewNote
	calls   int
	created bool
}

func (adapter *recordingLegacyMasteryAdapter) MigrateLegacyNote(_ context.Context, _ uuid.UUID, note ReviewNote) (uuid.UUID, bool, error) {
	adapter.note = note
	adapter.calls++
	return uuid.MustParse("11111111-1111-1111-1111-111111111111"), adapter.created, nil
}

func TestServiceMigratesOnlyEligibleSourceNoteToWorkspaceAdapter(t *testing.T) {
	note := ReviewNote{NotePath: "wiki/concepts/router.md", Title: "路由", Body: "来自受控笔记源的正文"}
	adapter := &recordingLegacyMasteryAdapter{created: true}
	service := NewService(nil, fixedLegacyNoteSource{notes: []ReviewNote{note}}, nil).WithLegacyMasteryAdapter(adapter)

	result, err := service.MigrateLegacyNote(context.Background(), uuid.New(), note.NotePath)
	require.NoError(t, err)
	require.True(t, result.Created)
	require.Equal(t, note.NotePath, result.NotePath)
	require.Equal(t, note, adapter.note)
	require.Equal(t, 1, adapter.calls)

	_, err = service.MigrateLegacyNote(context.Background(), uuid.New(), "wiki/concepts/not-from-source.md")
	require.ErrorIs(t, err, errReviewNoteNotFound)
	require.Equal(t, 1, adapter.calls)
}
