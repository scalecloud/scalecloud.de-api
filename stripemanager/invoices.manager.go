package stripemanager

import (
	"context"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/invoice"
)

func (paymentHandler *PaymentHandler) GetSubscriptionInvoices(c context.Context, tokenDetails firebasemanager.TokenDetails, request ListInvoicesRequest) (ListInvoicesReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleBilling})
	if err != nil {
		return ListInvoicesReply{}, err
	}
	totalResults, err := paymentHandler.CountTotalInvoices(request.SubscriptionID)
	if err != nil {
		return ListInvoicesReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	params := &stripe.InvoiceListParams{
		Customer: stripe.String(request.SubscriptionID),
	}
	params.Limit = stripe.Int64(int64(request.PageSize))
	if request.PageIndex > 0 {
		startingAfter := request.PageIndex * request.PageSize
		params.StartingAfter = stripe.String(strconv.Itoa(startingAfter))
	}
	iter := invoice.List(params)
	var invoices []Invoice
	for iter.Next() {
		inv := iter.Invoice()
		invoices = append(invoices, Invoice{
			InvoiceID:        inv.ID,
			SubscriptionID:   inv.Customer.ID,
			Created:          inv.Created,
			Total:            inv.Total,
			Currency:         string(inv.Currency),
			Status:           inv.Status,
			HostedInvoiceUrl: inv.HostedInvoiceURL,
		})
	}
	if err := iter.Err(); err != nil {
		return ListInvoicesReply{}, err
	}
	reply := ListInvoicesReply{
		SubscriptionID: request.SubscriptionID,
		Invoices:       invoices,
		PageIndex:      request.PageIndex,
		TotalResults:   totalResults,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) CountTotalInvoices(subscriptionID string) (int64, error) {
	stripe.Key = paymentHandler.StripeConnection.Key

	params := &stripe.InvoiceListParams{
		Customer: stripe.String(subscriptionID),
	}
	params.Limit = stripe.Int64(1)
	var totalResults int64
	var iter *invoice.Iter
	for {
		iter = invoice.List(params)
		if !iter.Next() {
			break
		}
		totalResults++
		params.StartingAfter = stripe.String(iter.Invoice().ID)
	}
	if err := iter.Err(); err != nil {
		return 0, err
	}

	return totalResults, nil
}
