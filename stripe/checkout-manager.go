package stripe

import (
	"net/http"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"go.uber.org/zap"
)

func createConnection() {
	// This is a public sample test API key.
	// Don’t submit any personally identifiable information in requests made with this key.
	// Sign in to see your own test API key embedded in code samples.
	stripe.Key = getStripeKey()

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
