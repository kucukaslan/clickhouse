package api

import (
	"context"
	"kucukaslan/clickhouse/domain"
	"time"

	"kucukaslan/clickhouse/buildinfo"
	"kucukaslan/clickhouse/database"

	"github.com/gofiber/fiber/v2"
)

// HealthCheck handles the /health endpoint
// @Summary Health check endpoint
// @Description Check the health status of the service and its dependencies
// @Tags Health
// @Produce json
// @Success 200 {object} domain.HealthResponse "Service is healthy"
// @Success 503 {object} domain.HealthResponse "Service is unhealthy"
// @Router /health [get]
func HealthCheck(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response := domain.HealthResponse{
		Timestamp: time.Now(),
		BuildInfo: buildinfo.GetInfo(),
		Services:  domain.ServiceHealthStatus{},
	}

	// Check ClickHouse health
	clickhouseHealthy := true
	if err := database.ClickHouseHealthCheck(ctx); err != nil {
		clickhouseHealthy = false
		response.Services.ClickHouse = domain.ServiceStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	} else {
		response.Services.ClickHouse = domain.ServiceStatus{
			Status: "healthy",
		}
	}

	// Check Redis health
	redisHealthy := true
	if err := database.RedisHealthCheck(ctx); err != nil {
		redisHealthy = false
		response.Services.Redis = domain.ServiceStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	} else {
		response.Services.Redis = domain.ServiceStatus{
			Status: "healthy",
		}
	}

	// Determine overall status
	if clickhouseHealthy && redisHealthy {
		response.Status = "healthy"
		return c.Status(fiber.StatusOK).JSON(response)
	}

	response.Status = "unhealthy"
	return c.Status(fiber.StatusServiceUnavailable).JSON(response)
}
