package stripemanager

import (
	"testing"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/stripe/stripe-go/v87"
	"go.uber.org/zap"
)

func TestIsSeatUpdateOwnerTransfer(t *testing.T) {
	tests := []struct {
		name  string
		roles []mongodbmanager.Role
		want  bool
	}{
		{name: "owner role", roles: []mongodbmanager.Role{mongodbmanager.RoleOwner}, want: true},
		{name: "administrator role", roles: []mongodbmanager.Role{mongodbmanager.RoleAdministrator}},
		{name: "no roles"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			seat := mongodbmanager.Seat{Roles: testCase.roles}
			if got := isSeatUpdateOwnerTransfer(seat); got != testCase.want {
				t.Fatalf("isSeatUpdateOwnerTransfer() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestHasOwnerTriggeredOwnerTransfer(t *testing.T) {
	ownerSeat := mongodbmanager.Seat{UID: "owner-uid"}
	tests := []struct {
		name    string
		uid     string
		wantErr bool
	}{
		{name: "current owner", uid: "owner-uid"},
		{name: "different user", uid: "other-uid", wantErr: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := hasOwnerTriggeredOwnerTransfer(authmanager.TokenDetails{UID: testCase.uid}, ownerSeat)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("hasOwnerTriggeredOwnerTransfer() error = %v, wantErr %t", err, testCase.wantErr)
			}
		})
	}
}

func TestIsOwnerSeatUpdate(t *testing.T) {
	tests := []struct {
		name       string
		ownerUID   string
		requestUID string
		roles      []mongodbmanager.Role
		want       bool
	}{
		{name: "same owner", ownerUID: "owner", requestUID: "owner", roles: []mongodbmanager.Role{mongodbmanager.RoleOwner}, want: true},
		{name: "different user", ownerUID: "owner", requestUID: "other", roles: []mongodbmanager.Role{mongodbmanager.RoleOwner}},
		{name: "owner role removed", ownerUID: "owner", requestUID: "owner", roles: []mongodbmanager.Role{mongodbmanager.RoleUser}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			owner := mongodbmanager.Seat{UID: testCase.ownerUID}
			request := mongodbmanager.Seat{UID: testCase.requestUID, Roles: testCase.roles}
			if got := isOwnerSeatUpdate(owner, request); got != testCase.want {
				t.Fatalf("isOwnerSeatUpdate() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestHandleSubscriptionStatusError(t *testing.T) {
	paymentHandler := &PaymentHandler{Log: zap.NewNop()}
	tests := []struct {
		status stripe.SubscriptionStatus
		want   string
	}{
		{status: stripe.SubscriptionStatusCanceled, want: "ownership cannot be transferred if the subscription is canceled"},
		{status: stripe.SubscriptionStatusIncomplete, want: "ownership cannot be transferred if the subscription is incomplete"},
		{status: stripe.SubscriptionStatusPastDue, want: "ownership cannot be transferred if the subscription is past due"},
		{status: stripe.SubscriptionStatusPaused, want: "ownership cannot be transferred if the subscription is paused"},
		{status: stripe.SubscriptionStatusTrialing, want: "ownership cannot be transferred if the subscription is in trial period"},
		{status: stripe.SubscriptionStatusUnpaid, want: "ownership cannot be transferred if the subscription is unpaid"},
		{status: "unknown", want: "ownership cannot be transferred, please contact support"},
	}
	for _, testCase := range tests {
		t.Run(string(testCase.status), func(t *testing.T) {
			err := paymentHandler.handleSubscriptionStatusError(testCase.status)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("handleSubscriptionStatusError(%q) = %v, want %q", testCase.status, err, testCase.want)
			}
		})
	}
}
