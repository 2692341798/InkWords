package api

import "inkwords-backend/internal/service"

type StreamAPI struct {
	generatorService     *service.GeneratorService
	decompositionService *service.DecompositionService
	userService          *service.UserService
}

func NewStreamAPI(userService *service.UserService) *StreamAPI {
	return &StreamAPI{
		generatorService:     service.NewGeneratorService(),
		decompositionService: service.NewDecompositionService(),
		userService:          userService,
	}
}

