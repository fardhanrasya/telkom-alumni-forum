package http

import (
	"net/http"

	activityService "anoa.com/telkomalumiforum/internal/modules/activity/service"
	"anoa.com/telkomalumiforum/pkg/response"
	"github.com/gin-gonic/gin"
)

type ActivityHandler struct {
	service activityService.ActivityService
}

func NewActivityHandler(service activityService.ActivityService) *ActivityHandler {
	return &ActivityHandler{service: service}
}

func (h *ActivityHandler) Heartbeat(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	streak, err := h.service.Heartbeat(c.Request.Context(), userID)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, streak)
}
