package textbook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func assertBlueprintChapterMembership(t *testing.T, db *gorm.DB, workspaceID, projectID uuid.UUID, approved *BlueprintRevision) {
	t.Helper()
	for _, name := range []string{"missing", "other project", "soft deleted", "malformed", "noncanonical"} {
		t.Run("blueprint chapter membership/"+name, func(t *testing.T) {
			// Each case uses a rollback-only fixture so the approved baseline is untouched.
			tx := db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			repo := NewGormRepository(tx)
			ctx := context.Background()
			var doc sharedtextbook.Blueprint
			require.NoError(t, json.Unmarshal(approved.DocumentJSON, &doc))
			switch name {
			case "missing":
				doc.Volumes[0].Chapters[0].ID = uuid.NewString()
			case "other project":
				other, err := repo.CreateProject(ctx, CreateProjectInput{WorkspaceID: workspaceID, Title: "其它项目", Audience: sharedtextbook.AudienceFoundation, Primary: CreateSourceInput{Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin"}})
				require.NoError(t, err)
				chapter, err := repo.CreateChapter(ctx, workspaceID, CreateChapterInput{ProjectID: other.ID, SortOrder: 1, Title: "其它项目章节", ChapterProfile: sharedtextbook.ChapterProfileConcept})
				require.NoError(t, err)
				doc.Volumes[0].Chapters[0].ID = chapter.ID.String()
			case "soft deleted":
				require.NoError(t, tx.Delete(&Chapter{}, "id = ?", doc.Volumes[0].Chapters[0].ID).Error)
			case "malformed":
				doc.Volumes[0].Chapters[0].ID = "future-chapter"
			case "noncanonical":
				doc.Volumes[0].Chapters[0].ID = "urn:uuid:" + doc.Volumes[0].Chapters[0].ID
			}
			var before int64
			require.NoError(t, tx.Model(&BlueprintRevision{}).Where("project_id = ?", projectID).Count(&before).Error)
			_, err := repo.CreateBlueprint(ctx, workspaceID, CreateBlueprintInput{ProjectID: projectID, Volumes: doc.Volumes})
			assert.ErrorIs(t, err, ErrInvalidState, "draft creation must reject invisible chapter references")
			var after int64
			require.NoError(t, tx.Model(&BlueprintRevision{}).Where("project_id = ?", projectID).Count(&after).Error)
			assert.Equal(t, before, after, "rejected creation must not save a draft")

			// Simulate a draft saved by the earlier API, with valid identity and content hash.
			doc.RevisionID = uuid.NewString()
			doc.RevisionNumber = approved.RevisionNumber + 2
			doc.ContentHash = digestJSON(blueprintPayload(uuid.MustParse(doc.BookContractRevision), uuid.MustParse(doc.StyleSheetRevision), doc.Volumes))
			require.NoError(t, doc.ValidateGenerationReadiness())
			encoded, err := json.Marshal(doc)
			require.NoError(t, err)
			legacy := BlueprintRevision{ID: uuid.MustParse(doc.RevisionID), ProjectID: projectID, RevisionNumber: doc.RevisionNumber, DocumentJSON: datatypes.JSON(encoded), ContentHash: strings.TrimPrefix(doc.ContentHash, "sha256:"), Status: StatusDraft}
			require.NoError(t, tx.Create(&legacy).Error)
			_, err = repo.ApproveBlueprint(ctx, workspaceID, projectID, legacy.ID)
			require.ErrorIs(t, err, ErrInvalidState, "approval must recheck legacy draft chapter membership")
			require.NoError(t, tx.First(&legacy, "id = ?", legacy.ID).Error)
			require.Equal(t, StatusDraft, legacy.Status)
			var project Project
			require.NoError(t, tx.First(&project, "id = ?", projectID).Error)
			require.Equal(t, approved.ID, *project.ApprovedBlueprintRevisionID)
		})
	}
	t.Run("blueprint chapter membership/valid multiple volumes", func(t *testing.T) {
		tx := db.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()
		repo := NewGormRepository(tx)
		ctx := context.Background()
		chapter, err := repo.CreateChapter(ctx, workspaceID, CreateChapterInput{ProjectID: projectID, SortOrder: 2, Title: "原章节标题", ChapterProfile: sharedtextbook.ChapterProfileConcept})
		require.NoError(t, err)
		var doc sharedtextbook.Blueprint
		require.NoError(t, json.Unmarshal(approved.DocumentJSON, &doc))
		second := doc.Volumes[0].Chapters[0]
		second.ID, second.Title = chapter.ID.String(), "蓝图内的新标题"
		second.PrerequisiteIDs = []string{doc.Volumes[0].Chapters[0].ID}
		doc.Volumes = append(doc.Volumes, sharedtextbook.BlueprintVolume{ID: "second-volume", Title: "实践", Sort: 2, Chapters: []sharedtextbook.BlueprintChapter{second}})
		draft, err := repo.CreateBlueprint(ctx, workspaceID, CreateBlueprintInput{ProjectID: projectID, Volumes: doc.Volumes})
		require.NoError(t, err)
		result, err := repo.ApproveBlueprint(ctx, workspaceID, projectID, draft.ID)
		require.NoError(t, err)
		require.Equal(t, StatusApproved, result.Status)
		var plan []string
		require.NoError(t, tx.Raw("EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM textbook_chapters WHERE project_id = ? AND id IN ? AND deleted_at IS NULL", projectID, []uuid.UUID{uuid.MustParse(doc.Volumes[0].Chapters[0].ID), chapter.ID}).Scan(&plan).Error)
		t.Log("membership query plan:\n" + strings.Join(plan, "\n"))
	})
}
