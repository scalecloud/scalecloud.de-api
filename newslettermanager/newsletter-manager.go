package newslettermanager

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/mail"
	"time"

	"github.com/scalecloud/scalecloud.de-api/emailmanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"go.uber.org/zap"
)

type NewsletterConnection struct {
	mongoHandler    *mongomanager.MongoConnection
	eMailConnection *emailmanager.EMailConnection
	log             *zap.Logger
}

func InitNewsletterConnection(ctx context.Context, log *zap.Logger, mongoHandler *mongomanager.MongoConnection, eMailConnection *emailmanager.EMailConnection) (*NewsletterConnection, error) {
	log.Info("Init Newsletter Connection")
	stripeConnection := &NewsletterConnection{
		mongoHandler:    mongoHandler,
		eMailConnection: eMailConnection,
		log:             log.Named("newsletterconnection"),
	}
	return stripeConnection, nil
}

const cooldownDuration = 10 * time.Minute

func (newsletterHandler NewsletterConnection) NewsletterSubscribe(c context.Context, request NewsletterSubscribeRequest) (NewsletterSubscribeReply, error) {
	if !IsValidEmail(request.EMail) {
		newsletterHandler.log.Warn("Invalid E-Mail tried to subscribe to newsletter: " + request.EMail)
		reply := NewsletterSubscribeReply{
			NewsletterSubscribeReplyStatus: NewsletterSubscribeReplyStatusInvalidEmail,
			EMail:                          request.EMail,
		}
		return reply, nil
	}
	newsletterSubscriber, err := newsletterHandler.mongoHandler.GetNewsletterSubscriber(c, request.EMail)
	if err != nil {
		newsletterHandler.log.Error(
			"Error while checking if E-Mail is already subscribed to newsletter",
			zap.String("email", request.EMail),
			zap.Error(err),
		)
		return NewsletterSubscribeReply{}, errors.New("Error while checking if E-Mail is already subscribed to newsletter: " + request.EMail)
	}
	if newsletterSubscriber == (mongomanager.NewsletterSubscriber{}) {
		return newsletterHandler.newsletterSubscribeWithEntryNotFound(c, request)
	} else {
		return newsletterHandler.newsletterSubscribeWithEntryFound(c, request, newsletterSubscriber)
	}
}

func (newsletterHandler NewsletterConnection) newsletterSubscribeWithEntryFound(c context.Context, request NewsletterSubscribeRequest, newsletterSubscriber mongomanager.NewsletterSubscriber) (NewsletterSubscribeReply, error) {
	newsletterHandler.log.Warn("E-Mail is already subscribed to newsletter and tried to subscribe again: " + request.EMail)
	if newsletterSubscriber.Status == mongomanager.NewsletterStatusActive {
		newsletterHandler.log.Info("E-Mail is already subscribed to newsletter: " + request.EMail)
		reply := NewsletterSubscribeReply{
			NewsletterSubscribeReplyStatus: NewsletterSubscribeReplyStatusSuccess,
			EMail:                          request.EMail,
		}
		return reply, nil
	}
	if newsletterSubscriber.Status == mongomanager.NewsletterStatusBounced {
		newsletterHandler.log.Warn("Newsletter subscriber tried to subscribe again after E-Mail bounced: " + request.EMail)
	}
	if newsletterSubscriber.Status == mongomanager.NewsletterStatusPending {
		newsletterHandler.log.Warn("Newsletter subscriber tried to subscribe again after E-Mail was already pending: " + request.EMail)
	}
	err := CanSendVerificationEmail(&newsletterSubscriber.VerificationTokenSentAt)
	if err != nil {
		newsletterHandler.log.Warn("Verification E-Mail was sent recently at: " + newsletterSubscriber.VerificationTokenSentAt.String())
		return NewsletterSubscribeReply{
			NewsletterSubscribeReplyStatus: NewsletterSubscribeReplyStatusRateLimited,
			EMail:                          request.EMail,
		}, nil
	}
	if newsletterSubscriber.VerificationToken == "" {
		newsletterHandler.log.Warn("Verification token is empty for newsletter subscriber: " + request.EMail)
		verificationToken, err := generateVerificationToken()
		if err != nil {
			return NewsletterSubscribeReply{}, err
		}
		newsletterSubscriber.VerificationToken = verificationToken
	}
	err = newsletterHandler.sendConfirmationMail(request.EMail, newsletterSubscriber.VerificationToken)
	if err != nil {
		return NewsletterSubscribeReply{}, err
	}
	timestamp := time.Now()
	newsletterSubscriber.VerificationTokenSentAt = timestamp
	newsletterSubscriber.LastUpdated = timestamp
	err = newsletterHandler.mongoHandler.UpdateNewsletterSubscriber(c, newsletterSubscriber)
	if err != nil {
		newsletterHandler.log.Error("Error while updating newsletter subscriber",
			zap.String("email", request.EMail),
			zap.Error(err))
		return NewsletterSubscribeReply{}, errors.New("Error while updating newsletter subscriber: " + request.EMail)
	}
	reply := NewsletterSubscribeReply{
		NewsletterSubscribeReplyStatus: NewsletterSubscribeReplyStatusSuccess,
		EMail:                          request.EMail,
	}
	return reply, nil
}

func (newsletterHandler NewsletterConnection) newsletterSubscribeWithEntryNotFound(c context.Context, request NewsletterSubscribeRequest) (NewsletterSubscribeReply, error) {
	verificationToken, err := generateVerificationToken()
	if err != nil {
		return NewsletterSubscribeReply{}, err
	}
	unsubscribeToken, err := generateVerificationToken()
	if err != nil {
		return NewsletterSubscribeReply{}, err
	}
	err = newsletterHandler.sendConfirmationMail(request.EMail, verificationToken)
	if err != nil {
		return NewsletterSubscribeReply{}, err
	}
	timestamp := time.Now()
	newsletterSubscriber := mongomanager.NewsletterSubscriber{
		EMail:                   request.EMail,
		Status:                  mongomanager.NewsletterStatusPending,
		SubscribedAt:            timestamp,
		VerificationToken:       verificationToken,
		VerificationTokenSentAt: timestamp,
		UnsubscribeToken:        unsubscribeToken,
		LastUpdated:             timestamp,
	}
	err = newsletterHandler.mongoHandler.CreateNewsletterSubscriber(c, newsletterSubscriber)
	if err != nil {
		newsletterHandler.log.Error("Error while creating newsletter subscriber",
			zap.String("email", request.EMail),
			zap.Error(err))
		return NewsletterSubscribeReply{}, errors.New("Error while creating newsletter subscriber: " + request.EMail)
	}
	reply := NewsletterSubscribeReply{
		NewsletterSubscribeReplyStatus: NewsletterSubscribeReplyStatusSuccess,
		EMail:                          request.EMail,
	}
	return reply, nil
}

func CanSendVerificationEmail(sentAt *time.Time) error {
	if sentAt == nil {
		return nil
	}
	cooldownEnd := sentAt.Add(cooldownDuration)
	if time.Now().Before(cooldownEnd) {
		return errors.New("verification E-Mail was sent recently, please wait before trying again")
	}
	return nil
}

func (newsletterHandler NewsletterConnection) NewsletterConfirm(c context.Context, request NewsletterConfirmRequest) (NewsletterConfirmReply, error) {
	newsletterSubscriber, err := newsletterHandler.mongoHandler.GetNewsletterSubscriberByVerificationToken(c, request.VerificationToken)
	if err != nil {
		newsletterHandler.log.Error(
			"Error while searching for newsletter subscriber by verification token",
			zap.String("verificationToken", request.VerificationToken),
			zap.Error(err),
		)
		return NewsletterConfirmReply{}, errors.New("error while searching for newsletter subscriber")
	}
	if newsletterSubscriber == (mongomanager.NewsletterSubscriber{}) {
		newsletterHandler.log.Warn("Verification token not found in newsletter database: " + request.VerificationToken)
		return NewsletterConfirmReply{}, errors.New("no matching entry was found in the newsletter database. Please register again for the newsletter")
	}
	if newsletterSubscriber.Status == mongomanager.NewsletterStatusActive {
		newsletterHandler.log.Warn("Newsletter subscriber is already confirmed: " + request.VerificationToken)
		confirmed := true
		reply := NewsletterConfirmReply{
			Confirmed: &confirmed,
		}
		return reply, nil
	}

	timestamp := time.Now()
	newsletterSubscriber.Status = mongomanager.NewsletterStatusActive
	newsletterSubscriber.ConfirmedAt = timestamp
	newsletterSubscriber.LastUpdated = timestamp

	err = newsletterHandler.mongoHandler.UpdateNewsletterSubscriber(c, newsletterSubscriber)
	if err != nil {
		newsletterHandler.log.Error("error while updating newsletter subscriber",
			zap.String("email", newsletterSubscriber.EMail),
			zap.Error(err))
		return NewsletterConfirmReply{}, errors.New("error while updating newsletter subscriber, please try again later")
	}
	confirmed := true
	reply := NewsletterConfirmReply{
		Confirmed: &confirmed,
	}
	return reply, nil
}

func (newsletterHandler NewsletterConnection) NewsletterUnsubscribe(c context.Context, request NewsletterUnsubscribeRequest) (NewsletterUnsubscribeReply, error) {
	newsletterSubscriber, err := newsletterHandler.mongoHandler.GetNewsletterSubscriberByUnsubscribeToken(c, request.UnsubscribeToken)
	if err != nil {
		newsletterHandler.log.Error(
			"Error while searching for newsletter subscriber by unsubscribe token",
			zap.String("unsubscribeToken", request.UnsubscribeToken),
			zap.Error(err),
		)
		return NewsletterUnsubscribeReply{}, errors.New("error while searching for newsletter subscriber")
	}
	if (newsletterSubscriber == mongomanager.NewsletterSubscriber{}) {
		newsletterHandler.log.Warn("Tried to unsubscribe but no matching entry was found in the newsletter database: " + request.UnsubscribeToken)
		unsubscribed := false
		reply := NewsletterUnsubscribeReply{
			Unsubscribed: &unsubscribed,
		}
		return reply, nil
	}

	err = newsletterHandler.mongoHandler.DeleteNewsletterSubscriber(c, newsletterSubscriber)
	if err != nil {
		newsletterHandler.log.Error("Error while deleting newsletter subscriber",
			zap.String("email", newsletterSubscriber.EMail),
			zap.Error(err))
		return NewsletterUnsubscribeReply{}, errors.New("error while deleting newsletter subscriber")
	}
	unsubscribed := true
	newsletterHandler.log.Info("Newsleter subscriber was deleted: " + newsletterSubscriber.EMail)
	reply := NewsletterUnsubscribeReply{
		Unsubscribed: &unsubscribed,
	}
	return reply, nil
}

func generateVerificationToken() (string, error) {
	tokenBytes := make([]byte, 64)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", errors.New("failed to generate verification token")
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	return token, nil
}

func (newsletterHandler NewsletterConnection) sendConfirmationMail(email, verificationToken string) error {
	newsletterHandler.log.Info("Sending confirmation E-Mail to: " + email)

	confirmationLink := "https://scalecloud.de/newsletter/confirm/" + verificationToken

	subject := "Please confirm your newsletter subscription"
	body := `
        <html>
        <body>
            <p>Thank you for subscribing to our newsletter!</p>
            <p>Please confirm your E-Mail address by clicking the link below:</p>
            <p><a href="` + confirmationLink + `">Confirm Subscription</a></p>
            <p>If you did not request this, you can safely ignore this E-Mail.</p>
        </body>
        </html>
    `
	emailMessage := emailmanager.EMail{
		To:      []string{email},
		Subject: subject,
		Body:    body,
	}

	err := newsletterHandler.eMailConnection.SendEMail(emailMessage)
	if err != nil {
		newsletterHandler.log.Error("Failed to send confirmation E-Mail", zap.Error(err))
		return err
	}

	newsletterHandler.log.Info("Confirmation E-Mail sent successfully to: " + email)
	return nil
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
