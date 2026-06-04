package generation

import (
	"context"

	"github.com/google/uuid"

	"inkwords-backend/internal/prompt"
	"inkwords-backend/internal/service"
)

// LegacyAdapter keeps the existing generator implementation reachable while service-owned use cases are being extracted.
type LegacyAdapter struct {
	legacy *service.GeneratorService
}

// NewLegacyAdapter wraps the legacy generator service for incremental migration.
func NewLegacyAdapter(legacy *service.GeneratorService) *LegacyAdapter {
	return &LegacyAdapter{legacy: legacy}
}

// Generate proxies single-article generation to the legacy implementation until llm-stream owns the full use case.
func (a *LegacyAdapter) Generate(
	ctx context.Context,
	userID uuid.UUID,
	sourceContent string,
	sourceType string,
	style string,
	chunkChan chan<- string,
	errChan chan<- error,
) {
	if a == nil || a.legacy == nil {
		return
	}

	a.legacy.GenerateBlogStream(
		ctx,
		userID,
		sourceContent,
		sourceType,
		prompt.DefaultScenarioModeForSource(sourceType),
		style,
		chunkChan,
		errChan,
	)
}
