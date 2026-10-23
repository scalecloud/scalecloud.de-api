package stripemanager

import (
	"context"
	"fmt"

	"github.com/scalecloud/scalecloud.de-api/authentikmanager"
	"github.com/scalecloud/scalecloud.de-api/emailmanager"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/newslettermanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager/secret"
	"github.com/stripe/stripe-go/v87"
	"go.uber.org/zap"
)

type StripeConnection struct {
	Key            string
	EndpointSecret string
	Log            *zap.Logger
}

type PaymentHandler struct {
	AuthentikConnection  *authentikmanager.AuthentikManager
	InvitationClaimURL   string
	StripeConnection     *StripeConnection
	MongoDBConnection    *mongodbmanager.MongoDBConnection
	EMailConnection      *emailmanager.EMailConnection
	NewsletterConnection *newslettermanager.NewsletterConnection
	Log                  *zap.Logger
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

func (stripeConnection *StripeConnection) CheckHealth(ctx context.Context) error {
	client := stripe.NewClient(stripeConnection.Key)
	prices := client.V1Prices.List(ctx, &stripe.PriceListParams{
		ListParams: stripe.ListParams{Limit: stripe.Int64(1), Single: true},
	})
	for _, err := range prices.All(ctx) {
		if err != nil {
			return fmt.Errorf("stripe prices API health check failed: %w", err)
		}
	}
	if err := prices.Err(); err != nil {
		return fmt.Errorf("stripe prices API health check failed: %w", err)
	}
	return nil
}
