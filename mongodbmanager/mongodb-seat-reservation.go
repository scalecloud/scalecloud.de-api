package mongodbmanager

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrSeatCapacityReached = errors.New("all purchased seats are already assigned or invited")

const (
	seatReservationLockLease = 30 * time.Second
	seatReservationLockWait  = 5 * time.Second
)

func (mongoDBConnection *MongoDBConnection) CreatePendingSeatWithinCapacity(ctx context.Context, seat Seat, maxSeats int64) error {
	if maxSeats < 1 {
		return ErrSeatCapacityReached
	}
	lockContext, cancel := context.WithTimeout(ctx, seatReservationLockWait)
	defer cancel()
	collection, err := mongoDBConnection.getCollection(lockContext, databaseSubscription, collectionSeatLocks)
	if err != nil {
		return err
	}
	_, err = collection.UpdateOne(
		lockContext,
		bson.M{"_id": seat.SubscriptionID},
		bson.M{"$setOnInsert": bson.M{"lockUntil": time.Unix(0, 0).UTC()}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	lockBytes := make([]byte, 16)
	if _, err := rand.Read(lockBytes); err != nil {
		return err
	}
	lockID := base64.RawURLEncoding.EncodeToString(lockBytes)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		now := time.Now().UTC()
		result, err := collection.UpdateOne(
			lockContext,
			bson.M{"_id": seat.SubscriptionID, "lockUntil": bson.M{"$lte": now}},
			bson.M{"$set": bson.M{"lockUntil": now.Add(seatReservationLockLease), "lockID": lockID}},
		)
		if err != nil {
			return err
		}
		if result.MatchedCount == 1 {
			break
		}
		select {
		case <-lockContext.Done():
			return errors.New("timed out waiting to reserve subscription seat")
		case <-ticker.C:
		}
	}
	defer func() {
		releaseContext, releaseCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer releaseCancel()
		_, _ = collection.UpdateOne(
			releaseContext,
			bson.M{"_id": seat.SubscriptionID, "lockID": lockID},
			bson.M{"$set": bson.M{"lockUntil": time.Unix(0, 0).UTC()}, "$unset": bson.M{"lockID": ""}},
		)
	}()

	usedSeats, err := mongoDBConnection.CountSeats(ctx, seat.SubscriptionID)
	if err != nil {
		return err
	}
	if usedSeats >= maxSeats {
		return ErrSeatCapacityReached
	}
	return mongoDBConnection.CreatePendingSeat(ctx, seat)
}
