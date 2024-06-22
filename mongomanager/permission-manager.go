package mongomanager

import (
	"context"
	"errors"
	"net/http"
)

func (mongoConnection *MongoConnection) HasPermission(ctx context.Context, uid, subscriptionID string, requiredRole Role) error {
	seat, err := mongoConnection.GetSeat(ctx, subscriptionID, uid)
	if err != nil {
		return err
	}
	if !containsRole(seat, requiredRole) {
		mongoConnection.Log.Warn("user with UID " + uid + " tried checking permission for role " + string(requiredRole) + " on subscriptionID " + subscriptionID)
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
