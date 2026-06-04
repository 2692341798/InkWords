package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
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
		blogID, err := readPayloadUUID(decoded.Payload, "blog_id")
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
		blogID, err := readPayloadUUID(decoded.Payload, "blog_id")
		if err != nil {
			return err
		}
		return updateBlogByID(ctx, r.db, blogID, map[string]any{
			"content": readPayloadString(decoded.Payload, "final_content"),
		}, "update continued blog")
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
	default:
		return nil
	}

	blogID, err := readPayloadUUID(decoded.Payload, "blog_id")
	if err != nil {
		return err
	}

	var blog model.Blog
	if err := r.db.WithContext(ctx).Select("id", "user_id").First(&blog, "id = ?", blogID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("load generated blog owner: blog %s not found", blogID)
		}
		return fmt.Errorf("load generated blog owner: %w", err)
	}

	updateTx := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", blog.UserID).
		UpdateColumn("tokens_used", gorm.Expr("tokens_used + ?", decoded.Usage.EstimatedTokens))
	if updateTx.Error != nil {
		return fmt.Errorf("accumulate user tokens: %w", updateTx.Error)
	}
	if updateTx.RowsAffected == 0 {
		return fmt.Errorf("accumulate user tokens: user %s not found", blog.UserID)
	}
	return nil
}

func updateBlogByID(ctx context.Context, db *gorm.DB, blogID uuid.UUID, updates map[string]any, action string) error {
	resultTx := db.WithContext(ctx).Model(&model.Blog{}).Where("id = ?", blogID).Updates(updates)
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

func readPayloadUUID(payload map[string]any, key string) (uuid.UUID, error) {
	value := readPayloadString(payload, key)
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %s: %w", key, err)
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
