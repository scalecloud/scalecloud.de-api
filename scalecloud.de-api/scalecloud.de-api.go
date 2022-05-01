package scalecloud

import (
	"context"
	"os"

	"github.com/scalecloud/scalecloud.de-api/tree/main/firebase"
	"github.com/scalecloud/scalecloud.de-api/tree/main/mongo"
	"github.com/scalecloud/scalecloud.de-api/tree/main/stripe"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

type subscription struct {
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	SubscriptionArticelID string  `json:"artist"`
	PricePerMonth         float64 `json:"pricepermonth"`
	Started               string  `json:"Started"`
	EndsOn                string  `json:"EndsOn"`
}

var subscriptionsPlaceholder = []subscription{
	{
		ID:                    "sub_INYwS5uFiirGNs",
		Title:                 "Ruby",
		SubscriptionArticelID: "si_INYwzY0bSrDTHX",
		PricePerMonth:         10.00,
		Started:               "2022-01-01",
		EndsOn:                "2022-12-31",
	},
	{
		ID:                    "sub_123abc",
		Title:                 "Jade",
		SubscriptionArticelID: "si_aaa111",
		PricePerMonth:         15.00,
		Started:               "2021-01-01",
		EndsOn:                "2023-05-31",
	},
}

func Init() {
	logger.Info("Init scalecloud.de-api")
	firebase.InitFirebase()
	mongo.InitMongo()
	stripe.InitStripe()
}

func IsAuthenticated(ctx context.Context, uid string, token string) bool {
	return firebase.VerifyIDToken(ctx, uid, token)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetDashboardSubscriptions(c context.Context) (subscriptions []subscription, err error) {
	logger.Info("GetDashboardSubscriptions")
	return subscriptionsPlaceholder, nil
}
