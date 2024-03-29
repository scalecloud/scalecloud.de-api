package stripemanager

type PaymentMethodOverviewCard struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth uint64 `json:"exp_month"`
	ExpYear  uint64 `json:"exp_year"`
}

type PaymentMethodOverviewSEPADebit struct {
	Country string `json:"country"`
	Last4   string `json:"last4"`
}

type PaymentMethodOverviewPayPal struct {
	Email string `json:"email"`
}

type PaymentMethodOverviewReply struct {
	HasValidPaymentMethod          *bool                          `json:"has_valid_payment_method" validate:"required"`
	Type                           string                         `json:"type" validate:"required"`
	PaymentMethodOverviewCard      PaymentMethodOverviewCard      `json:"card,omitempty"`
	PaymentMethodOverviewSEPADebit PaymentMethodOverviewSEPADebit `json:"sepa_debit,omitempty"`
	PaymentMethodOverviewPayPal    PaymentMethodOverviewPayPal    `json:"paypal,omitempty"`
}
