package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ReadinessCheck reports whether one dependency is ready to serve traffic.
type ReadinessCheck func(ctx context.Context) error

// HealthAPI centralizes the minimal liveness and readiness contract shared by all services.
type HealthAPI struct {
	serviceName string
	checks      map[string]ReadinessCheck
}

// NewGormReadinessCheck verifies that the shared GORM connection can answer a ping.
func NewGormReadinessCheck(database *gorm.DB) ReadinessCheck {
	return func(ctx context.Context) error {
		if database == nil {
			return errors.New("database is not initialized")
		}

		sqlDB, err := database.DB()
		if err != nil {
			return err
		}

		return sqlDB.PingContext(ctx)
	}
}

// NewRequiredValueCheck verifies that a required runtime value has been resolved before serving traffic.
func NewRequiredValueCheck(value string, missingMessage string) ReadinessCheck {
	return func(context.Context) error {
		if strings.TrimSpace(value) == "" {
			return errors.New(missingMessage)
		}
		return nil
	}
}

// NewHealthAPI builds a health controller with service-specific readiness checks.
func NewHealthAPI(serviceName string, checks map[string]ReadinessCheck) *HealthAPI {
	clonedChecks := make(map[string]ReadinessCheck, len(checks))
	for name, check := range checks {
		clonedChecks[name] = check
	}

	return &HealthAPI{
		serviceName: serviceName,
		checks:      clonedChecks,
	}
}

// RegisterHealthRoutes keeps legacy ping compatibility while adding standard liveness and readiness endpoints.
func RegisterHealthRoutes(engine *gin.Engine, healthAPI *HealthAPI) {
	engine.GET("/api/v1/ping", healthAPI.Ping)
	engine.GET("/health", healthAPI.Health)
	engine.GET("/ready", healthAPI.Ready)
}

// Ping preserves the legacy compatibility response already consumed by older clients and scripts.
func (h *HealthAPI) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "pong",
		"data":    nil,
	})
}

// Health reports process liveness only and does not depend on external services.
func (h *HealthAPI) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": h.serviceName,
	})
}

// Ready reports dependency readiness so Compose can wait for services that are truly ready.
func (h *HealthAPI) Ready(c *gin.Context) {
	checkResults := make(map[string]gin.H, len(h.checks))
	statusCode := http.StatusOK

	for name, check := range h.checks {
		if err := check(c.Request.Context()); err != nil {
			statusCode = http.StatusServiceUnavailable
			checkResults[name] = gin.H{
				"status": "error",
				"error":  err.Error(),
			}
			continue
		}

		checkResults[name] = gin.H{
			"status": "ok",
		}
	}

	overallStatus := "ready"
	if statusCode != http.StatusOK {
		overallStatus = "not_ready"
	}

	c.JSON(statusCode, gin.H{
		"status":  overallStatus,
		"service": h.serviceName,
		"checks":  checkResults,
	})
}
