package export

import (
	"context"
	"errors"

	"github.com/google/uuid"

	blogcontracts "inkwords-backend/internal/domain/blog/contracts"
	legacyservice "inkwords-backend/internal/service"
)

// ErrSeriesNotFound re-exports the shared blog export error so callers can stay scoped to export-service.
var ErrSeriesNotFound = blogcontracts.ErrSeriesNotFound

// Service wraps the legacy blog export implementation behind a service-owned boundary.
type Service struct {
	legacy *legacyservice.BlogService
}

// NewService creates an export-service scoped adapter for the existing export implementation.
func NewService(legacy *legacyservice.BlogService) *Service {
	return &Service{legacy: legacy}
}

// ExportSeriesToPDF delegates PDF export to the existing blog service while keeping the dependency local to export-service.
func (s *Service) ExportSeriesToPDF(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) (string, string, error) {
	if s == nil || s.legacy == nil {
		return "", "", errors.New("export service is not configured")
	}
	return s.legacy.ExportSeriesToPDF(ctx, blogID, userID)
}

// ExportToObsidian delegates single-blog Obsidian export to the existing implementation.
func (s *Service) ExportToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error {
	if s == nil || s.legacy == nil {
		return errors.New("export service is not configured")
	}
	return s.legacy.ExportToObsidian(ctx, blogID, userID)
}

// ExportSeriesToObsidian delegates series export to Obsidian to the existing implementation.
func (s *Service) ExportSeriesToObsidian(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) error {
	if s == nil || s.legacy == nil {
		return errors.New("export service is not configured")
	}
	return s.legacy.ExportSeriesToObsidian(ctx, blogID, userID)
}
