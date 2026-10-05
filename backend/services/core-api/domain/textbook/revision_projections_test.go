package textbook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type projectionHistoryRepository struct {
	Repository
	owner   uuid.UUID
	history *ChapterWorkspace
}

func (r projectionHistoryRepository) GetChapterWorkspace(_ context.Context, owner, chapter uuid.UUID) (*ChapterWorkspace, error) {
	if owner != r.owner || chapter != r.history.Chapter.ID {
		return nil, ErrNotFound
	}
	return r.history, nil
}

func TestRevisionProjectionsKeepHistoricalApprovalAndRejectOtherContent(t *testing.T) {
	owner, chapterID, oldID, currentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	document, err := json.Marshal(map[string]any{"format": "inkwords.textbook.sample.v1", "sample": map[string]any{"learning_arc": validProjectionLearningArc()}, "video_runbook": validProjectionRunbook()})
	require.NoError(t, err)
	old := ChapterRevision{ID: oldID, ChapterID: chapterID, Kind: sharedtextbook.RevisionKindApproved, Markdown: "# 旧批准稿", ContentHash: strings.Repeat("a", 64), DocumentJSON: document}
	current := old
	current.ID = currentID
	current.Markdown = "# 新批准稿"
	candidate := old
	candidate.ID = uuid.New()
	candidate.Kind = sharedtextbook.RevisionKindCandidate
	history := &ChapterWorkspace{Chapter: &Chapter{ID: chapterID, Title: "章节", ApprovedRevisionID: &currentID}, Revisions: []ChapterRevision{old, current, candidate}}
	service := NewService(projectionHistoryRepository{owner: owner, history: history})
	projection, err := service.GetRevisionProjections(context.Background(), owner, chapterID, oldID)
	require.NoError(t, err)
	require.Equal(t, old.Markdown, projection.Blog.Markdown)
	require.Equal(t, oldID.String(), projection.Learning.RevisionID)
	require.Equal(t, owner, projection.WorkspaceID)
	projection, err = service.GetApprovedRevisionProjections(context.Background(), owner, chapterID)
	require.NoError(t, err)
	require.Equal(t, currentID.String(), projection.Learning.RevisionID)
	_, err = service.GetRevisionProjections(context.Background(), owner, chapterID, candidate.ID)
	require.ErrorIs(t, err, ErrInvalidState)
	_, err = service.GetRevisionProjections(context.Background(), uuid.New(), chapterID, oldID)
	require.Error(t, err)
	_, err = service.GetRevisionProjections(context.Background(), owner, chapterID, uuid.New())
	require.ErrorIs(t, err, ErrInvalidState)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return owner, nil }))
	router.GET("/chapters/:chapterID/projections", NewHandler(service).GetApprovedChapterProjections)
	for _, test := range []struct {
		revision string
		status   int
	}{{oldID.String(), http.StatusOK}, {candidate.ID.String(), http.StatusBadRequest}, {"invalid", http.StatusBadRequest}, {uuid.Nil.String(), http.StatusBadRequest}} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/chapters/"+chapterID.String()+"/projections?revision_id="+test.revision, nil))
		require.Equal(t, test.status, response.Code)
		if test.status == http.StatusOK {
			require.Contains(t, response.Body.String(), old.Markdown)
			require.Contains(t, response.Body.String(), owner.String())
		}
	}
}
