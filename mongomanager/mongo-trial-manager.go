package mongomanager

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const databaseProduct = "product"
const collectionTrial = "trial"

func (mongoConnection *MongoConnection) CreateTrial(ctx context.Context, trial Trial) error {
	err := ValidateStruct(trial)
	if err != nil {
		return err
	}
	return mongoConnection.createDocument(ctx, databaseProduct, collectionTrial, trial)
}

func (mongoConnection *MongoConnection) GetTrial(ctx context.Context, trialFilter Trial) (Trial, error) {
	err := ValidateStruct(trialFilter)
	if err != nil {
		return Trial{}, err
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
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Trial{}, nil
		}
	}
	var trial Trial
	decodeErr := singleResult.Decode(&trial)
	if decodeErr != nil {
		return Trial{}, decodeErr
	}
	return trial, nil
}

func ValidateStruct(s interface{}) error {
	val := validator.New(validator.WithRequiredStructEnabled())
	return val.Struct(s)
}
