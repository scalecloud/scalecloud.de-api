package mongodbmanager

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func (mongoDBConnection *MongoDBConnection) ensureSeatIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "subscriptionID", Value: 1},
			{Key: "email", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("UniqueSubscriptionEmail"),
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	name, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		mongoDBConnection.Log.Error("Error creating index for seats", zap.String("error", err.Error()))
		return err
	}

	mongoDBConnection.Log.Info("Required index for collection " + collection.Name() + " is present. Index: " + name)
	return nil
}

func (mongoDBConnection *MongoDBConnection) ensureSeatInvitationTokenIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "invitationTokenHash", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true).SetName("UniqueSeatInvitationTokenHash"),
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		mongoDBConnection.Log.Error("Error creating index for seat invitation tokens", zap.Error(err))
		return err
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) CreateSeat(ctx context.Context, seat Seat) error {
	if seat.Status == "" {
		seat.Status = SeatStatusActive
	}
	err := ValidateStruct(seat)
	if err != nil {
		return err
	}
	return mongoDBConnection.createDocument(ctx, databaseSubscription, collectionSeats, seat)
}

func (mongoDBConnection *MongoDBConnection) CreatePendingSeat(ctx context.Context, seat Seat) error {
	if seat.SubscriptionID == "" || seat.EMail == "" || len(seat.Roles) == 0 || seat.InvitationTokenHash == "" || seat.InvitationExpiresAt == nil {
		return errors.New("pending seat is incomplete")
	}
	seat.Status = SeatStatusPending
	seat.UID = ""
	seat.EMailVerified = nil
	return mongoDBConnection.createDocument(ctx, databaseSubscription, collectionSeats, seat)
}

func (mongoDBConnection *MongoDBConnection) CountSeats(ctx context.Context, subscriptionID string) (int64, error) {
	return mongoDBConnection.countDocuments(ctx, databaseSubscription, collectionSeats, bson.M{"subscriptionID": subscriptionID})
}

func (mongoDBConnection *MongoDBConnection) GetAllSeats(ctx context.Context, subscriptionID string) ([]Seat, error) {
	return mongoDBConnection.GetSeats(ctx, subscriptionID, 0, 0)
}

func (mongoDBConnection *MongoDBConnection) GetSeats(ctx context.Context, subscriptionID string, pageIndex int, pageSize int) ([]Seat, error) {
	if subscriptionID == "" {
		return []Seat{}, errors.New("subscription ID is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
	}
	var seats []Seat
	opts := options.Find()
	opts.SetLimit(int64(pageSize))
	opts.SetSkip(int64(pageIndex * pageSize))
	opts.SetSort(bson.D{{Key: "email", Value: 1}})
	err := mongoDBConnection.findDocuments(ctx, databaseSubscription, collectionSeats, filter, &seats, opts)
	if err != nil {
		return []Seat{}, err
	}
	for index := range seats {
		if seats[index].Status == "" {
			seats[index].Status = SeatStatusActive
		}
	}
	return seats, nil
}

func (mongoDBConnection *MongoDBConnection) GetSeat(ctx context.Context, subscriptionID, uid string) (Seat, error) {
	if subscriptionID == "" {
		return Seat{}, errors.New("subscription ID is empty")
	}
	if uid == "" {
		return Seat{}, errors.New("uid is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
		"uid":            uid,
	}
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, filter)
	if err != nil {
		mongoDBConnection.Log.Error("Error finding seat", zap.Error(err))
		return Seat{}, errors.New("error finding seat")
	}
	if singleResult == nil {
		return Seat{}, nil
	}
	var seat Seat
	decodeErr := singleResult.Decode(&seat)
	if decodeErr != nil {
		return Seat{}, decodeErr
	}
	if seat.Status == "" {
		seat.Status = SeatStatusActive
	}
	return seat, nil
}

func (mongoDBConnection *MongoDBConnection) GetSeatByEmail(ctx context.Context, subscriptionID, email string) (Seat, error) {
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, bson.M{
		"subscriptionID": subscriptionID,
		"email":          email,
	})
	if err != nil {
		return Seat{}, err
	}
	if singleResult == nil {
		return Seat{}, nil
	}
	var seat Seat
	if err := singleResult.Decode(&seat); err != nil {
		return Seat{}, err
	}
	if seat.Status == "" {
		seat.Status = SeatStatusActive
	}
	return seat, nil
}

func (mongoDBConnection *MongoDBConnection) GetSeatByInvitationTokenHash(ctx context.Context, tokenHash string) (Seat, error) {
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, bson.M{
		"invitationTokenHash": tokenHash,
	})
	if err != nil {
		return Seat{}, err
	}
	if singleResult == nil {
		return Seat{}, nil
	}
	var seat Seat
	if err := singleResult.Decode(&seat); err != nil {
		return Seat{}, err
	}
	if seat.Status == "" {
		seat.Status = SeatStatusActive
	}
	return seat, nil
}

func (mongoDBConnection *MongoDBConnection) GetPendingSeatByInvitationTokenHash(ctx context.Context, tokenHash string, now time.Time) (Seat, error) {
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, bson.M{
		"status":              SeatStatusPending,
		"invitationTokenHash": tokenHash,
		"invitationExpiresAt": bson.M{"$gt": now},
	})
	if err != nil {
		return Seat{}, err
	}
	if singleResult == nil {
		return Seat{}, nil
	}
	var seat Seat
	if err := singleResult.Decode(&seat); err != nil {
		return Seat{}, err
	}
	return seat, nil
}

func (mongoDBConnection *MongoDBConnection) BeginSeatClaim(ctx context.Context, tokenHash, email, uid string, authentikUserPK int64, now time.Time) (Seat, error) {
	if uid == "" {
		return Seat{}, errors.New("uid is empty")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return Seat{}, err
	}
	verified := true
	result := collection.FindOneAndUpdate(
		ctx,
		bson.M{
			"status":              SeatStatusPending,
			"invitationTokenHash": tokenHash,
			"invitationExpiresAt": bson.M{"$gt": now},
			"email":               email,
		},
		bson.M{"$set": bson.M{
			"status":          SeatStatusClaiming,
			"uid":             uid,
			"emailVerified":   verified,
			"authentikUserPk": authentikUserPK,
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var seat Seat
	if err := result.Decode(&seat); err != nil {
		return Seat{}, err
	}
	return seat, nil
}

func (mongoDBConnection *MongoDBConnection) CompleteSeatClaim(ctx context.Context, tokenHash, uid string) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"invitationTokenHash": tokenHash, "uid": uid, "status": SeatStatusClaiming},
		bson.M{
			"$set":   bson.M{"status": SeatStatusActive},
			"$unset": bson.M{"invitationTokenHash": "", "invitationExpiresAt": ""},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("seat claim no longer belongs to this user")
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) DeletePendingSeat(ctx context.Context, subscriptionID, email string) error {
	return mongoDBConnection.deleteDocument(ctx, databaseSubscription, collectionSeats, bson.M{
		"subscriptionID": subscriptionID,
		"email":          email,
		"status":         SeatStatusPending,
	})
}

func (mongoDBConnection *MongoDBConnection) DeletePendingSeatByInvitationTokenHash(ctx context.Context, tokenHash string) error {
	return mongoDBConnection.deleteDocument(ctx, databaseSubscription, collectionSeats, bson.M{
		"status":              SeatStatusPending,
		"invitationTokenHash": tokenHash,
	})
}

func (mongoDBConnection *MongoDBConnection) GetOwnerSeat(ctx context.Context, subscriptionID string) (Seat, error) {
	if subscriptionID == "" {
		return Seat{}, errors.New("subscription ID is empty")
	}
	filter := bson.M{
		"subscriptionID": subscriptionID,
		"roles": bson.M{
			"$in": []string{string(RoleOwner)},
		},
	}
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseSubscription, collectionSeats, filter)
	if err != nil {
		mongoDBConnection.Log.Error("Error finding seat", zap.Error(err))
		return Seat{}, errors.New("error finding seat")
	}
	if singleResult == nil {
		return Seat{}, nil
	}
	var seat Seat
	decodeErr := singleResult.Decode(&seat)
	if decodeErr != nil {
		return Seat{}, decodeErr
	}
	return seat, nil
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudDriveID(ctx context.Context, subscriptionID, driveID string) error {
	if subscriptionID == "" || driveID == "" {
		return errors.New("subscription ID and OpenCloud drive ID are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"roles":          bson.M{"$in": []string{string(RoleOwner)}},
	}, bson.M{"$set": bson.M{"openCloudDriveId": driveID}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("subscription owner seat not found")
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) GetActiveSeatsByAuthentikUserPK(ctx context.Context, userPK int64) ([]Seat, error) {
	if userPK <= 0 {
		return nil, errors.New("Authentik user ID is required")
	}
	filter := bson.M{
		"authentikUserPk": userPK,
		"$or": []bson.M{
			{"status": SeatStatusActive},
			{"status": bson.M{"$exists": false}},
		},
	}
	var seats []Seat
	if err := mongoDBConnection.findDocuments(ctx, databaseSubscription, collectionSeats, filter, &seats, options.Find()); err != nil {
		return nil, err
	}
	return seats, nil
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudUserID(ctx context.Context, subscriptionID string, userPK int64, openCloudUserID string) error {
	if subscriptionID == "" || userPK <= 0 || openCloudUserID == "" {
		return errors.New("subscription ID, Authentik user ID, and OpenCloud user ID are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseSubscription, collectionSeats)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(ctx, bson.M{
		"subscriptionID":  subscriptionID,
		"authentikUserPk": userPK,
		"$or": []bson.M{
			{"status": SeatStatusActive},
			{"status": bson.M{"$exists": false}},
		},
	}, bson.M{"$set": bson.M{"openCloudUserId": openCloudUserID}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("active subscription seat not found")
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) UpdateSeat(ctx context.Context, seat Seat) error {
	filter := bson.M{
		"subscriptionID": seat.SubscriptionID,
		"uid":            seat.UID,
		"$or": []bson.M{
			{"status": SeatStatusActive},
			{"status": bson.M{"$exists": false}},
		},
	}
	update := bson.M{"$set": bson.M{
		"email":         seat.EMail,
		"emailVerified": seat.EMailVerified,
		"roles":         seat.Roles,
	}}
	if seat.Status == SeatStatusActive {
		update["$set"].(bson.M)["status"] = SeatStatusActive
	}
	return mongoDBConnection.updateDocument(ctx, databaseSubscription, collectionSeats, filter, update)
}

func (mongoDBConnection *MongoDBConnection) DeleteSeat(ctx context.Context, seat Seat) error {
	filter := bson.M{
		"subscriptionID": seat.SubscriptionID,
		"uid":            seat.UID,
	}
	return mongoDBConnection.deleteDocument(ctx, databaseSubscription, collectionSeats, filter)
}
