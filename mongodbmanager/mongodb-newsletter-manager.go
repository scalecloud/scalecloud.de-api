package mongodbmanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func (mongoDBConnection *MongoDBConnection) ensureNewsletterIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("UniqueNewsletterEmail"),
	}
	collection, err := mongoDBConnection.getCollection(ctx, databaseNewsletters, collectionSubscribers)
	if err != nil {
		return err
	}
	name, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		mongoDBConnection.Log.Error("Error creating index for newsletter", zap.String("error", err.Error()))
		return err
	}

	mongoDBConnection.Log.Info("Required index for collection " + collection.Name() + " is present. Index: " + name)
	return nil
}

func (mongoDBConnection *MongoDBConnection) CreateNewsletterSubscriber(ctx context.Context, newsletterSubscriber NewsletterSubscriber) error {
	err := ValidateStruct(newsletterSubscriber)
	if err != nil {
		return err
	}
	return mongoDBConnection.createDocument(ctx, databaseNewsletters, collectionSubscribers, newsletterSubscriber)
}

func (mongoDBConnection *MongoDBConnection) CountNewsletterSubscriber(ctx context.Context, email string) (int64, error) {
	return mongoDBConnection.countDocuments(ctx, databaseNewsletters, collectionSubscribers, bson.M{"email": email})
}

func (mongoDBConnection *MongoDBConnection) GetAllNewsletterSubscribers(ctx context.Context) ([]NewsletterSubscriber, error) {
	return mongoDBConnection.GetNewsletterSubscribers(ctx, 0, 0)
}

func (mongoDBConnection *MongoDBConnection) GetNewsletterSubscribers(ctx context.Context, pageIndex int, pageSize int) ([]NewsletterSubscriber, error) {
	filter := bson.M{
		"status": NewsletterStatusActive,
	}
	var newsletterSubscribers []NewsletterSubscriber
	opts := options.Find()
	opts.SetLimit(int64(pageSize))
	opts.SetSkip(int64(pageIndex * pageSize))
	opts.SetSort(bson.D{{Key: "email", Value: 1}})
	err := mongoDBConnection.findDocuments(ctx, databaseNewsletters, collectionSubscribers, filter, &newsletterSubscribers, opts)
	if err != nil {
		return []NewsletterSubscriber{}, err
	}
	return newsletterSubscribers, nil
}

func (mongoDBConnection *MongoDBConnection) GetNewsletterSubscriber(ctx context.Context, email string) (NewsletterSubscriber, error) {
	if email == "" {
		return NewsletterSubscriber{}, errors.New("email is empty")
	}

	filter := bson.M{
		"email": email,
	}
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
	if err != nil {
		mongoDBConnection.Log.Error("Error finding document", zap.Error(err))
		return NewsletterSubscriber{}, errors.New("error finding newsletter subscriber")
	}
	if singleResult == nil {
		return NewsletterSubscriber{}, nil
	}
	var newsletterSubscriber NewsletterSubscriber
	decodeErr := singleResult.Decode(&newsletterSubscriber)
	if decodeErr != nil {
		return NewsletterSubscriber{}, decodeErr
	}
	return newsletterSubscriber, nil
}

func (mongoDBConnection *MongoDBConnection) GetNewsletterSubscriberByVerificationToken(ctx context.Context, verificationToken string) (NewsletterSubscriber, error) {
	if verificationToken == "" {
		return NewsletterSubscriber{}, errors.New("verificationToken is empty")
	}

	filter := bson.M{
		"verificationToken": verificationToken,
	}
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
	if err != nil {
		mongoDBConnection.Log.Error("Error finding document", zap.Error(err))
		return NewsletterSubscriber{}, errors.New("error finding newsletter subscriber")
	}
	if singleResult == nil {
		return NewsletterSubscriber{}, nil
	}
	var newsletterSubscriber NewsletterSubscriber
	decodeErr := singleResult.Decode(&newsletterSubscriber)
	if decodeErr != nil {
		return NewsletterSubscriber{}, decodeErr
	}
	return newsletterSubscriber, nil
}

func (mongoDBConnection *MongoDBConnection) GetNewsletterSubscriberByUnsubscribeToken(ctx context.Context, unsubscribeToken string) (NewsletterSubscriber, error) {
	if unsubscribeToken == "" {
		return NewsletterSubscriber{}, errors.New("unsubscribeToken is empty")
	}
	filter := bson.M{
		"unsubscribeToken": unsubscribeToken,
	}
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
	if err != nil {
		mongoDBConnection.Log.Error("Error finding document", zap.Error(err))
		return NewsletterSubscriber{}, errors.New("error finding newsletter subscriber")
	}
	if singleResult == nil {
		return NewsletterSubscriber{}, nil
	}
	var newsletterSubscriber NewsletterSubscriber
	decodeErr := singleResult.Decode(&newsletterSubscriber)
	if decodeErr != nil {
		return NewsletterSubscriber{}, decodeErr
	}
	return newsletterSubscriber, nil
}

func (mongoDBConnection *MongoDBConnection) UpdateNewsletterSubscriber(ctx context.Context, newsletterSubscriber NewsletterSubscriber) error {
	filter := bson.M{
		"email": newsletterSubscriber.EMail,
	}
	update := bson.M{
		"$set": newsletterSubscriber,
	}
	return mongoDBConnection.updateDocument(ctx, databaseNewsletters, collectionSubscribers, filter, update)
}

func (mongoDBConnection *MongoDBConnection) DeleteNewsletterSubscriber(ctx context.Context, newsletterSubscriber NewsletterSubscriber) error {
	filter := bson.M{
		"email": newsletterSubscriber.EMail,
	}
	return mongoDBConnection.deleteDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
}
