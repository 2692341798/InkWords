package textbook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	shared "inkwords-backend/shared/kernel/textbook"
)

func TestBookNoticeSubjectsMustBelongToExactApprovedSet(t *testing.T) {
	id := uuid.New()
	notices, err := shared.FreezePublicationNotices([]shared.PublicationNoticeDraft{{Title: "许可", Text: "fixture license", SourceURL: "https://example.org/LICENSE", PreparedBy: "委托 AI", SubjectRefs: []string{"chapter-revision:" + id.String()}}})
	require.NoError(t, err)
	require.NoError(t, validateBookNoticeSubjects(notices, []bookBuildChapterSource{{RevisionID: id}}, nil))
	require.ErrorIs(t, validateBookNoticeSubjects(notices, []bookBuildChapterSource{{RevisionID: uuid.New()}}, nil), ErrInvalidState)
}

func testFrozenBookNotices(t *testing.T, database *gorm.DB, workspaceID uuid.UUID, project *Project, approved *ChapterRevision) {
	t.Helper()
	tx := database.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	repo := NewGormRepository(tx)
	input := CreateBookBuildInput{ProjectID: project.ID, Notices: []shared.PublicationNoticeDraft{{Title: "许可声明", Text: "Copyright fixture\nLicense fixture.\n", SourceURL: "https://example.org/LICENSE", PreparedBy: "用户委托 AI", SubjectRefs: []string{"chapter-revision:" + approved.ID.String()}}}}
	first, err := repo.CreateBookBuild(context.Background(), workspaceID, input)
	require.NoError(t, err)
	var manifest struct {
		Book shared.CanonicalBookAST `json:"book"`
	}
	require.NoError(t, json.Unmarshal(first.ManifestJSON, &manifest))
	require.NoError(t, manifest.Book.Validate())
	require.Equal(t, shared.CanonicalBookASTWithNoticesFormat, manifest.Book.Format)
	require.Equal(t, input.Notices[0].Text, manifest.Book.Notices[0].Text)
	retry, err := repo.CreateBookBuild(context.Background(), workspaceID, input)
	require.NoError(t, err)
	require.Equal(t, first.ID, retry.ID)
	input.Notices[0].Text += "New attribution evidence."
	next, err := repo.CreateBookBuild(context.Background(), workspaceID, input)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, next.ID)
	var retained BookBuildRow
	require.NoError(t, tx.First(&retained, "id = ?", first.ID).Error)
	require.Equal(t, first.ManifestHash, retained.ManifestHash)
	require.JSONEq(t, string(first.ManifestJSON), string(retained.ManifestJSON))
	input.Notices[0].SubjectRefs = []string{"chapter-revision:" + uuid.NewString()}
	_, err = repo.CreateBookBuild(context.Background(), workspaceID, input)
	require.ErrorIs(t, err, ErrInvalidState)
	t.Log("publication notices: real PostgreSQL freeze, retry, changed statement and foreign subject rejection passed")
}
