package apimanager

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

type seatInvitationTokenRequest struct {
	Token string `json:"token" binding:"required,min=40,max=128"`
}

func (api *Api) previewSeatInvitation(c *gin.Context) {
	var request seatInvitationTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation token"})
		return
	}
	reply, err := api.paymentHandler.PreviewSeatInvitation(c, request.Token)
	if errors.Is(err, stripemanager.ErrInvitationUnavailable) {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation is not available"})
		return
	}
	if err != nil {
		api.log.Error("Could not load invitation preview", zap.String("route", c.FullPath()), zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "invitation preview is temporarily unavailable"})
		return
	}
	c.JSON(http.StatusOK, reply)
}

func (api *Api) claimSeatInvitation(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil {
		return
	}
	var request seatInvitationTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation token"})
		return
	}
	reply, err := api.paymentHandler.ClaimSeatInvitation(c, request.Token, tokenDetails)
	switch {
	case errors.Is(err, stripemanager.ErrInvitationUnavailable):
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation is not available"})
	case errors.Is(err, stripemanager.ErrInvitationEmailMismatch), errors.Is(err, stripemanager.ErrInvitationEmailUnverified):
		c.JSON(http.StatusForbidden, gin.H{"error": "signed-in account cannot claim this invitation"})
	case err != nil:
		api.log.Error("Could not claim seat invitation", zap.String("route", c.FullPath()), zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "invitation claim is temporarily unavailable; retry"})
	default:
		c.JSON(http.StatusOK, reply)
	}
}
