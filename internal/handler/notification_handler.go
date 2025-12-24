package handler

import (
	"fmt"
	"log"
	"net/http"

	"anoa.com/telkomalumiforum/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type NotificationHandler struct {
	service     service.NotificationService
	redisClient *redis.Client
	upgrader    websocket.Upgrader
}

func NewNotificationHandler(service service.NotificationService, redisClient *redis.Client) *NotificationHandler {
	return &NotificationHandler{
		service:     service,
		redisClient: redisClient,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// REST Endpoints

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	limit := 20
	offset := 0
	// TODO: Add query params for limit/offset

	notifications, err := h.service.GetNotifications(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": notifications})
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.service.MarkAsRead(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.service.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	count, err := h.service.UnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// WebSocket Endpoint

func (h *NotificationHandler) HandleWebSocket(c *gin.Context) {
	// For WebSocket, we might need to get token from Query param if not in header
	// But assuming the middleware handles it or we handle it here if middleware fails
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		// Fallback: check query param 'token' if middleware didn't set user_id
		// Ideally middleware should be improved, but for now specific WS auth:
		// NOTE: In a real app, don't reimplement auth logic here.
		// We assume the route is protected by middleware.
		// If middleware fails for WS due to missing headers, the request won't reach here.
		// So we advise client to pass token in header (some libs allow it) or
		// we modify middleware to support query param.

		// For now, assuming middleware put user_id in context.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket: %v", err)
		return
	}
	defer conn.Close()

	// Redis Subscription
	if h.redisClient == nil {
		// If no Redis, simple fallback or close
		log.Println("Redis client is nil, cannot subscribe")
		return
	}

	channel := fmt.Sprintf("user_notifications:%s", userIDStr)
	pubsub := h.redisClient.Subscribe(c.Request.Context(), channel)
	defer pubsub.Close()

	// Wait for confirmation that subscription is created
	_, err = pubsub.Receive(c.Request.Context())
	if err != nil {
		log.Printf("Failed to subscribe to redis channel: %v", err)
		return
	}

	ch := pubsub.Channel()

	// Handle disconnect properly
	// Create a channel to signal client disconnect
	clientClosed := make(chan struct{})

	go func() {
		defer close(clientClosed)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Client disconnected or error
				return
			}
		}
	}()

	// Loop to send messages from Redis to WS
	for {
		select {
		case msg := <-ch:
			// Msg from Redis
			// payload is JSON string of Notification
			// We can just forward it directly

			// Optional: Unmarshal to verify/manipulate?
			// For performance, just writing message is faster if format is already JSON

			err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			if err != nil {
				log.Printf("Failed to write message to websocket: %v", err)
				return
			}
		case <-clientClosed:
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}
