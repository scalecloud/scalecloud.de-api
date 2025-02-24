package email

import (
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type SMTPConnection struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type EMailHandler struct {
	EMailConnection *SMTPConnection
	Log             *zap.Logger
}

type Email struct {
	To      []string
	Subject string
	Body    string
}

func (emailHandler *EMailHandler) sendEMail(email Email) error {
	m := gomail.NewMessage()
	m.SetHeader("From", emailHandler.EMailConnection.From)
	m.SetHeader("To", email.To...)
	m.SetHeader("Subject", email.Subject)
	m.SetBody("text/html", email.Body)

	d := gomail.NewDialer(emailHandler.EMailConnection.Host, emailHandler.EMailConnection.Port, emailHandler.EMailConnection.Username, emailHandler.EMailConnection.Password)

	if err := d.DialAndSend(m); err != nil {
		emailHandler.Log.Error("Failed to send email", zap.Error(err))
		return err
	}

	emailHandler.Log.Info("Email sent successfully")
	return nil
}
