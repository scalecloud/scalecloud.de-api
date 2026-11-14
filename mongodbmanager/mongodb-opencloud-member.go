package mongodbmanager

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (mongoDBConnection *MongoDBConnection) ensureOpenCloudMemberIndexes(ctx context.Context) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudMembers)
	if err != nil {
		return err
	}
	_, err = collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "subscriptionID", Value: 1}, {Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("UniqueOpenCloudSubscriptionEmail"),
		},
		{
			Keys:    bson.D{{Key: "invitationTokenHash", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("UniqueOpenCloudInvitationTokenHash"),
		},
		{
			Keys: bson.D{
				{Key: "authentikUserPk", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("OpenCloudMemberAuthentikStatus"),
		},
		{
			Keys: bson.D{
				{Key: "openCloudUserId", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("OpenCloudMemberUserStatus"),
		},
	})
	return err
}

func (mongoDBConnection *MongoDBConnection) CreateOpenCloudMember(ctx context.Context, member OpenCloudMember) error {
	member.EMail = strings.ToLower(strings.TrimSpace(member.EMail))
	if member.Status == "" {
		member.Status = OpenCloudMemberActive
	}
	if err := ValidateStruct(member); err != nil {
		return err
	}
	return mongoDBConnection.createDocument(ctx, databaseOpenCloud, collectionOpenCloudMembers, member)
}

func (mongoDBConnection *MongoDBConnection) CreatePendingOpenCloudMember(ctx context.Context, member OpenCloudMember) error {
	member.EMail = strings.ToLower(strings.TrimSpace(member.EMail))
	if member.SubscriptionID == "" || member.EMail == "" || member.InvitationTokenHash == "" || member.InvitationExpiresAt == nil {
		return errors.New("pending OpenCloud member is incomplete")
	}
	member.Status = OpenCloudMemberPending
	member.UID = ""
	member.EMailVerified = nil
	return mongoDBConnection.createDocument(ctx, databaseOpenCloud, collectionOpenCloudMembers, member)
}

func (mongoDBConnection *MongoDBConnection) CreatePendingOpenCloudMemberWithinCapacity(ctx context.Context, member OpenCloudMember, maxStorageBytes int64) error {
	if member.OpenCloudStorageBytes < 0 || maxStorageBytes < 1 || member.OpenCloudStorageBytes > maxStorageBytes {
		return ErrOpenCloudStorageCapacityReached
	}
	return mongoDBConnection.WithOpenCloudAllocationLock(ctx, member.SubscriptionID, func() error {
		usedStorage, err := mongoDBConnection.GetOpenCloudStorageUsedBytes(ctx, member.SubscriptionID)
		if err != nil {
			return err
		}
		if usedStorage > maxStorageBytes || member.OpenCloudStorageBytes > maxStorageBytes-usedStorage {
			return ErrOpenCloudStorageCapacityReached
		}
		return mongoDBConnection.CreatePendingOpenCloudMember(ctx, member)
	})
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudMembers(ctx context.Context, subscriptionID string) ([]OpenCloudMember, error) {
	if subscriptionID == "" {
		return nil, errors.New("subscription ID is empty")
	}
	var members []OpenCloudMember
	err := mongoDBConnection.findDocuments(ctx, databaseOpenCloud, collectionOpenCloudMembers,
		bson.M{"subscriptionID": subscriptionID},
		&members,
		options.Find().SetSort(bson.D{{Key: "email", Value: 1}}))
	if err != nil {
		return nil, err
	}
	for i := range members {
		if members[i].Status == "" {
			members[i].Status = OpenCloudMemberActive
		}
	}
	return members, nil
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudMember(ctx context.Context, subscriptionID, uid string) (OpenCloudMember, error) {
	if subscriptionID == "" || uid == "" {
		return OpenCloudMember{}, errors.New("subscription ID and UID are required")
	}
	return mongoDBConnection.findOpenCloudMember(ctx, bson.M{"subscriptionID": subscriptionID, "uid": uid})
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudMemberByEmail(ctx context.Context, subscriptionID, email string) (OpenCloudMember, error) {
	if subscriptionID == "" || strings.TrimSpace(email) == "" {
		return OpenCloudMember{}, errors.New("subscription ID and email are required")
	}
	return mongoDBConnection.findOpenCloudMember(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"email":          strings.ToLower(strings.TrimSpace(email)),
	})
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudOwner(ctx context.Context, subscriptionID string) (OpenCloudMember, error) {
	if subscriptionID == "" {
		return OpenCloudMember{}, errors.New("subscription ID is empty")
	}
	return mongoDBConnection.findOpenCloudMember(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"roles":          bson.M{"$in": []string{string(OpenCloudRoleOwner)}},
	})
}

func (mongoDBConnection *MongoDBConnection) findOpenCloudMember(ctx context.Context, filter bson.M) (OpenCloudMember, error) {
	result, err := mongoDBConnection.findOneDocument(ctx, databaseOpenCloud, collectionOpenCloudMembers, filter)
	if err != nil {
		return OpenCloudMember{}, err
	}
	if result == nil {
		return OpenCloudMember{}, nil
	}
	var member OpenCloudMember
	if err := result.Decode(&member); err != nil {
		return OpenCloudMember{}, err
	}
	if member.Status == "" {
		member.Status = OpenCloudMemberActive
	}
	return member, nil
}

func (mongoDBConnection *MongoDBConnection) GetPendingOpenCloudMemberByInvitationTokenHash(ctx context.Context, tokenHash string, now time.Time) (OpenCloudMember, error) {
	return mongoDBConnection.findOpenCloudMember(ctx, bson.M{
		"status":              OpenCloudMemberPending,
		"invitationTokenHash": tokenHash,
		"invitationExpiresAt": bson.M{"$gt": now},
	})
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudMemberByInvitationTokenHash(ctx context.Context, tokenHash string) (OpenCloudMember, error) {
	return mongoDBConnection.findOpenCloudMember(ctx, bson.M{"invitationTokenHash": tokenHash})
}

func (mongoDBConnection *MongoDBConnection) BeginOpenCloudMemberClaim(ctx context.Context, tokenHash, email, uid string, authentikUserPK int64, now time.Time) (OpenCloudMember, error) {
	if uid == "" || authentikUserPK <= 0 {
		return OpenCloudMember{}, errors.New("UID and Authentik user ID are required")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudMembers)
	if err != nil {
		return OpenCloudMember{}, err
	}
	verified := true
	result := collection.FindOneAndUpdate(ctx, bson.M{
		"status":              OpenCloudMemberPending,
		"invitationTokenHash": tokenHash,
		"invitationExpiresAt": bson.M{"$gt": now},
		"email":               email,
	}, bson.M{"$set": bson.M{
		"status":          OpenCloudMemberClaiming,
		"uid":             uid,
		"emailVerified":   verified,
		"authentikUserPk": authentikUserPK,
	}}, options.FindOneAndUpdate().SetReturnDocument(options.After))
	var member OpenCloudMember
	if err := result.Decode(&member); err != nil {
		return OpenCloudMember{}, err
	}
	return member, nil
}

func (mongoDBConnection *MongoDBConnection) CompleteOpenCloudMemberClaim(ctx context.Context, tokenHash, uid string) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudMembers)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(ctx,
		bson.M{"invitationTokenHash": tokenHash, "uid": uid, "status": OpenCloudMemberClaiming},
		bson.M{
			"$set":   bson.M{"status": OpenCloudMemberActive},
			"$unset": bson.M{"invitationTokenHash": "", "invitationExpiresAt": ""},
		})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("OpenCloud member claim no longer belongs to this user")
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) DeletePendingOpenCloudMember(ctx context.Context, subscriptionID, email string) error {
	return mongoDBConnection.deleteDocument(ctx, databaseOpenCloud, collectionOpenCloudMembers, bson.M{
		"subscriptionID": subscriptionID,
		"email":          strings.ToLower(strings.TrimSpace(email)),
		"status":         OpenCloudMemberPending,
	})
}

func (mongoDBConnection *MongoDBConnection) DeletePendingOpenCloudMemberByInvitationTokenHash(ctx context.Context, tokenHash string) error {
	return mongoDBConnection.deleteDocument(ctx, databaseOpenCloud, collectionOpenCloudMembers, bson.M{
		"status":              OpenCloudMemberPending,
		"invitationTokenHash": tokenHash,
	})
}

func (mongoDBConnection *MongoDBConnection) DeleteOpenCloudMember(ctx context.Context, member OpenCloudMember) error {
	filter := bson.M{"subscriptionID": member.SubscriptionID, "email": member.EMail}
	if member.UID != "" {
		filter["uid"] = member.UID
	}
	return mongoDBConnection.deleteDocument(ctx, databaseOpenCloud, collectionOpenCloudMembers, filter)
}

func (mongoDBConnection *MongoDBConnection) DeleteOpenCloudMembersBySubscription(ctx context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return errors.New("subscription ID is empty")
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudMembers)
	if err != nil {
		return err
	}
	_, err = collection.DeleteMany(ctx, bson.M{"subscriptionID": subscriptionID})
	return err
}

func (mongoDBConnection *MongoDBConnection) GetActiveOpenCloudMembersByAuthentikUserPK(ctx context.Context, userPK int64) ([]OpenCloudMember, error) {
	if userPK <= 0 {
		return nil, errors.New("authentik user ID is required")
	}
	var members []OpenCloudMember
	err := mongoDBConnection.findDocuments(ctx, databaseOpenCloud, collectionOpenCloudMembers, bson.M{
		"authentikUserPk": userPK,
		"$or": []bson.M{
			{"status": OpenCloudMemberActive},
			{"status": bson.M{"$exists": false}},
		},
	}, &members, options.Find())
	return members, err
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudMemberUserID(ctx context.Context, subscriptionID string, userPK int64, userID string) error {
	if subscriptionID == "" || userPK <= 0 || userID == "" {
		return errors.New("subscription ID, Authentik user ID, and OpenCloud user ID are required")
	}
	return mongoDBConnection.updateOpenCloudMember(ctx, bson.M{
		"subscriptionID":  subscriptionID,
		"authentikUserPk": userPK,
		"$or": []bson.M{
			{"status": OpenCloudMemberActive},
			{"status": bson.M{"$exists": false}},
		},
	}, bson.M{"$set": bson.M{"openCloudUserId": userID}}, "active OpenCloud member not found")
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudOwnerDriveID(ctx context.Context, subscriptionID, driveID string) error {
	if subscriptionID == "" || driveID == "" {
		return errors.New("subscription ID and OpenCloud drive ID are required")
	}
	return mongoDBConnection.updateOpenCloudMember(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"roles":          bson.M{"$in": []string{string(OpenCloudRoleOwner)}},
	}, bson.M{"$set": bson.M{"openCloudDriveId": driveID}}, "OpenCloud subscription owner not found")
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudMemberStorageBytes(ctx context.Context, subscriptionID, uidOrEmail string, storageBytes int64) error {
	if subscriptionID == "" || uidOrEmail == "" || storageBytes < 0 {
		return errors.New("subscription ID, member identity, and valid storage allocation are required")
	}
	identity := bson.M{"uid": uidOrEmail}
	if strings.Contains(uidOrEmail, "@") {
		identity = bson.M{"email": strings.ToLower(strings.TrimSpace(uidOrEmail))}
	}
	filter := bson.M{"subscriptionID": subscriptionID}
	for key, value := range identity {
		filter[key] = value
	}
	return mongoDBConnection.updateOpenCloudMember(ctx, filter,
		bson.M{"$set": bson.M{"openCloudStorageBytes": storageBytes}}, "OpenCloud subscription member not found")
}

func (mongoDBConnection *MongoDBConnection) SetOpenCloudOwnerStorageBytes(ctx context.Context, subscriptionID string, storageBytes int64) error {
	if subscriptionID == "" || storageBytes < 0 {
		return errors.New("subscription ID and valid storage allocation are required")
	}
	return mongoDBConnection.updateOpenCloudMember(ctx, bson.M{
		"subscriptionID": subscriptionID,
		"roles":          bson.M{"$in": []string{string(OpenCloudRoleOwner)}},
	}, bson.M{"$set": bson.M{"openCloudStorageBytes": storageBytes}}, "OpenCloud subscription owner not found")
}

func (mongoDBConnection *MongoDBConnection) UpdateOpenCloudMemberRoles(ctx context.Context, subscriptionID, uid string, roles []OpenCloudRole) error {
	if subscriptionID == "" || uid == "" {
		return errors.New("subscription ID and member UID are required")
	}
	identity := bson.M{"uid": uid}
	if strings.Contains(uid, "@") {
		identity = bson.M{"email": strings.ToLower(strings.TrimSpace(uid))}
	}
	filter := bson.M{"subscriptionID": subscriptionID}
	for key, value := range identity {
		filter[key] = value
	}
	return mongoDBConnection.updateOpenCloudMember(ctx, filter, bson.M{"$set": bson.M{"roles": roles}}, "OpenCloud subscription member not found")
}

func (mongoDBConnection *MongoDBConnection) updateOpenCloudMember(ctx context.Context, filter, update bson.M, notFoundMessage string) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseOpenCloud, collectionOpenCloudMembers)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New(notFoundMessage)
	}
	return nil
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudUserStorageBytes(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, errors.New("OpenCloud user ID is required")
	}
	var members []OpenCloudMember
	if err := mongoDBConnection.findDocuments(ctx, databaseOpenCloud, collectionOpenCloudMembers, bson.M{
		"openCloudUserId": userID,
		"$or": []bson.M{
			{"status": OpenCloudMemberActive},
			{"status": bson.M{"$exists": false}},
		},
	}, &members, options.Find()); err != nil {
		return 0, err
	}
	var total int64
	for _, member := range members {
		if member.OpenCloudStorageBytes < 0 || member.OpenCloudStorageBytes > math.MaxInt64-total {
			return 0, errors.New("invalid OpenCloud user allocation total")
		}
		total += member.OpenCloudStorageBytes
	}
	return total, nil
}

func (mongoDBConnection *MongoDBConnection) HasOpenCloudPermission(ctx context.Context, tokenDetails authmanager.TokenDetails, subscriptionID string, requiredRoles []OpenCloudRole) error {
	member, err := mongoDBConnection.GetOpenCloudMember(ctx, subscriptionID, tokenDetails.UID)
	if err != nil || !ContainsOpenCloudRole(member, requiredRoles) {
		mongoDBConnection.Log.Warn("user with UID " + tokenDetails.UID + " tried to access OpenCloud subscription " + subscriptionID)
		return errors.New("Forbidden")
	}
	return nil
}

func ContainsOpenCloudRole(member OpenCloudMember, requiredRoles []OpenCloudRole) bool {
	roleMap := make(map[OpenCloudRole]bool, len(requiredRoles))
	for _, role := range requiredRoles {
		roleMap[role] = true
	}
	for _, role := range member.Roles {
		if role == OpenCloudRoleOwner || roleMap[role] {
			return true
		}
	}
	return false
}

func (mongoDBConnection *MongoDBConnection) GetOpenCloudStorageUsedBytes(ctx context.Context, subscriptionID string) (int64, error) {
	if subscriptionID == "" {
		return 0, errors.New("subscription ID is required")
	}
	members, err := mongoDBConnection.GetOpenCloudMembers(ctx, subscriptionID)
	if err != nil {
		return 0, err
	}
	var usedBytes int64
	for _, member := range members {
		if member.OpenCloudStorageBytes < 0 || member.OpenCloudStorageBytes > math.MaxInt64-usedBytes {
			return 0, errors.New("invalid OpenCloud member allocation total")
		}
		usedBytes += member.OpenCloudStorageBytes
	}
	resources, err := mongoDBConnection.GetOpenCloudResourcesBySubscription(ctx, subscriptionID)
	if err != nil {
		return 0, err
	}
	for _, resource := range resources {
		if resource.ResourceType != "space" || resource.Scope != "allocation" {
			continue
		}
		if resource.StorageQuotaBytes < 0 || resource.StorageQuotaBytes > math.MaxInt64-usedBytes {
			return 0, errors.New("invalid OpenCloud space allocation total")
		}
		usedBytes += resource.StorageQuotaBytes
	}
	return usedBytes, nil
}
