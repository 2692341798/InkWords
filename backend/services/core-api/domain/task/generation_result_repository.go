package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// GormGenerationResultRepository persists structured generation task results
// into core-api owned business tables.
type GormGenerationResultRepository struct {
	db *gorm.DB
}

// NewGormGenerationResultRepository creates the GORM-backed repository used by core-api.
func NewGormGenerationResultRepository(db *gorm.DB) *GormGenerationResultRepository {
	return &GormGenerationResultRepository{db: db}
}

// PersistGenerationResult applies the final generation business facts to blogs.
func (r *GormGenerationResultRepository) PersistGenerationResult(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	decoded, err := decodeGenerationResult(result)
	if err != nil {
		return fmt.Errorf("decode generation result for task %s: %w", taskID, err)
	}
	switch decoded.TaskSubtype {
	case "generate_single":
		return r.persistSingleResult(ctx, taskID, decoded.Payload)
	case "continue":
		return r.persistContinuationResult(ctx, decoded.Payload)
	case "generate_series":
		return r.persistSeriesResult(ctx, decoded.Payload)
	default:
		return nil
	}
}

func (r *GormGenerationResultRepository) persistSingleResult(ctx context.Context, taskID uuid.UUID, payload map[string]any) error {
	blogID, err := readPayloadUUID(payload)
	if err != nil {
		blogID = taskID
	}
	techStacksJSON, err := marshalStringSlice(readPayloadStringSlice(payload, "tech_stacks"))
	if err != nil {
		return err
	}

	updates := map[string]any{
		"title":       readPayloadString(payload, "title"),
		"content":     readPayloadString(payload, "content"),
		"source_type": readPayloadString(payload, "source_type"),
		"word_count":  readPayloadInt(payload, "word_count"),
		"tech_stacks": datatypes.JSON(techStacksJSON),
		"status":      int16(1),
	}
	if blogID == taskID {
		if err := r.createSingleResultBlog(ctx, taskID, payload, techStacksJSON); err != nil {
			return err
		}
	}
	return updateBlogByID(ctx, r.db, blogID, updates, "update generated blog")
}

func (r *GormGenerationResultRepository) createSingleResultBlog(ctx context.Context, taskID uuid.UUID, payload map[string]any, techStacksJSON []byte) error {
	workspaceID, err := r.taskWorkspaceID(ctx, taskID)
	if err != nil {
		return err
	}
	created := blogRecord{
		ID: taskID, WorkspaceID: workspaceID, Title: readPayloadString(payload, "title"),
		Content: readPayloadString(payload, "content"), SourceType: readPayloadString(payload, "source_type"),
		WordCount: readPayloadInt(payload, "word_count"), TechStacks: datatypes.JSON(techStacksJSON), Status: 1,
	}
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).FirstOrCreate(&created).Error; err != nil {
		return fmt.Errorf("create generated blog: %w", err)
	}
	return nil
}

func (r *GormGenerationResultRepository) persistContinuationResult(ctx context.Context, payload map[string]any) error {
	blogID, err := readPayloadUUID(payload)
	if err != nil {
		return err
	}
	return updateBlogByID(ctx, r.db, blogID, map[string]any{
		"content": readPayloadString(payload, "final_content"),
	}, "update continued blog")
}

func (r *GormGenerationResultRepository) persistSeriesResult(ctx context.Context, payload map[string]any) error {
	parentRaw, ok := payload["parent_blog"].(map[string]any)
	if !ok {
		return fmt.Errorf("read parent_blog: invalid payload")
	}
	parentID, err := readPayloadUUID(parentRaw)
	if err != nil {
		return err
	}
	rawChapters, ok := payload["chapters"].([]any)
	if !ok {
		return fmt.Errorf("read chapters: invalid payload")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := updateBlogByID(ctx, tx, parentID, map[string]any{
			"title": readPayloadString(parentRaw, "title"), "content": readPayloadString(parentRaw, "content"), "status": int16(1),
		}, "update series parent blog"); err != nil {
			return err
		}
		for _, rawChapter := range rawChapters {
			if err := persistSeriesChapter(ctx, tx, rawChapter); err != nil {
				return err
			}
		}
		return nil
	})
}

func persistSeriesChapter(ctx context.Context, tx *gorm.DB, rawChapter any) error {
	chapter, ok := rawChapter.(map[string]any)
	if !ok {
		return fmt.Errorf("read chapter: invalid payload")
	}
	blogID, err := readPayloadUUID(chapter)
	if err != nil {
		return err
	}
	techStacksJSON, err := marshalStringSlice(readPayloadStringSlice(chapter, "tech_stacks"))
	if err != nil {
		return err
	}
	status := int16(1)
	if readPayloadString(chapter, "status") == "failed" {
		status = 2
	}
	return updateBlogByID(ctx, tx, blogID, map[string]any{
		"chapter_sort": readPayloadInt(chapter, "chapter_sort"), "title": readPayloadString(chapter, "title"),
		"content": readPayloadString(chapter, "content"), "word_count": readPayloadInt(chapter, "word_count"),
		"tech_stacks": datatypes.JSON(techStacksJSON), "status": status,
	}, "update series chapter blog")
}

func (r *GormGenerationResultRepository) taskWorkspaceID(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error) {
	var task JobTask
	if err := r.db.WithContext(ctx).Select("id", "workspace_id").First(&task, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, fmt.Errorf("load generation task workspace: task %s not found", taskID)
		}
		return uuid.Nil, fmt.Errorf("load generation task workspace: %w", err)
	}
	if task.WorkspaceID != nil {
		return *task.WorkspaceID, nil
	}
	return uuid.Nil, fmt.Errorf("generation task %s has no workspace", taskID)
}

func updateBlogByID(ctx context.Context, db *gorm.DB, blogID uuid.UUID, updates map[string]any, action string) error {
	resultTx := db.WithContext(ctx).Model(&blogRecord{}).Where("id = ?", blogID).Updates(updates)
	if resultTx.Error != nil {
		return fmt.Errorf("%s: %w", action, resultTx.Error)
	}
	if resultTx.RowsAffected == 0 {
		return fmt.Errorf("%s: blog %s not found", action, blogID)
	}
	return nil
}

func decodeGenerationResult(result map[string]any) (GenerationResult, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return GenerationResult{}, fmt.Errorf("marshal result: %w", err)
	}

	var decoded GenerationResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return GenerationResult{}, fmt.Errorf("unmarshal result: %w", err)
	}
	return decoded, nil
}

func readPayloadUUID(payload map[string]any) (uuid.UUID, error) {
	value := readPayloadString(payload, "blog_id")
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse blog_id: %w", err)
	}
	return parsed, nil
}

func readPayloadString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func readPayloadInt(payload map[string]any, key string) int {
	switch value := payload[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func readPayloadStringSlice(payload map[string]any, key string) []string {
	rawItems, ok := payload[key].([]any)
	if ok {
		items := make([]string, 0, len(rawItems))
		for _, rawItem := range rawItems {
			if item, ok := rawItem.(string); ok {
				items = append(items, item)
			}
		}
		return items
	}

	if direct, ok := payload[key].([]string); ok {
		return append([]string(nil), direct...)
	}
	return []string{}
}

func marshalStringSlice(items []string) ([]byte, error) {
	raw, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal tech_stacks: %w", err)
	}
	return raw, nil
}

var _ BlogResultRepository = (*GormGenerationResultRepository)(nil)
