package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func (mongoConnection *MongoConnection) ensureNewsletterIndex() error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("UniqueNewsletterEmail"),
	}
	collection, err := mongoConnection.getCollection(context.Background(), databaseNewsletters, collectionSubscribers)
	if err != nil {
		return err
	}
	name, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		mongoConnection.Log.Error("Error creating index for newsletter", zap.String("error", err.Error()))
		return err
	}

	mongoConnection.Log.Info("Required index for collection " + collection.Name() + " is present. Index: " + name)
	return nil
}

func (mongoConnection *MongoConnection) CreateNewsletterSubscriber(ctx context.Context, newsletterSubscriber NewsletterSubscriber) error {
	err := ValidateStruct(newsletterSubscriber)
	if err != nil {
		return err
	}
	return mongoConnection.createDocument(ctx, databaseNewsletters, collectionSubscribers, newsletterSubscriber)
}

func (mongoConnection *MongoConnection) CountNewsletterSubscriber(ctx context.Context, email string) (int64, error) {
	return mongoConnection.countDocuments(ctx, databaseNewsletters, collectionSubscribers, bson.M{"email": email})
}

func (mongoConnection *MongoConnection) GetAllNewsletterSubscribers(ctx context.Context) ([]NewsletterSubscriber, error) {
	return mongoConnection.GetNewsletterSubscribers(ctx, 0, 0)
}

func (mongoConnection *MongoConnection) GetNewsletterSubscribers(ctx context.Context, pageIndex int, pageSize int) ([]NewsletterSubscriber, error) {
	filter := bson.M{
		"status": NewsletterStatusActive,
	}
	var newsletterSubscribers []NewsletterSubscriber
	opts := options.Find()
	opts.SetLimit(int64(pageSize))
	opts.SetSkip(int64(pageIndex * pageSize))
	opts.SetSort(bson.D{{Key: "email", Value: 1}})
	err := mongoConnection.findDocuments(ctx, databaseNewsletters, collectionSubscribers, filter, &newsletterSubscribers, opts)
	if err != nil {
		return []NewsletterSubscriber{}, err
	}
	return newsletterSubscribers, nil
}

func (mongoConnection *MongoConnection) GetNewsletterSubscriber(ctx context.Context, email string) (NewsletterSubscriber, error) {
	if email == "" {
		return NewsletterSubscriber{}, errors.New("email is empty")
	}

	filter := bson.M{
		"email": email,
	}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return NewsletterSubscriber{}, nil
		} else {
			mongoConnection.Log.Error("Error finding document", zap.Error(err))
			return NewsletterSubscriber{}, errors.New("error finding newsletter subscriber")
		}
	}
	var newsletterSubscriber NewsletterSubscriber
	decodeErr := singleResult.Decode(&newsletterSubscriber)
	if decodeErr != nil {
		return NewsletterSubscriber{}, decodeErr
	}
	return newsletterSubscriber, nil
}

func (mongoConnection *MongoConnection) GetNewsletterSubscriberByVerificationToken(ctx context.Context, verificationToken string) (NewsletterSubscriber, error) {
	if verificationToken == "" {
		return NewsletterSubscriber{}, errors.New("verificationToken is empty")
	}

	filter := bson.M{
		"verificationToken": verificationToken,
	}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
	if err != nil {
		mongoConnection.Log.Error("Error finding document", zap.Error(err))
		return NewsletterSubscriber{}, errors.New("error finding newsletter subscriber")
	}
	var newsletterSubscriber NewsletterSubscriber
	decodeErr := singleResult.Decode(&newsletterSubscriber)
	if decodeErr != nil {
		return NewsletterSubscriber{}, decodeErr
	}
	return newsletterSubscriber, nil
}

func (mongoConnection *MongoConnection) GetNewsletterSubscriberByUnsubscribeToken(ctx context.Context, unsubscribeToken string) (NewsletterSubscriber, error) {
	if unsubscribeToken == "" {
		return NewsletterSubscriber{}, errors.New("unsubscribeToken is empty")
	}
	filter := bson.M{
		"unsubscribeToken": unsubscribeToken,
	}
	singleResult, err := mongoConnection.findOneDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
	if err != nil {
		mongoConnection.Log.Error("Error finding document", zap.Error(err))
		return NewsletterSubscriber{}, nil
	}
	var newsletterSubscriber NewsletterSubscriber
	decodeErr := singleResult.Decode(&newsletterSubscriber)
	if decodeErr != nil {
		return NewsletterSubscriber{}, decodeErr
	}
	return newsletterSubscriber, nil
}

func (mongoConnection *MongoConnection) UpdateNewsletterSubscriber(ctx context.Context, newsletterSubscriber NewsletterSubscriber) error {
	filter := bson.M{
		"email": newsletterSubscriber.EMail,
	}
	update := bson.M{
		"$set": newsletterSubscriber,
	}
	return mongoConnection.updateDocument(ctx, databaseNewsletters, collectionSubscribers, filter, update)
}

func (mongoConnection *MongoConnection) DeleteNewsletterSubscriber(ctx context.Context, newsletterSubscriber NewsletterSubscriber) error {
	filter := bson.M{
		"email": newsletterSubscriber.EMail,
	}
	return mongoConnection.deleteDocument(ctx, databaseNewsletters, collectionSubscribers, filter)
}
