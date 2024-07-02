package mongomanager

import (
	"context"
	"errors"
	"net/http"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
)

func (mongoConnection *MongoConnection) HasPermission(ctx context.Context, tokenDetails firebasemanager.TokenDetails, subscriptionID string, requiredRoles []Role) error {
	seat, err := mongoConnection.GetSeat(ctx, subscriptionID, tokenDetails.UID)
	if err != nil {
		mongoConnection.Log.Warn("user with UID " + tokenDetails.UID + " tried to access subscriptionID " + subscriptionID + " error: " + err.Error())
		return errors.New(http.StatusText(http.StatusForbidden))
	}
	if !containsRole(seat, requiredRoles) {
		mongoConnection.Log.Warn("user with UID " + tokenDetails.UID + " is missing role " + " on subscriptionID " + subscriptionID)
		return errors.New(http.StatusText(http.StatusForbidden))
	}

	return nil
}

func containsRole(seat Seat, requiredRoles []Role) bool {
	// Convert requiredRoles slice to a map for constant-time lookups
	roleMap := make(map[Role]bool)
	for _, role := range requiredRoles {
		roleMap[role] = true
	}

	// Check if seat has any of the required roles or is a RoleOwner
	for _, seatRole := range seat.Roles {
		if roleMap[seatRole] || seatRole == RoleOwner {
			return true
		}
	}

	return false
}
