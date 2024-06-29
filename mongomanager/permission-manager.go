package mongomanager

import (
	"context"
	"errors"
	"net/http"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
)

func (mongoConnection *MongoConnection) HasPermission(ctx context.Context, tokenDetails firebasemanager.TokenDetails, subscriptionID string, requiredRole Role) error {
	seat, err := mongoConnection.GetSeat(ctx, subscriptionID, tokenDetails.UID)
	if err != nil {
		mongoConnection.Log.Warn("user with UID " + tokenDetails.UID + " tried to access subscriptionID " + subscriptionID + " error: " + err.Error())
		return errors.New(http.StatusText(http.StatusForbidden))
	}
	if !containsRole(seat, requiredRole) {
		mongoConnection.Log.Warn("user with UID " + tokenDetails.UID + " has no permission for role " + string(requiredRole) + " on subscriptionID " + subscriptionID)
		return errors.New(http.StatusText(http.StatusForbidden))
	}

	return nil
}

func containsRole(seat Seat, requiredRole Role) bool {
	for _, role := range seat.Roles {
		if role == requiredRole || role == RoleOwner {
			return true
		}
	}
	return false
}
