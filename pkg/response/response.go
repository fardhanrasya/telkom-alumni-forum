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

// GetOptionalUserID retrieves user ID if present, otherwise returns uuid.Nil without error
func GetOptionalUserID(c *gin.Context) uuid.UUID {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil
	}

	return userID
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
