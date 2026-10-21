package mongodbmanager

import (
	"context"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (mongoDBConnection *MongoDBConnection) CreateTrial(ctx context.Context, trial Trial) error {
	err := ValidateStruct(trial)
	if err != nil {
		return err
	}
	return mongoDBConnection.createDocument(ctx, databaseProduct, collectionTrial, trial)
}

func (mongoDBConnection *MongoDBConnection) GetTrial(ctx context.Context, trialFilter TrialFilter) (Trial, error) {
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
	singleResult, err := mongoDBConnection.findOneDocument(ctx, databaseProduct, collectionTrial, filter)
	if err != nil {
		return Trial{}, err
	}
	if singleResult == nil {
		return Trial{}, nil
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
