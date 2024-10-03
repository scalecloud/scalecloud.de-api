package stripemanager

import "github.com/stripe/stripe-go/v80"

type Invoice struct {
	InvoiceID        string               `json:"invoiceID" validate:"required"`
	SubscriptionID   string               `json:"subscriptionID" validate:"required"`
	Created          int64                `json:"created" validate:"required"`
	Total            int64                `json:"total" validate:"required"`
	Currency         string               `json:"currency" validate:"required"`
	Status           stripe.InvoiceStatus `json:"status" validate:"required"`
	HostedInvoiceUrl string               `json:"hosted_invoice_url" validate:"required"`
}

type ListInvoicesRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	PageSize       int    `json:"pageSize" validate:"gte=1"`
	StartingAfter  string `json:"startingAfter"`
	EndingBefore   string `json:"endingBefore"`
}

type ListInvoicesReply struct {
	SubscriptionID string    `json:"subscriptionID" validate:"required"`
	Invoices       []Invoice `json:"invoices" validate:"required"`
	TotalResults   int64     `json:"totalResults" validate:"gte=1"`
}
