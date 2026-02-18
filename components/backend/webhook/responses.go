package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	DeliveryID string `json:"delivery_id,omitempty"`
}

// SuccessResponse represents a JSON success response
type SuccessResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	DeliveryID string `json:"delivery_id,omitempty"`
}

// RespondWithError sends a JSON error response (FR-003, FR-009, FR-010)
func RespondWithError(c *gin.Context, statusCode int, errorType string, message string, deliveryID string) {
	c.JSON(statusCode, ErrorResponse{
		Error:      errorType,
		Message:    message,
		DeliveryID: deliveryID,
	})
}

// RespondWithSuccess sends a JSON success response (FR-009)
func RespondWithSuccess(c *gin.Context, message string, deliveryID string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Status:     "success",
		Message:    message,
		DeliveryID: deliveryID,
	})
}

// RespondUnauthorized sends a 401 Unauthorized response (FR-003)
func RespondUnauthorized(c *gin.Context, message string, deliveryID string) {
	RespondWithError(c, http.StatusUnauthorized, "unauthorized", message, deliveryID)
}

// RespondBadRequest sends a 400 Bad Request response (FR-010)
func RespondBadRequest(c *gin.Context, message string, deliveryID string) {
	RespondWithError(c, http.StatusBadRequest, "bad_request", message, deliveryID)
}

// RespondPayloadTooLarge sends a 413 Payload Too Large response (FR-005)
func RespondPayloadTooLarge(c *gin.Context, deliveryID string) {
	RespondWithError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "Webhook payload exceeds 10MB limit", deliveryID)
}

// RespondInternalServerError sends a 500 Internal Server Error response
func RespondInternalServerError(c *gin.Context, message string, deliveryID string) {
	RespondWithError(c, http.StatusInternalServerError, "internal_server_error", message, deliveryID)
}

// RespondAccepted sends a 202 Accepted response (for async processing fallback)
func RespondAccepted(c *gin.Context, message string, deliveryID string) {
	c.JSON(http.StatusAccepted, SuccessResponse{
		Status:     "accepted",
		Message:    message,
		DeliveryID: deliveryID,
	})
}
