package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(db *gorm.DB, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// CheckHealth handles the /healthz endpoint
// @Summary Health check
// @Description Check the health status of the API service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Service is healthy"
// @Failure 503 {object} map[string]interface{} "Service is unhealthy"
// @Router /healthz [get]
func (h *HealthHandler) CheckHealth(c *gin.Context) {
	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		h.logger.Error("Failed to get database connection", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Database connection error",
		})
		return
	}

	// Ping the database
	if err := sqlDB.Ping(); err != nil {
		h.logger.Error("Failed to ping database", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Database ping failed",
		})
		return
	}

	// All checks passed
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Service is healthy",
	})
}
