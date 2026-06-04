package api

import (
	streamdomain "inkwords-backend/internal/domain/stream"
	"inkwords-backend/internal/service"
)

// Deprecated: llm-stream now owns its primary route registration in services/llm-stream.
// StreamAPI remains as a transitional adapter while generation and decomposition use cases are being migrated.
type StreamAPI struct {
	generatorService     *service.GeneratorService
	decompositionService *service.DecompositionService
	userService          *service.UserService
	streamDomainHandler  *streamdomain.Handler
}

// NewStreamAPIWithDeps creates a StreamAPI with explicitly injected dependencies.
func NewStreamAPIWithDeps(generatorService *service.GeneratorService, decompositionService *service.DecompositionService, userService *service.UserService, streamDomainHandler *streamdomain.Handler) *StreamAPI {
	return &StreamAPI{
		generatorService:     generatorService,
		decompositionService: decompositionService,
		userService:          userService,
		streamDomainHandler:  streamDomainHandler,
	}
}
