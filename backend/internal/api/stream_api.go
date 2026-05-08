package api

import (
	streamdomain "inkwords-backend/internal/domain/stream"
	"inkwords-backend/internal/service"
)

type StreamAPI struct {
	generatorService     *service.GeneratorService
	decompositionService *service.DecompositionService
	userService          *service.UserService
	streamDomainHandler  *streamdomain.Handler
}

func NewStreamAPI(userService *service.UserService) *StreamAPI {
	generatorService := service.NewGeneratorService()
	decompositionService := service.NewDecompositionService()
	streamService := streamdomain.NewService(generatorService, decompositionService, userService)
	return NewStreamAPIWithDeps(generatorService, decompositionService, userService, streamdomain.NewHandler(streamService))
}

func NewStreamAPIWithDeps(generatorService *service.GeneratorService, decompositionService *service.DecompositionService, userService *service.UserService, streamDomainHandler *streamdomain.Handler) *StreamAPI {
	return &StreamAPI{
		generatorService:     generatorService,
		decompositionService: decompositionService,
		userService:          userService,
		streamDomainHandler:  streamDomainHandler,
	}
}
