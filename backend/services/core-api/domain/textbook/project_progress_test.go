package textbook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func assertProgressKeepsApprovedManuscriptWithMissingSampleProfile(t *testing.T, database *gorm.DB, workspaceID, projectID, contractID uuid.UUID) {
	t.Helper()
	t.Run("approved manuscript remains visible while sample profile coverage is incomplete", func(t *testing.T) {
		// Model the two-profile approved contract in an isolated fixture transaction.
		// Production contract revisions remain immutable.
		tx := database.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()
		var contract BookContractRevision
		require.NoError(t, tx.First(&contract, "id = ?", contractID).Error)
		var document sharedtextbook.BookContract
		require.NoError(t, json.Unmarshal(contract.DocumentJSON, &document))
		document.ChapterProfiles = append(document.ChapterProfiles, sharedtextbook.ChapterProfileHandsOn)
		require.NoError(t, document.Validate())
		data, err := json.Marshal(document)
		require.NoError(t, err)
		require.NoError(t, tx.Model(&contract).Update("document_json", data).Error)

		progress, err := NewGormRepository(tx).GetProjectProgress(context.Background(), workspaceID, projectID)
		require.NoError(t, err)
		require.Equal(t, ProjectStageState{Key: "sample", Status: "in_progress", Reason: "已有 1 章人工批准的样章，但仍缺少 动手实践 画像的代表性样章。"}, progress.Stages[2])
		require.Equal(t, ProjectStageState{Key: "chapters", Status: "in_progress", Reason: "已有 1 章人工批准的母稿修订；仍缺少 动手实践 画像的代表性样章，批量生产尚未就绪。"}, progress.Stages[3])
		require.Equal(t, "ready", progress.Stages[5].Status, "existing approved material remains available for learning")
	})
}
