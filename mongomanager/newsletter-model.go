package mongomanager

import "time"

type NewsletterStatus string

const (
	NewsletterStatusActive  NewsletterStatus = "active"
	NewsletterStatusPending NewsletterStatus = "pending"
	NewsletterStatusBounced NewsletterStatus = "bounced"
)

type NewsletterSubscriber struct {
	EMail                   string           `bson:"email" json:"email" validate:"required,email"`
	Status                  NewsletterStatus `bson:"status" json:"status" validate:"required"`
	SubscribedAt            time.Time        `bson:"subscribedAt" json:"subscribedAt"`
	ConfirmedAt             time.Time        `bson:"confirmedAt,omitempty" json:"confirmedAt,omitempty"`
	VerificationToken       string           `bson:"verificationToken,omitempty" json:"verificationToken,omitempty"`
	VerificationTokenSentAt time.Time        `bson:"verificationTokenSentAt,omitempty" json:"verificationTokenSentAt,omitempty"`
	UnsubscribeToken        string           `bson:"unsubscribeToken,omitempty" json:"unsubscribeToken,omitempty"`
	LastUpdated             time.Time        `bson:"lastUpdated" json:"lastUpdated" validate:"required"`
}
