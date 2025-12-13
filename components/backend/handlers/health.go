package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health returns a simple health check handler
// @Summary      Health check
// @Description  Health check endpoint for monitoring backend service status
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string  "Service is healthy"
// @Router       /health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
