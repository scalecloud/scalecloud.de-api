package stripemanager

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) GetMyPermission(c context.Context, tokenDetails authmanager.TokenDetails, request PermissionRequest) (PermissionReply, error) {
	mySeat, err := paymentHandler.MongoConnection.GetSeat(c, request.SubscriptionID, tokenDetails.UID)
	if err != nil {
		return PermissionReply{}, err
	}
	if mySeat.UID == "" {
		paymentHandler.Log.Warn("user with UID " + tokenDetails.UID + " tried to access subscriptionID " + request.SubscriptionID + " but has no seat")
		return PermissionReply{}, errors.New(http.StatusText(http.StatusForbidden))
	}
	reply := PermissionReply{
		MySeat: mySeat,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) GetSubscriptionListSeats(c context.Context, tokenDetails authmanager.TokenDetails, request ListSeatRequest) (ListSeatReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return ListSeatReply{}, err
	}
	totalResults, err := paymentHandler.MongoConnection.CountSeats(c, request.SubscriptionID)
	if err != nil {
		return ListSeatReply{}, err
	}
	if totalResults == 0 {
		return ListSeatReply{}, errors.New("no seats found")
	}
	pagedSeats, err := paymentHandler.MongoConnection.GetSeats(c, request.SubscriptionID, request.PageIndex, request.PageSize)
	if err != nil {
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
		Seats:          pagedSeats,
		PageIndex:      request.PageIndex,
		TotalResults:   totalResults,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) GetSubscriptionSeatDetail(c context.Context, tokenDetails authmanager.TokenDetails, request SeatDetailRequest) (SeatDetailReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return SeatDetailReply{}, err
	}
	selectedSeat, err := paymentHandler.MongoConnection.GetSeat(c, request.SubscriptionID, request.UID)
	if err != nil {
		return SeatDetailReply{}, err
	}
	mySeat, err := paymentHandler.MongoConnection.GetSeat(c, request.SubscriptionID, tokenDetails.UID)
	if err != nil {
		return SeatDetailReply{}, err
	}
	reply := SeatDetailReply{
		SelectedSeat: selectedSeat,
		MySeat:       mySeat,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) GetSubscriptionUpdateSeat(c context.Context, tokenDetails authmanager.TokenDetails, request UpdateSeatDetailRequest) (UpdateSeatDetailReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SeatUpdated.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return UpdateSeatDetailReply{}, err
	}
	err = paymentHandler.handleOwnerTransfer(c, tokenDetails, request.SeatUpdated)
	if err != nil {
		return UpdateSeatDetailReply{}, err
	}
	err = paymentHandler.MongoConnection.UpdateSeat(c, request.SeatUpdated)
	if err != nil {
		return UpdateSeatDetailReply{}, err
	}
	updatedSeat, err := paymentHandler.MongoConnection.GetSeat(c, request.SeatUpdated.SubscriptionID, request.SeatUpdated.UID)
	if err != nil {
		return UpdateSeatDetailReply{}, err
	}
	reply := UpdateSeatDetailReply{
		Seat: updatedSeat,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) GetSubscriptionAddSeat(c context.Context, tokenDetails authmanager.TokenDetails, request AddSeatRequest) (AddSeatReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return AddSeatReply{}, err
	}
	parsedEmail, err := mail.ParseAddress(request.EMail)
	if err != nil {
		return AddSeatReply{}, errors.New("E-Mail is invalid")
	}
	request.EMail = strings.ToLower(parsedEmail.Address)
	if len(request.Roles) == 0 {
		return AddSeatReply{}, errors.New("no role selected")
	}
	for _, role := range request.Roles {
		switch role {
		case mongomanager.RoleAdministrator, mongomanager.RoleUser, mongomanager.RoleBilling:
		default:
			return AddSeatReply{}, errors.New("invalid role for invited seat")
		}
	}
	seats, err := paymentHandler.MongoConnection.GetAllSeats(c, request.SubscriptionID)
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
	return paymentHandler.CreateSeatInvitation(c, request.SubscriptionID, request.EMail, tokenDetails.EMail, request.Roles, quantity)
}

func containsEmail(seats []mongomanager.Seat, email string) bool {
	if len(seats) == 0 {
		return false
	}
	for _, seat := range seats {
		if strings.EqualFold(strings.TrimSpace(seat.EMail), strings.TrimSpace(email)) {
			return true
		}
	}
	return false
}

func seatAvailable(seats []mongomanager.Seat, quantity int64) bool {
	return int64(len(seats)) < quantity
}

func (paymentHandler *PaymentHandler) GetSubscriptionRemoveSeat(c context.Context, tokenDetails authmanager.TokenDetails, request DeleteSeatRequest) (DeleteSeatReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SeatToDelete.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return DeleteSeatReply{}, err
	}
	seatToRemove, err := paymentHandler.MongoConnection.GetSeat(c, request.SeatToDelete.SubscriptionID, request.SeatToDelete.UID)
	if request.SeatToDelete.UID == "" && request.SeatToDelete.EMail != "" {
		seatToRemove, err = paymentHandler.MongoConnection.GetSeatByEmail(c, request.SeatToDelete.SubscriptionID, strings.ToLower(strings.TrimSpace(request.SeatToDelete.EMail)))
	}
	if err != nil {
		return DeleteSeatReply{}, err
	}
	if mongomanager.ContainsRole(seatToRemove, []mongomanager.Role{mongomanager.RoleOwner}) {
		return DeleteSeatReply{}, errors.New("cannot remove owner")
	}
	if seatToRemove.Status == mongomanager.SeatStatusPending {
		err = paymentHandler.MongoConnection.DeletePendingSeat(c, seatToRemove.SubscriptionID, seatToRemove.EMail)
	} else {
		err = paymentHandler.RemoveSeatFromSubscriptionGroup(c, seatToRemove)
		if err == nil {
			err = paymentHandler.MongoConnection.DeleteSeat(c, seatToRemove)
			if err != nil {
				if restoreErr := paymentHandler.AddSeatToSubscriptionGroup(c, seatToRemove); restoreErr != nil {
					paymentHandler.Log.Error("Could not restore Authentik membership after seat delete failed", zap.Error(restoreErr))
				}
			}
		}
	}
	if err != nil {
		return DeleteSeatReply{}, err
	}
	reply := DeleteSeatReply{
		DeletedSeat: seatToRemove,
		Success:     true,
	}
	return reply, nil
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
