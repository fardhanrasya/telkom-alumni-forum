package response

import (
	"log"
	"net/http"

	"anoa.com/telkomalumiforum/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUserID retrieves the authenticated user ID from the context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, apperror.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil, apperror.ErrUnauthorized
	}

	return userID, nil
}

// ResponseError standardized error response
func ResponseError(c *gin.Context, err error) {
	code := apperror.MapErrorToStatus(err)
	
	// Log internal errors
	if code == http.StatusInternalServerError {
		log.Printf("[Internal Error]: %v", err)
	}

	c.JSON(code, gin.H{"error": err.Error()})
}
