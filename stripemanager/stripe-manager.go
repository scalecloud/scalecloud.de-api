package stripemanager

import (
	"context"

	"github.com/scalecloud/scalecloud.de-api/emailmanager"
	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager/secret"
	"go.uber.org/zap"
)

type StripeConnection struct {
	Key            string
	EndpointSecret string
	Log            *zap.Logger
}

type PaymentHandler struct {
	FirebaseConnection *firebasemanager.FirebaseConnection
	StripeConnection   *StripeConnection
	MongoConnection    *mongomanager.MongoConnection
	EMailConnection    *emailmanager.EMailConnection
	Log                *zap.Logger
}

func InitStripeConnection(ctx context.Context, log *zap.Logger) (*StripeConnection, error) {
	log.Info("Init Stripe Connection")
	key, err := secret.GetStripeKey()
	if err != nil {
		return nil, err
	}
	endpointSecret, err := secret.GetStripeEndpointSecret()
	if err != nil {
		return nil, err
	}
	stripeConnection := &StripeConnection{
		Key:            key,
		EndpointSecret: endpointSecret,
		Log:            log.Named("stripeconnection"),
	}
	return stripeConnection, nil
}
