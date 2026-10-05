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

func testBookBuildFreezesVideoRunbooks(t *testing.T, db *gorm.DB, workspace, project uuid.UUID, build *BookBuildRow) {
	t.Helper()
	frozen, err := shared.ReadBookVideoRunbooks(json.RawMessage(build.ManifestJSON))
	require.NoError(t, err)
	require.NotNil(t, frozen, "new builds must freeze video guides, including absence")
	require.Len(t, frozen.Chapters, 1)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	// Mutate only the isolated test database. Old builds must never consult it.
	var revision ChapterRevision
	require.NoError(t, tx.First(&revision, "id = ?", frozen.Chapters[0].RevisionID).Error)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(revision.DocumentJSON, &doc))
	delete(doc, "video_runbook")
	if frozen.Chapters[0].VideoRunbook == nil {
		doc["video_runbook"] = json.RawMessage(validFrozenRunbookJSON)
	}
	changed, err := json.Marshal(doc)
	require.NoError(t, err)
	require.NoError(t, tx.Model(&revision).Update("document_json", string(changed)).Error)
	fresh, err := NewGormRepository(tx).CreateBookBuild(context.Background(), workspace, CreateBookBuildInput{ProjectID: project})
	require.NoError(t, err)
	require.NotEqual(t, build.ID, fresh.ID, "projection changes participate in freeze identity")
	again, err := shared.ReadBookVideoRunbooks(json.RawMessage(build.ManifestJSON))
	require.NoError(t, err)
	require.Equal(t, frozen, again)
	updated, err := shared.ReadBookVideoRunbooks(json.RawMessage(fresh.ManifestJSON))
	require.NoError(t, err)
	require.NotEqual(t, frozen.Chapters[0].RunbookHash, updated.Chapters[0].RunbookHash)
}

const validFrozenRunbookJSON = `{"format":"inkwords.video-runbook.v1","stack":"go","observation_goal":"静态结构","recommendation":{"primary":"编辑器","alternative":"其他编辑器","reason":"只读源码","manual_capture_required":true},"steps":[{"tool":"编辑器","tool_version":"实际版本","start_state":"同源文件","action":"定位节点","shortcut_or_menu":"打开文件","input":"main.go:5","expected_view":"结构可读","narration":"静态观察","capture_point":"source","recovery":"重新核对","completion_signal":"定位完成"}],"capture_checklist":["记录哈希"],"manual_capture_pending":true,"verification_status":"unverified"}`
