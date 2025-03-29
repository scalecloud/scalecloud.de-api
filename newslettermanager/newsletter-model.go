package newslettermanager

type NewsletterSubscribeReplyStatus string

const (
	NewsletterSubscribeReplyStatusSuccess      NewsletterSubscribeReplyStatus = "success"
	NewsletterSubscribeReplyStatusInvalidEmail NewsletterSubscribeReplyStatus = "invalid_email"
	NewsletterSubscribeReplyStatusRateLimited  NewsletterSubscribeReplyStatus = "rate_limited"
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

type NewsletterUnsubscribeReplyStatus string

const (
	NewsletterUnsubscribeReplyStatusUnsubscribed NewsletterUnsubscribeReplyStatus = "unsubscribed"
	NewsletterUnsubscribeReplyStatusNotFound     NewsletterUnsubscribeReplyStatus = "not_found"
)

type NewsletterUnsubscribeReply struct {
	NewsletterUnsubscribeReplyStatus NewsletterUnsubscribeReplyStatus `json:"newsletterUnsubscribeReplyStatus" validate:"required"`
}
