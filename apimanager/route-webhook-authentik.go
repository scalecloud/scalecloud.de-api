package apimanager

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

const authentikWebhookMaxBodySize = 1 << 20

func (api *Api) authentikWebhookRequired(c *gin.Context) {
	token := api.webhookHandler.AuthentikWebhookToken
	if token == "" {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "Webhook unavailable"})
		return
	}

	const bearerPrefix = "Bearer "
	authorization := c.GetHeader("Authorization")
	if !strings.HasPrefix(authorization, bearerPrefix) ||
		subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authorization, bearerPrefix)), []byte(token)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	c.Next()
}

func (api *Api) handleAuthentikWebhook(c *gin.Context) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, authentikWebhookMaxBodySize))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"message": "Webhook payload too large"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Could not read webhook payload"})
		return
	}

	payload := bson.M{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid JSON payload"})
		return
	}
	eventUUID, hasEventUUID := payload["event_uuid"].(string)
	action, hasAction := payload["action"].(string)
	created, hasCreated := payload["created"].(string)
	if !hasEventUUID || strings.TrimSpace(eventUUID) == "" || !hasAction || strings.TrimSpace(action) == "" || !hasCreated {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Payload must include event_uuid, action, and created"})
		return
	}
	createdAt, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "created must be an RFC3339 timestamp"})
		return
	}
	payload["created"] = createdAt.UTC()

	inserted, err := api.webhookHandler.MongoDBConnection.StoreAuthentikEvent(c.Request.Context(), payload)
	if err != nil {
		api.log.Error("Could not store Authentik webhook event", zap.String("event_uuid", eventUUID), zap.Error(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Could not store webhook event"})
		return
	}
	if inserted {
		c.Status(http.StatusAccepted)
		return
	}
	c.Status(http.StatusOK)
}
