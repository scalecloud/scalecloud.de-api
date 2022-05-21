package stripe

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
)

var logger, _ = zap.NewProduction()

var subscriptionsPlaceholder = []Subscription{
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

var subscriptionDetailPlaceholder = []SubscriptionDetail{
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

func InitStripe() {
	logger.Info("Init stripe")
}

func createConnection() {
	// This is a public sample test API key.
	// Don’t submit any personally identifiable information in requests made with this key.
	// Sign in to see your own test API key embedded in code samples.
	stripe.Key = "sk_test_API_KEY"

	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/create-checkout-session", createCheckoutSession)
	addr := "localhost:4242"
	logger.Info("Stripe server listening on", zap.String("addr", addr))

	logger.Error("Error creating session", zap.Any("ListenAndServe:", http.ListenAndServe(addr, nil)))
}

func createCheckoutSession(w http.ResponseWriter, r *http.Request) {
	domain := "http://localhost:4242"
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				// Provide the exact Price ID (for example, pr_1234) of the product you want to sell
				Price:    stripe.String("{{PRICE_ID}}"),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(domain + "/success.html"),
		CancelURL:  stripe.String(domain + "/cancel.html"),
	}

	s, err := session.New(params)

	if err != nil {
		logger.Error("Error creating session", zap.Error(err))
	}

	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}

func GetDashboardSubscriptions(c context.Context, customer string) (subscriptions []Subscription, err error) {
	logger.Info("GetDashboardSubscriptions")
	return subscriptionsPlaceholder, nil
}

func GetSubscriptionByID(c context.Context, id, customer string) (subscriptionDetail SubscriptionDetail, err error) {
	logger.Info("GetSubscriptionByID")
	for _, sub := range subscriptionDetailPlaceholder {
		if sub.ID == id {
			return sub, nil
		}
	}
	return SubscriptionDetail{}, errors.New("Subscription not found")
}
