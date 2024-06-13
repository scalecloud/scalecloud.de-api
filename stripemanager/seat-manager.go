package stripemanager

import (
	"context"
	"errors"
	"net/mail"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
)

func (paymentHandler *PaymentHandler) checkAccess(tokenDetails firebasemanager.TokenDetails, seats []mongomanager.Seat, subscriptionID string) error {
	if !containsEmail(seats, tokenDetails.EMail) {
		paymentHandler.Log.Error("user with UID " + tokenDetails.UID + " is not allowed to access subscriptionID " + subscriptionID)
		return errors.New("access denied")
	}
	return nil
}

func (paymentHandler *PaymentHandler) GetSubscriptionListSeats(c context.Context, tokenDetails firebasemanager.TokenDetails, request ListSeatRequest) (ListSeatReply, error) {
	if request.SubscriptionID == "" {
		return ListSeatReply{}, errors.New("subscriptionID is empty")
	}
	seats, err := paymentHandler.MongoConnection.GetSeats(c, request.SubscriptionID)
	if err != nil {
		return ListSeatReply{}, err
	}
	err = paymentHandler.checkAccess(tokenDetails, seats, request.SubscriptionID)
	if err != nil {
		return ListSeatReply{}, err
	}
	emails, err := extractEmails(seats)
	if err != nil {
		paymentHandler.Log.Error("there should always be at least one seat")
		return ListSeatReply{}, err
	}
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if err != nil {
		return ListSeatReply{}, errors.New("subscription not found")
	}
	quantity := subscription.Items.Data[0].Quantity
	if quantity == 0 {
		return ListSeatReply{}, errors.New("quantity is 0")
	}
	reply := ListSeatReply{
		SubscriptionID: request.SubscriptionID,
		MaxSeats:       quantity,
		EMails:         emails,
	}
	return reply, nil
}

func extractEmails(seats []mongomanager.Seat) ([]string, error) {
	if seats != nil && len(seats) == 0 {
		return []string{}, errors.New("no seats found")
	}
	emails := make([]string, len(seats))
	for i, seat := range seats {
		emails[i] = seat.EMail
	}
	return emails, nil
}

func (paymentHandler *PaymentHandler) GetSubscriptionAddSeat(c context.Context, tokenDetails firebasemanager.TokenDetails, request AddSeatRequest) (AddSeatReply, error) {
	if request.SubscriptionID == "" {
		return AddSeatReply{}, errors.New("subscriptionID is empty")
	}
	if !IsValidEmail(request.EMail) {
		return AddSeatReply{}, errors.New("email is invalid")
	}
	if len(request.Roles) == 0 {
		return AddSeatReply{}, errors.New("no role selected")
	}
	seats, err := paymentHandler.MongoConnection.GetSeats(c, request.SubscriptionID)
	if err != nil {
		return AddSeatReply{}, err
	}
	err = paymentHandler.checkAccess(tokenDetails, seats, request.SubscriptionID)
	if err != nil {
		return AddSeatReply{}, err
	}
	exists := containsEmail(seats, request.EMail)
	if exists {
		return AddSeatReply{}, errors.New("seat already exists")
	}
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if err != nil {
		return AddSeatReply{}, errors.New("subscription not found")
	}
	quantity := subscription.Items.Data[0].Quantity
	if quantity == 0 {
		return AddSeatReply{}, errors.New("quantity is 0")
	}
	if !seatAvailable(seats, quantity) {
		return AddSeatReply{}, errors.New("already used all seats")
	}
	err = paymentHandler.FirebaseConnection.InviteSeat(c, request.EMail)
	if err != nil {
		return AddSeatReply{}, err
	}
	seat := mongomanager.Seat{
		SubscriptionID: request.SubscriptionID,
		EMail:          request.EMail,
		Roles:          request.Roles,
	}
	err = paymentHandler.MongoConnection.CreateSeat(c, seat)
	if err != nil {
		return AddSeatReply{}, err
	}
	paymentHandler.Log.Error("Invite E-Mail should be sent to " + request.EMail)
	reply := AddSeatReply{
		SubscriptionID: request.SubscriptionID,
		Success:        true,
		EMail:          request.EMail,
	}
	return reply, nil
}

func containsEmail(seats []mongomanager.Seat, email string) bool {
	if len(seats) == 0 {
		return false
	}
	for _, seat := range seats {
		if seat.EMail == email {
			return true
		}
	}
	return false
}

func seatAvailable(seats []mongomanager.Seat, quantity int64) bool {
	return int64(len(seats)) < quantity
}

func (paymentHandler *PaymentHandler) GetSubscriptionRemoveSeat(c context.Context, tokenDetails firebasemanager.TokenDetails, request RemoveSeatRequest) (RemoveSeatReply, error) {
	if request.SubscriptionID == "" {
		return RemoveSeatReply{}, errors.New("subscriptionID is empty")
	}
	seats, err := paymentHandler.MongoConnection.GetSeats(c, request.SubscriptionID)
	if err != nil {
		return RemoveSeatReply{}, err
	}
	err = paymentHandler.checkAccess(tokenDetails, seats, request.SubscriptionID)
	if err != nil {
		return RemoveSeatReply{}, err
	}

	reply := RemoveSeatReply{
		SubscriptionID: request.SubscriptionID,
		Success:        true,
		EMail:          request.EMail,
	}
	return reply, nil
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
