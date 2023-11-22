package mongomanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

const databaseProduct = "product"
const collectionTrial = "trial"

func (mongoConnection *MongoConnection) CreateTrial(ctx context.Context, trial Trial) error {
	return mongoConnection.createDocument(ctx, databaseProduct, collectionTrial, trial)
}

func (mongoConnection *MongoConnection) GetTrial(ctx context.Context, trialFilter Trial) (Trial, error) {
	if trialFilter.ProductType == "" {
		return Trial{}, errors.New("trial.Product is empty")
	}
	filter := bson.M{
		"productType": trialFilter.ProductType,
		"$or": []bson.M{
			{"customerID": trialFilter.CustomerID},
			{"paymentCardFingerprint": trialFilter.PaymentCardFingerprint},
			{"paymentPayPalEMail": trialFilter.PaymentPayPalEMail},
			{"paymentSEPAFingerprint": trialFilter.PaymentSEPAFingerprint},
		},
	}
	singleResult, err := mongoConnection.findDocument(ctx, databaseProduct, collectionTrial, filter)
	if err != nil {
		return Trial{}, err
	}
	var trial Trial
	decodeErr := singleResult.Decode(&trial)
	if decodeErr != nil {
		return Trial{}, decodeErr
	}
	return trial, nil
}
