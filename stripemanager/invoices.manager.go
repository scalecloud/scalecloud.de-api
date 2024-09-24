package stripemanager

import (
	"context"
	"errors"

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
	totalResults, err := CountTotalInvoices(request.SubscriptionID)
	if err != nil {
		return ListInvoicesReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	params := &stripe.InvoiceListParams{
		Subscription: stripe.String(request.SubscriptionID),
	}
	params.Limit = stripe.Int64(int64(request.PageSize))
	if request.EndingBefore != "" {
		params.EndingBefore = stripe.String(request.EndingBefore)
	} else if request.StartingAfter != "" {
		params.StartingAfter = stripe.String(request.StartingAfter)
	}
	invoiceList := invoice.List(params).InvoiceList()
	if invoiceList == nil {
		return ListInvoicesReply{}, errors.New("no invoices found")
	}
	var invoices []Invoice
	for _, inv := range invoiceList.Data {
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
	reply := ListInvoicesReply{
		SubscriptionID: request.SubscriptionID,
		Invoices:       invoices,
		TotalResults:   totalResults,
	}
	return reply, nil
}

func CountTotalInvoices(subscriptionID string) (int64, error) {
	var totalResults int64
	params := &stripe.InvoiceListParams{
		Subscription: stripe.String(subscriptionID),
	}
	params.Limit = stripe.Int64(100) // Use a larger limit to reduce the number of API calls

	for {
		iter := invoice.List(params)
		for iter.Next() {
			totalResults++
		}
		if err := iter.Err(); err != nil {
			return 0, err
		}
		if iter.Meta().HasMore {
			params.StartingAfter = stripe.String(iter.Invoice().ID)
		} else {
			break
		}
	}

	return totalResults, nil
}
