package projectcourse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type Service struct{ repository Repository }

type BlueprintPreview struct {
	Coverage sharedkernel.CoverageMatrix `json:"coverage"`
	Status   string                      `json:"status"`
}

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, input CreateInput) (*ProjectCourse, error) {
	if input.UserID == uuid.Nil || strings.TrimSpace(input.RepositoryURL) == "" || strings.TrimSpace(input.RequestedRef) == "" {
		return nil, fmt.Errorf("user, repository URL and requested ref are required")
	}
	if err := sharedkernel.AudienceLevel(input.AudienceLevel).Validate(); err != nil {
		return nil, err
	}
	course := &ProjectCourse{UserID: input.UserID, RepositoryURL: strings.TrimSpace(input.RepositoryURL), RequestedRef: strings.TrimSpace(input.RequestedRef), AudienceLevel: input.AudienceLevel, Status: StatusAnalyzing, BlueprintVersion: 1, BlueprintJSON: []byte(`{}`), CoverageJSON: []byte(`{}`), QualityReportJSON: []byte(`{}`)}
	if err := s.repository.Create(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (s *Service) Get(ctx context.Context, userID, courseID uuid.UUID) (*ProjectCourse, error) {
	return s.repository.GetByID(ctx, userID, courseID)
}

func (s *Service) UpdateBlueprint(ctx context.Context, userID, courseID uuid.UUID, update BlueprintUpdate) error {
	if update.ExpectedVersion < 1 {
		return fmt.Errorf("invalid blueprint update")
	}
	if len(update.ChapterUpdates) > 0 {
		current, err := s.repository.GetByID(ctx, userID, courseID)
		if err != nil {
			return err
		}
		var blueprint sharedkernel.Blueprint
		if err := json.Unmarshal(current.BlueprintJSON, &blueprint); err != nil {
			return fmt.Errorf("decode current blueprint: %w", err)
		}
		if err := blueprint.Validate(); err != nil {
			return fmt.Errorf("current blueprint is invalid: %w", err)
		}
		known := make(map[string]bool)
		for _, update := range update.ChapterUpdates {
			if strings.TrimSpace(update.ChapterID) == "" || strings.TrimSpace(update.Title) == "" || update.Sort < 1 || known[update.ChapterID] {
				return fmt.Errorf("invalid or duplicate chapter update")
			}
			known[update.ChapterID] = true
			found := false
			for volumeIndex := range blueprint.Volumes {
				for chapterIndex := range blueprint.Volumes[volumeIndex].Chapters {
					chapter := &blueprint.Volumes[volumeIndex].Chapters[chapterIndex]
					if chapter.ID == update.ChapterID {
						chapter.Title, chapter.Sort, chapter.Enabled = strings.TrimSpace(update.Title), update.Sort, update.Enabled
						found = true
					}
				}
			}
			if !found {
				return fmt.Errorf("chapter %q does not exist", update.ChapterID)
			}
		}
		blueprint.BlueprintVersion = update.ExpectedVersion + 1
		encoded, err := json.Marshal(blueprint)
		if err != nil {
			return err
		}
		update.BlueprintJSON = encoded
		if len(current.CoverageJSON) > 0 {
			update.CoverageJSON = current.CoverageJSON
		} else {
			update.CoverageJSON = []byte(`{}`)
		}
	}
	if len(update.BlueprintJSON) == 0 || len(update.CoverageJSON) == 0 {
		return fmt.Errorf("invalid blueprint update")
	}
	return s.repository.UpdateBlueprintCAS(ctx, userID, courseID, update)
}

func (s *Service) Approve(ctx context.Context, userID, courseID uuid.UUID, expectedVersion int) error {
	if expectedVersion < 1 {
		return fmt.Errorf("expected version must be positive")
	}
	return s.repository.Approve(ctx, userID, courseID, expectedVersion)
}

func (s *Service) PreviewBlueprint(ctx context.Context, userID, courseID uuid.UUID, update BlueprintUpdate) (*BlueprintPreview, error) {
	if update.ExpectedVersion < 1 {
		return nil, fmt.Errorf("invalid blueprint update")
	}
	current, err := s.repository.GetByID(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	var blueprint sharedkernel.Blueprint
	if err := json.Unmarshal(current.BlueprintJSON, &blueprint); err != nil {
		return nil, fmt.Errorf("decode current blueprint: %w", err)
	}
	if err := blueprint.Validate(); err != nil {
		return nil, fmt.Errorf("current blueprint is invalid: %w", err)
	}
	known := make(map[string]bool)
	enabled := make(map[string]bool)
	for _, volume := range blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			enabled[chapter.ID] = chapter.Enabled
		}
	}
	for _, update := range update.ChapterUpdates {
		if strings.TrimSpace(update.ChapterID) == "" || strings.TrimSpace(update.Title) == "" || update.Sort < 1 || known[update.ChapterID] {
			return nil, fmt.Errorf("invalid or duplicate chapter update")
		}
		known[update.ChapterID] = true
		if _, exists := enabled[update.ChapterID]; !exists {
			return nil, fmt.Errorf("chapter %q does not exist", update.ChapterID)
		}
		enabled[update.ChapterID] = update.Enabled
	}
	var coverage sharedkernel.CoverageMatrix
	if err := json.Unmarshal(current.CoverageJSON, &coverage); err != nil {
		return nil, fmt.Errorf("decode current coverage: %w", err)
	}
	apply := func(items []sharedkernel.CoverageItem) {
		for index := range items {
			items[index].Covered = false
			for _, chapterID := range items[index].ChapterIDs {
				if enabled[chapterID] {
					items[index].Covered = true
					break
				}
			}
		}
	}
	apply(coverage.Modules)
	apply(coverage.MainFlows)
	apply(coverage.Technologies)
	apply(coverage.Files)
	status := "complete_coverage"
	for _, items := range [][]sharedkernel.CoverageItem{coverage.Modules, coverage.MainFlows, coverage.Technologies, coverage.Files} {
		for _, item := range items {
			if !item.Covered {
				status = "customized_coverage"
			}
		}
	}
	return &BlueprintPreview{Coverage: coverage, Status: status}, nil
}
