package newslettermanager

type NewsletterSubscribeReplyStatus string

const (
	NewsletterSubscribeReplyStatusSuccess      NewsletterSubscribeReplyStatus = "success"
	NewsletterSubscribeReplyStatusInvalidEmail NewsletterSubscribeReplyStatus = "invalid_email"
)

type NewsletterSubscribeRequest struct {
	EMail string `json:"email" validate:"required,email"`
}

type NewsletterSubscribeReply struct {
	NewsletterSubscribeReplyStatus NewsletterSubscribeReplyStatus `json:"newsletterSubscribeReplyStatus" validate:"required"`
	EMail                          string                         `json:"email" validate:"required,email"`
}

type NewsletterConfirmRequest struct {
	VerificationToken string `json:"verificationToken" validate:"required"`
}

type NewsletterConfirmReply struct {
	Confirmed *bool `json:"confirmed" validate:"required"`
}

type NewsletterUnsubscribeRequest struct {
	UnsubscribeToken string `json:"unsubscribeToken" validate:"required"`
}

type NewsletterUnsubscribeReply struct {
	Unsubscribed *bool `json:"unsubscribed" validate:"required"`
}
