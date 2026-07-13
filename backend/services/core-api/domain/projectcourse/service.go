package projectcourse

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, input CreateInput) (*ProjectCourse, error) {
	if input.UserID == uuid.Nil || strings.TrimSpace(input.RepositoryURL) == "" || strings.TrimSpace(input.RequestedRef) == "" {
		return nil, fmt.Errorf("user, repository URL and requested ref are required")
	}
	if len(input.ResolvedCommitSHA) != 40 {
		return nil, fmt.Errorf("resolved commit SHA must be 40 characters")
	}
	if _, err := hex.DecodeString(input.ResolvedCommitSHA); err != nil {
		return nil, fmt.Errorf("resolved commit SHA must be hexadecimal")
	}
	course := &ProjectCourse{UserID: input.UserID, RepositoryURL: strings.TrimSpace(input.RepositoryURL), RequestedRef: strings.TrimSpace(input.RequestedRef), ResolvedCommitSHA: strings.ToLower(input.ResolvedCommitSHA), AudienceLevel: input.AudienceLevel, Status: StatusAwaitingApproval, BlueprintVersion: 1, BlueprintJSON: []byte(`{}`), CoverageJSON: []byte(`{}`), QualityReportJSON: []byte(`{}`)}
	if err := s.repository.Create(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (s *Service) Get(ctx context.Context, userID, courseID uuid.UUID) (*ProjectCourse, error) {
	return s.repository.GetByID(ctx, userID, courseID)
}

func (s *Service) UpdateBlueprint(ctx context.Context, userID, courseID uuid.UUID, update BlueprintUpdate) error {
	if update.ExpectedVersion < 1 || len(update.BlueprintJSON) == 0 || len(update.CoverageJSON) == 0 {
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
