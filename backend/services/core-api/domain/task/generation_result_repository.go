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
		blogID, err := readPayloadUUID(decoded.Payload)
		if err != nil {
			return err
		}
		techStacksJSON, err := marshalStringSlice(readPayloadStringSlice(decoded.Payload, "tech_stacks"))
		if err != nil {
			return err
		}

		updates := map[string]any{
			"title":       readPayloadString(decoded.Payload, "title"),
			"content":     readPayloadString(decoded.Payload, "content"),
			"source_type": readPayloadString(decoded.Payload, "source_type"),
			"word_count":  readPayloadInt(decoded.Payload, "word_count"),
			"tech_stacks": datatypes.JSON(techStacksJSON),
			"status":      int16(1),
		}
		return updateBlogByID(ctx, r.db, blogID, updates, "update generated blog")
	case "continue":
		blogID, err := readPayloadUUID(decoded.Payload)
		if err != nil {
			return err
		}
		return updateBlogByID(ctx, r.db, blogID, map[string]any{
			"content": readPayloadString(decoded.Payload, "final_content"),
		}, "update continued blog")
	case "generate_series":
		parentRaw, ok := decoded.Payload["parent_blog"].(map[string]any)
		if !ok {
			return fmt.Errorf("read parent_blog: invalid payload")
		}
		parentID, err := readPayloadUUID(parentRaw)
		if err != nil {
			return err
		}

		rawChapters, ok := decoded.Payload["chapters"].([]any)
		if !ok {
			return fmt.Errorf("read chapters: invalid payload")
		}

		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := updateBlogByID(ctx, tx, parentID, map[string]any{
				"title":   readPayloadString(parentRaw, "title"),
				"content": readPayloadString(parentRaw, "content"),
				"status":  int16(1),
			}, "update series parent blog"); err != nil {
				return err
			}

			for _, rawChapter := range rawChapters {
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

				if err := updateBlogByID(ctx, tx, blogID, map[string]any{
					"chapter_sort": readPayloadInt(chapter, "chapter_sort"),
					"title":        readPayloadString(chapter, "title"),
					"content":      readPayloadString(chapter, "content"),
					"word_count":   readPayloadInt(chapter, "word_count"),
					"tech_stacks":  datatypes.JSON(techStacksJSON),
					"status":       status,
				}, "update series chapter blog"); err != nil {
					return err
				}
			}

			return nil
		})
	default:
		return nil
	}
}

// AccumulateTokens applies token accounting after blogs have been updated.
func (r *GormGenerationResultRepository) AccumulateTokens(ctx context.Context, taskID uuid.UUID, result map[string]any) error {
	decoded, err := decodeGenerationResult(result)
	if err != nil {
		return fmt.Errorf("decode generation result for task %s: %w", taskID, err)
	}
	switch decoded.TaskSubtype {
	case "generate_single", "continue":
	case "generate_series":
	default:
		return nil
	}

	userID, err := r.usageOwnerUserID(ctx, taskID, decoded)
	if err != nil {
		return err
	}

	updateTx := r.db.WithContext(ctx).Model(&userRecord{}).
		Where("id = ?", userID).
		UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", decoded.Usage.billableTokens()))
	if updateTx.Error != nil {
		return fmt.Errorf("accumulate user tokens: %w", updateTx.Error)
	}
	if updateTx.RowsAffected == 0 {
		return fmt.Errorf("accumulate user tokens: user %s not found", userID)
	}
	return nil
}

func (r *GormGenerationResultRepository) usageOwnerUserID(ctx context.Context, taskID uuid.UUID, decoded GenerationResult) (uuid.UUID, error) {
	blogID, err := usageOwnerBlogID(decoded)
	if err == nil {
		var blog blogRecord
		if err := r.db.WithContext(ctx).Select("id", "user_id").First(&blog, "id = ?", blogID).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return uuid.Nil, fmt.Errorf("load generated blog owner: %w", err)
			}
		} else {
			return blog.UserID, nil
		}
	}

	var task JobTask
	if err := r.db.WithContext(ctx).Select("id", "requested_by").First(&task, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, fmt.Errorf("load generation task owner: task %s not found", taskID)
		}
		return uuid.Nil, fmt.Errorf("load generation task owner: %w", err)
	}
	return task.RequestedBy, nil
}

func usageOwnerBlogID(decoded GenerationResult) (uuid.UUID, error) {
	if decoded.TaskSubtype == "generate_series" {
		parentRaw, ok := decoded.Payload["parent_blog"].(map[string]any)
		if !ok {
			return uuid.Nil, fmt.Errorf("read parent_blog: invalid payload")
		}
		return readPayloadUUID(parentRaw)
	}

	return readPayloadUUID(decoded.Payload)
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
var _ UsageRepository = (*GormGenerationResultRepository)(nil)
