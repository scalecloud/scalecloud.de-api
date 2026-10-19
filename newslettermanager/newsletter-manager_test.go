package newslettermanager

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/scalecloud/scalecloud.de-api/emailmanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"go.uber.org/zap"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "valid address", value: "person@example.com", valid: true},
		{name: "invalid address", value: "not-an-email"},
		{name: "empty address", value: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(test *testing.T) {
			if got := IsValidEmail(testCase.value); got != testCase.valid {
				t.Fatalf("IsValidEmail(%q) = %t, want %t", testCase.value, got, testCase.valid)
			}
		})
	}
}

func TestCanSendVerificationEmail(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		sentAt  *time.Time
		wantErr bool
	}{
		{name: "never sent", sentAt: nil},
		{name: "cooldown active", sentAt: timePointer(now.Add(-cooldownDuration / 2)), wantErr: true},
		{name: "cooldown expired", sentAt: timePointer(now.Add(-cooldownDuration - time.Second))},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(test *testing.T) {
			err := CanSendVerificationEmail(testCase.sentAt)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("CanSendVerificationEmail() error = %v, wantErr %t", err, testCase.wantErr)
			}
		})
	}
}

func TestGenerateVerificationToken(t *testing.T) {
	token, err := generateVerificationToken()
	if err != nil {
		t.Fatalf("generate verification token: %v", err)
	}
	if strings.ContainsAny(token, "+/") {
		t.Fatalf("token is not URL-safe: %q", token)
	}
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("decode verification token: %v", err)
	}
	if len(decoded) != 64 {
		t.Fatalf("decoded token has %d bytes, want 64", len(decoded))
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func TestNewsletterSubscribeCreatesPendingSubscriberAndSendsToken(t *testing.T) {
	store := &fakeNewsletterStore{}
	mailer := &fakeNewsletterMailer{}
	handler := newTestNewsletterConnection(store, mailer)

	reply, err := handler.NewsletterSubscribe(t.Context(), NewsletterSubscribeRequest{EMail: "person@example.com"})
	if err != nil {
		t.Fatalf("NewsletterSubscribe() error = %v", err)
	}
	if reply.NewsletterSubscribeReplyStatus != NewsletterSubscribeReplyStatusSuccess {
		t.Fatalf("reply status = %q, want success", reply.NewsletterSubscribeReplyStatus)
	}
	if store.createCalls != 1 || store.subscriber.Status != mongomanager.NewsletterStatusPending {
		t.Fatalf("subscriber was not created pending: calls=%d subscriber=%+v", store.createCalls, store.subscriber)
	}
	if store.subscriber.VerificationToken == "" || store.subscriber.UnsubscribeToken == "" || store.subscriber.VerificationToken == store.subscriber.UnsubscribeToken {
		t.Fatalf("expected distinct verification and unsubscribe tokens: %+v", store.subscriber)
	}
	if mailer.calls != 1 || mailer.message.To[0] != "person@example.com" {
		t.Fatalf("confirmation email was not sent: calls=%d message=%+v", mailer.calls, mailer.message)
	}
	if !strings.Contains(mailer.message.Body, store.subscriber.VerificationToken) {
		t.Fatal("confirmation email does not contain the stored verification token")
	}
}

func TestNewsletterSubscribeDoesNotResendToActiveSubscriber(t *testing.T) {
	store := &fakeNewsletterStore{
		existing: true,
		subscriber: mongomanager.NewsletterSubscriber{
			EMail:  "person@example.com",
			Status: mongomanager.NewsletterStatusActive,
		},
	}
	mailer := &fakeNewsletterMailer{}
	reply, err := newTestNewsletterConnection(store, mailer).NewsletterSubscribe(t.Context(), NewsletterSubscribeRequest{EMail: "person@example.com"})
	if err != nil {
		t.Fatalf("NewsletterSubscribe() error = %v", err)
	}
	if reply.NewsletterSubscribeReplyStatus != NewsletterSubscribeReplyStatusSuccess || mailer.calls != 0 || store.updateCalls != 0 {
		t.Fatalf("active resubscribe was not idempotent: reply=%+v mailCalls=%d updateCalls=%d", reply, mailer.calls, store.updateCalls)
	}
}

func TestNewsletterSubscribeRateLimitsPendingResend(t *testing.T) {
	sentAt := time.Now().Add(-cooldownDuration / 2)
	store := &fakeNewsletterStore{
		existing: true,
		subscriber: mongomanager.NewsletterSubscriber{
			EMail:                   "person@example.com",
			Status:                  mongomanager.NewsletterStatusPending,
			VerificationToken:       "saved-verification-token",
			VerificationTokenSentAt: sentAt,
		},
	}
	mailer := &fakeNewsletterMailer{}
	reply, err := newTestNewsletterConnection(store, mailer).NewsletterSubscribe(t.Context(), NewsletterSubscribeRequest{EMail: "person@example.com"})
	if err != nil {
		t.Fatalf("NewsletterSubscribe() error = %v", err)
	}
	if reply.NewsletterSubscribeReplyStatus != NewsletterSubscribeReplyStatusRateLimited || mailer.calls != 0 || store.updateCalls != 0 {
		t.Fatalf("pending resubscribe was not rate limited: reply=%+v mailCalls=%d updateCalls=%d", reply, mailer.calls, store.updateCalls)
	}
}

func TestNewsletterConfirmActivatesPendingSubscriber(t *testing.T) {
	store := &fakeNewsletterStore{
		subscriber: mongomanager.NewsletterSubscriber{
			EMail:  "person@example.com",
			Status: mongomanager.NewsletterStatusPending,
		},
	}
	reply, err := newTestNewsletterConnection(store, &fakeNewsletterMailer{}).NewsletterConfirm(t.Context(), NewsletterConfirmRequest{VerificationToken: "valid-token"})
	if err != nil {
		t.Fatalf("NewsletterConfirm() error = %v", err)
	}
	if reply.Confirmed == nil || !*reply.Confirmed {
		t.Fatalf("confirmation reply = %+v, want confirmed", reply)
	}
	if store.updateCalls != 1 || store.updated.Status != mongomanager.NewsletterStatusActive || store.updated.ConfirmedAt.IsZero() {
		t.Fatalf("subscriber was not activated with a confirmation timestamp: %+v", store.updated)
	}
}

func TestNewsletterConfirmIsIdempotentForActiveSubscriber(t *testing.T) {
	store := &fakeNewsletterStore{subscriber: mongomanager.NewsletterSubscriber{Status: mongomanager.NewsletterStatusActive}}
	reply, err := newTestNewsletterConnection(store, &fakeNewsletterMailer{}).NewsletterConfirm(t.Context(), NewsletterConfirmRequest{VerificationToken: "already-used"})
	if err != nil {
		t.Fatalf("NewsletterConfirm() error = %v", err)
	}
	if reply.Confirmed == nil || !*reply.Confirmed || store.updateCalls != 0 {
		t.Fatalf("active confirmation was not idempotent: reply=%+v updateCalls=%d", reply, store.updateCalls)
	}
}

func TestNewsletterUnsubscribeDeletesSubscriber(t *testing.T) {
	store := &fakeNewsletterStore{subscriber: mongomanager.NewsletterSubscriber{EMail: "person@example.com"}}
	reply, err := newTestNewsletterConnection(store, &fakeNewsletterMailer{}).NewsletterUnsubscribe(t.Context(), NewsletterUnsubscribeRequest{UnsubscribeToken: "valid-token"})
	if err != nil {
		t.Fatalf("NewsletterUnsubscribe() error = %v", err)
	}
	if reply.NewsletterUnsubscribeReplyStatus != NewsletterUnsubscribeReplyStatusUnsubscribed || store.deleteCalls != 1 {
		t.Fatalf("unsubscribe did not delete subscriber: reply=%+v deleteCalls=%d", reply, store.deleteCalls)
	}
}

func TestNewsletterUnsubscribeReturnsNotFoundWithoutDeleting(t *testing.T) {
	store := &fakeNewsletterStore{}
	reply, err := newTestNewsletterConnection(store, &fakeNewsletterMailer{}).NewsletterUnsubscribe(t.Context(), NewsletterUnsubscribeRequest{UnsubscribeToken: "unknown-token"})
	if err != nil {
		t.Fatalf("NewsletterUnsubscribe() error = %v", err)
	}
	if reply.NewsletterUnsubscribeReplyStatus != NewsletterUnsubscribeReplyStatusNotFound || store.deleteCalls != 0 {
		t.Fatalf("unknown token handling = reply %+v, deleteCalls=%d", reply, store.deleteCalls)
	}
}

func newTestNewsletterConnection(store *fakeNewsletterStore, mailer *fakeNewsletterMailer) *NewsletterConnection {
	return &NewsletterConnection{mongoHandler: store, eMailConnection: mailer, log: zap.NewNop()}
}

type fakeNewsletterStore struct {
	subscriber  mongomanager.NewsletterSubscriber
	existing    bool
	getErr      error
	createErr   error
	updateErr   error
	deleteErr   error
	createCalls int
	updateCalls int
	deleteCalls int
	updated     mongomanager.NewsletterSubscriber
}

func (store *fakeNewsletterStore) GetNewsletterSubscriber(context.Context, string) (mongomanager.NewsletterSubscriber, error) {
	if store.getErr != nil {
		return mongomanager.NewsletterSubscriber{}, store.getErr
	}
	if !store.existing {
		return mongomanager.NewsletterSubscriber{}, nil
	}
	return store.subscriber, nil
}

func (store *fakeNewsletterStore) CreateNewsletterSubscriber(_ context.Context, subscriber mongomanager.NewsletterSubscriber) error {
	store.createCalls++
	store.subscriber = subscriber
	return store.createErr
}

func (store *fakeNewsletterStore) UpdateNewsletterSubscriber(_ context.Context, subscriber mongomanager.NewsletterSubscriber) error {
	store.updateCalls++
	store.updated = subscriber
	store.subscriber = subscriber
	return store.updateErr
}

func (store *fakeNewsletterStore) GetNewsletterSubscriberByVerificationToken(context.Context, string) (mongomanager.NewsletterSubscriber, error) {
	return store.subscriber, store.getErr
}

func (store *fakeNewsletterStore) GetNewsletterSubscriberByUnsubscribeToken(context.Context, string) (mongomanager.NewsletterSubscriber, error) {
	if !store.existing && store.subscriber == (mongomanager.NewsletterSubscriber{}) {
		return mongomanager.NewsletterSubscriber{}, nil
	}
	return store.subscriber, store.getErr
}

func (store *fakeNewsletterStore) DeleteNewsletterSubscriber(context.Context, mongomanager.NewsletterSubscriber) error {
	store.deleteCalls++
	return store.deleteErr
}

type fakeNewsletterMailer struct {
	message emailmanager.EMail
	calls   int
	err     error
}

func (mailer *fakeNewsletterMailer) SendEMail(message emailmanager.EMail) error {
	mailer.calls++
	mailer.message = message
	return mailer.err
}
