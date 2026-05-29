package api

import (
	streamdomain "inkwords-backend/internal/domain/stream"
	"inkwords-backend/internal/service"
)

// StreamAPI adapts stream-related HTTP routes onto the stream domain handler.
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
