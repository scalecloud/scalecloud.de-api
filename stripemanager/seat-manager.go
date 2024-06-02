package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
)

func (paymentHandler *PaymentHandler) GetSubscriptionListSeat(c context.Context, tokenDetails firebasemanager.TokenDetails, request ListSeatRequest) (ListSeatReply, error) {
	if request.SubscriptionID == "" {
		return ListSeatReply{}, errors.New("subscription ID is empty")
	}
	filter := mongomanager.SeatFilter{
		SubscriptionID: request.SubscriptionID,
	}
	emails, err := paymentHandler.MongoConnection.GetSeatsEMail(c, filter)
	if err != nil {
		return ListSeatReply{}, err
	}
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if err != nil {
		return ListSeatReply{}, errors.New("subscription not found")
	}
	productID := subscription.Items.Data[0].Price.Product.ID
	product, err := paymentHandler.StripeConnection.GetProduct(c, productID)
	if err != nil {
		return ListSeatReply{}, errors.New("product not found")
	}
	metaData := product.Metadata
	productType, ok := metaData["productType"]
	if !ok {
		return ListSeatReply{}, errors.New("productType not found")
	}

	quantity := subscription.Items.Data[0].Quantity
	if quantity == 0 {
		return ListSeatReply{}, errors.New("quantity is 0")
	}

	reply := ListSeatReply{
		SubscriptionID: request.SubscriptionID,
		ProductName:    product.Name,
		ProductType:    productType,
		MaxSeats:       quantity,
		EMails:         emails,
	}
	return reply, nil
}
