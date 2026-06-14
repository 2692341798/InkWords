package api

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"inkwords-backend/shared/kernel/httpx"
)

// ReadinessCheck reports whether one dependency is ready to serve traffic.
type ReadinessCheck = httpx.ReadinessCheck

// HealthAPI centralizes the minimal liveness and readiness contract shared by all services.
type HealthAPI = httpx.HealthAPI

// NewGormReadinessCheck verifies that the shared GORM connection can answer a ping.
func NewGormReadinessCheck(database *gorm.DB) ReadinessCheck {
	return httpx.NewGormReadinessCheck(database)
}

// NewRequiredValueCheck verifies that a required runtime value has been resolved before serving traffic.
func NewRequiredValueCheck(value string, missingMessage string) ReadinessCheck {
	return httpx.NewRequiredValueCheck(value, missingMessage)
}

// NewHealthAPI builds a health controller with service-specific readiness checks.
func NewHealthAPI(serviceName string, checks map[string]ReadinessCheck) *HealthAPI {
	return httpx.NewHealthAPI(serviceName, checks)
}

// RegisterHealthRoutes keeps legacy ping compatibility while adding standard liveness and readiness endpoints.
func RegisterHealthRoutes(engine *gin.Engine, healthAPI *HealthAPI) {
	httpx.RegisterHealthRoutes(engine, healthAPI)
}
