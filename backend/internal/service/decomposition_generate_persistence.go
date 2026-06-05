package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func (s *DecompositionService) ensureSeriesParentAndDrafts(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	parentTitle string,
	sourceType string,
	gitURL string,
	outline []Chapter,
) ([]Chapter, error) {
	return s.seriesPersistence.EnsureSeriesParentAndDrafts(ctx, SeriesDraftPreflightInput{
		UserID:      userID,
		ParentID:    parentID,
		ParentTitle: parentTitle,
		SourceType:  sourceType,
		GitURL:      gitURL,
		Outline:     outline,
	})
}

func collectValidSeriesChildIDs(outline []Chapter) []uuid.UUID {
	validChildrenIDs := make([]uuid.UUID, 0, len(outline))
	for _, chapter := range outline {
		if chapter.ID == "" {
			continue
		}
		if id, err := uuid.Parse(chapter.ID); err == nil {
			validChildrenIDs = append(validChildrenIDs, id)
		}
	}

	return validChildrenIDs
}

func (s *DecompositionService) handleSeriesChapterCompletion(
	ctx context.Context,
	userID uuid.UUID,
	parentID uuid.UUID,
	sourceType string,
	chapter Chapter,
	content string,
	wordCount int,
	techStacks []string,
	collector *seriesTaskResultCollector,
) error {
	// Why: task_only 模式下系列章节的最终业务事实必须先收口进 result_json，
	// 否则 core-api 无法在任务成功路径里一次性接管父子博客持久化。
	if taskOnlyPersistenceMode() {
		collector.AddChapterSuccess(chapter, content, wordCount, techStacks)
		return nil
	}

	techStacksJSON, err := json.Marshal(techStacks)
	if err != nil {
		return fmt.Errorf("marshal series chapter tech stacks: %w", err)
	}

	var blogID uuid.UUID
	if chapter.ID != "" {
		parsedID, err := uuid.Parse(chapter.ID)
		if err != nil {
			return fmt.Errorf("parse chapter blog id: %w", err)
		}
		blogID = parsedID
	}

	return s.seriesPersistence.SaveSeriesChapter(ctx, SeriesChapterPersistenceInput{
		UserID:     userID,
		ParentID:   parentID,
		BlogID:     blogID,
		Chapter:    chapter,
		SourceType: sourceType,
		Content:    content,
		WordCount:  wordCount,
		TechStacks: datatypes.JSON(techStacksJSON),
	})
}

func (s *DecompositionService) handleSeriesChapterFailure(
	ctx context.Context,
	userID uuid.UUID,
	chapter Chapter,
	streamErr error,
	collector *seriesTaskResultCollector,
) {
	if taskOnlyPersistenceMode() {
		collector.AddChapterFailure(chapter, streamErr.Error())
		return
	}

	if chapter.ID == "" {
		return
	}

	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return
	}

	_ = s.seriesPersistence.MarkSeriesChapterFailed(ctx, userID, blogID)
}

func (s *DecompositionService) handleSkippedSeriesChapter(ctx context.Context, userID uuid.UUID, chapter Chapter) error {
	if strings.TrimSpace(chapter.ID) == "" {
		return nil
	}

	blogID, err := uuid.Parse(chapter.ID)
	if err != nil {
		return nil
	}

	return s.seriesPersistence.UpdateSkippedSeriesChapterMeta(ctx, userID, blogID, chapter)
}

func decodeTechStacksJSON(raw datatypes.JSON) []string {
	var techStacks []string
	if len(raw) == 0 {
		return []string{}
	}
	if err := json.Unmarshal(raw, &techStacks); err != nil {
		return []string{}
	}
	return techStacks
}
