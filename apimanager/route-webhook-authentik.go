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
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

const authentikWebhookMaxBodySize = 1 << 20
const authentikOpenCloudApplicationName = "OpenCloud"

type authentikWebhookEvent struct {
	EventUUID string `json:"event_uuid"`
	UserPK    int64  `json:"user_pk"`
	Username  string `json:"username"`
	Created   string `json:"created"`
}

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

	var event authentikWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		api.log.Info("Authentik webhook payload rejected as invalid JSON", zap.String("body", string(body)))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid JSON payload"})
		return
	}
	if strings.TrimSpace(event.EventUUID) == "" || event.UserPK == 0 || strings.TrimSpace(event.Created) == "" {
		api.log.Info("Authentik webhook payload missing required fields", zap.String("body", string(body)))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Payload must include event_uuid, user_pk, and created"})
		return
	}
	createdAt, err := time.Parse(time.RFC3339Nano, event.Created)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "created must be an RFC3339 timestamp"})
		return
	}
	payload := bson.M{
		"event_uuid":             event.EventUUID,
		"user_pk":                event.UserPK,
		"authorized_application": authentikOpenCloudApplicationName,
		"created":                createdAt.UTC(),
	}

	inserted, err := api.webhookHandler.MongoDBConnection.StoreAuthentikEvent(c.Request.Context(), payload)
	if err != nil {
		api.log.Error("Could not store Authentik webhook event", zap.String("event_uuid", event.EventUUID), zap.Error(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Could not store webhook event"})
		return
	}
	if api.webhookHandler.OpenCloudConnection != nil {
		if strings.TrimSpace(event.Username) == "" {
			api.log.Warn("Authentik OpenCloud event has no username; skipping seat activation", zap.Int64("authentik_user_pk", event.UserPK))
		} else {
			activated, activationErr := api.paymentHandler.ActivateOpenCloudSeats(c.Request.Context(), event.UserPK, event.Username)
			switch {
			case errors.Is(activationErr, opencloudmanager.ErrNotFound):
				api.log.Info("OpenCloud user not provisioned yet; a later login event can activate seats", zap.String("username", event.Username))
			case activationErr != nil:
				api.log.Error("Could not activate OpenCloud seats", zap.String("username", event.Username), zap.Error(activationErr))
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Could not activate OpenCloud seats"})
				return
			default:
				api.log.Info("Processed OpenCloud login event", zap.String("username", event.Username), zap.Int("activated_seats", activated))
			}
		}
	}
	if inserted {
		api.log.Info("Stored OpenCloud authorization event",
			zap.String("event_uuid", event.EventUUID),
			zap.Int64("authentik_user_pk", event.UserPK),
		)
		c.Status(http.StatusAccepted)
		return
	}
	c.Status(http.StatusOK)
}
