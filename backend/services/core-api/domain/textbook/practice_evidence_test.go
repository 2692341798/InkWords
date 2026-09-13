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
	"gorm.io/gorm"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func assertApprovedPracticeEvidence(t *testing.T, ctx context.Context, db *gorm.DB, service *Service, owner uuid.UUID, chapter Chapter, chunk sharedtextbook.SourceChunk) {
	t.Helper()
	id := "evidence-" + chunk.ID
	var unrelated ParsedChunk
	require.NoError(t, db.Where("id = ?", chunk.ID).First(&unrelated).Error)
	unrelated.ID, unrelated.Ordinal = chunk.ID+"-unrelated", 123
	unrelated.SearchText = "本章的另一段资料与本次任务无关。"
	unrelated.TextHash = sharedtextbook.PracticeExcerptHash(unrelated.SearchText)
	require.NoError(t, db.Create(&unrelated).Error)
	otherEvidenceID := "evidence-" + unrelated.ID
	set := sharedtextbook.PracticeSet{Version: sharedtextbook.PracticeSetVersion}
	for _, mode := range []sharedtextbook.LearningTaskMode{sharedtextbook.LearningTaskExplain, sharedtextbook.LearningTaskComplete, sharedtextbook.LearningTaskReproduce, sharedtextbook.LearningTaskTransfer, sharedtextbook.LearningTaskDiagnose, sharedtextbook.LearningTaskRetain} {
		task := sharedtextbook.PracticeTask{ID: "task-" + string(mode), Mode: mode, Prompt: "具体问题 " + string(mode), Variation: "改换请求路径", ExpectedAnswer: "区分注册与查找", EvidenceIDs: []string{id}, Hints: []sharedtextbook.PracticeHint{{Level: 1, Text: "先回忆条件"}, {Level: 2, Text: "查看方法"}, {Level: 3, Text: "检查路径"}}}
		for _, criterion := range sharedtextbook.PracticeRubricDimensions(mode) {
			criterion.Description = "核对 " + criterion.ID
			task.Rubric = append(task.Rubric, criterion)
		}
		if mode == sharedtextbook.LearningTaskRetain {
			task.MinDelayHours = 24
		}
		set.Tasks = append(set.Tasks, task)
	}
	document, err := json.Marshal(map[string]any{"format": "inkwords.textbook.sample.v1", "sample": map[string]any{"learning_arc": validProjectionLearningArc(), "practice_set": set, "evidence_ids": []string{id, otherEvidenceID}}, "video_runbook": validProjectionRunbook()})
	require.NoError(t, err)
	revision := ChapterRevision{ID: uuid.New(), ChapterID: chapter.ID, RevisionNumber: 10000, Kind: sharedtextbook.RevisionKindApproved, Markdown: "# 历史批准稿\n\n" + sharedtextbook.RenderPracticeSet(set), DocumentJSON: document, ContentHash: strings.Repeat("c", 64), CreatedBy: "manual"}
	require.NoError(t, db.Create(&revision).Error)
	selected, err := service.GetPracticeEvidence(ctx, owner, chapter.ID, revision.ID, set.Tasks[0].ID)
	require.NoError(t, err)
	require.Len(t, selected.Sources, 1, "unrelated chapter evidence must not enter the scoring request")
	require.Equal(t, chunk.SearchText, selected.Sources[0].Excerpt)
	require.Equal(t, chunk.TextHash, selected.Sources[0].Reference.ContentHash)
	require.Equal(t, sharedtextbook.PracticeExcerptHash(chunk.SearchText), selected.Sources[0].ExcerptHash)
	require.NotEqual(t, selected.Sources[0].Reference.ContentHash, selected.Sources[0].ExcerptHash, "retain imported encoding provenance independently")
	_, err = service.GetPracticeEvidence(ctx, uuid.New(), chapter.ID, revision.ID, set.Tasks[0].ID)
	require.Error(t, err)
	_, err = service.GetPracticeEvidence(ctx, owner, uuid.New(), revision.ID, set.Tasks[0].ID)
	require.Error(t, err)
	_, err = service.GetPracticeEvidence(ctx, owner, chapter.ID, revision.ID, "not-approved-task")
	require.Error(t, err)
	router := gin.New()
	router.Use(httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return owner, nil }))
	router.GET("/chapters/:chapterID/practice-evidence", NewHandler(service).GetPracticeEvidence)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/chapters/"+chapter.ID.String()+"/practice-evidence?revision_id="+revision.ID.String()+"&task_id="+set.Tasks[0].ID, nil))
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), chunk.SearchText)
	require.NotContains(t, response.Body.String(), otherEvidenceID)
	// Approved prose uses readable aliases; a scoring request must retain those
	// identities while resolving only this task's workspace-owned source bytes.
	for index := range set.Tasks {
		set.Tasks[index].EvidenceIDs = []string{"gin-route-mechanism"}
	}
	aliased, err := json.Marshal(map[string]any{"format": "inkwords.textbook.sample.v1", "evidence_aliases": map[string]string{"gin-route-mechanism": id}, "sample": map[string]any{"learning_arc": validProjectionLearningArc(), "practice_set": set, "evidence_ids": []string{"gin-route-mechanism", "unrelated-unresolved-alias"}}, "video_runbook": validProjectionRunbook()})
	require.NoError(t, err)
	require.NoError(t, db.Model(&ChapterRevision{}).Where("id = ?", revision.ID).Updates(map[string]any{"document_json": aliased, "markdown": "# 历史批准稿\n\n" + sharedtextbook.RenderPracticeSet(set)}).Error)
	selected, err = service.GetPracticeEvidence(ctx, owner, chapter.ID, revision.ID, set.Tasks[0].ID)
	require.NoError(t, err)
	require.Len(t, selected.Sources, 1)
	require.Equal(t, "gin-route-mechanism", selected.Sources[0].Reference.ID)
	require.Equal(t, chunk.ID, selected.Sources[0].Reference.ChunkID)
	require.Equal(t, chunk.SearchText, selected.Sources[0].Excerpt)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	var snapshot SourceSnapshot
	require.NoError(t, tx.Where("id = ?", selected.Sources[0].Snapshot.ID).First(&snapshot).Error)
	require.NoError(t, tx.Model(&Source{}).Where("id = ?", snapshot.SourceID).Updates(map[string]any{"role": sharedtextbook.SourceRoleOfficial, "official_confirmed": true}).Error)
	official, err := NewService(NewGormRepository(tx)).GetPracticeEvidence(ctx, owner, chapter.ID, revision.ID, set.Tasks[0].ID)
	require.NoError(t, tx.Rollback().Error)
	require.NoError(t, err, "a task may cite official supporting sources only")
	require.Equal(t, sharedtextbook.SourceRoleOfficial, official.Sources[0].Snapshot.Role)
	broken := strings.Replace(string(aliased), `"gin-route-mechanism":"`+id+`"`, `"gin-route-mechanism":"evidence-missing-chunk"`, 1)
	require.NoError(t, db.Model(&ChapterRevision{}).Where("id = ?", revision.ID).Update("document_json", []byte(broken)).Error)
	_, err = service.GetPracticeEvidence(ctx, owner, chapter.ID, revision.ID, set.Tasks[0].ID)
	require.Error(t, err, "unresolvable aliases cannot supply scoring evidence")
	require.NoError(t, db.Model(&ChapterRevision{}).Where("id = ?", revision.ID).Update("document_json", aliased).Error)
	require.NoError(t, db.Model(&ChapterRevision{}).Where("id = ?", revision.ID).Update("kind", sharedtextbook.RevisionKindCandidate).Error)
	_, err = service.GetPracticeEvidence(ctx, owner, chapter.ID, revision.ID, set.Tasks[0].ID)
	require.Error(t, err, "candidate text cannot become scoring authority")
}
