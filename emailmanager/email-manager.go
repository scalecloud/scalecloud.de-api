package emailmanager

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

type smtpCredentials struct {
	Host     string `json:"host" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	From     string `json:"from" validate:"required,email"`
}

type EMailConnection struct {
	Client *mail.Client
	From   string
	Log    *zap.Logger
}

type EMail struct {
	To      []string
	Subject string
	Body    string
}

func InitEMailConnection(log *zap.Logger) (*EMailConnection, error) {
	log.Info("Init mail handler")
	smtpConnection, err := readSMTPCredentials(log)
	if err != nil {
		return nil, err
	}

	client, err := mail.NewClient(
		smtpConnection.Host,
		mail.WithPort(smtpConnection.Port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(smtpConnection.Username),
		mail.WithPassword(smtpConnection.Password),
	)
	if err != nil {
		return nil, err
	}

	eMailHandler := &EMailConnection{
		Client: client,
		From:   smtpConnection.From,
		Log:    log.Named("emailhandler"),
	}

	return eMailHandler, nil
}

func readSMTPCredentials(log *zap.Logger) (*smtpCredentials, error) {
	port, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SMTP_PORT")))
	if err != nil {
		return nil, errors.New("SMTP_PORT must be a valid integer")
	}
	smtpConnection := &smtpCredentials{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     port,
		Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}

	validate := validator.New()
	err = validate.Struct(smtpConnection)
	if err != nil {
		return nil, errors.New("Incomplete or invalid SMTP credentials: " + err.Error())
	}

	log.Info("SMTP connection read", zap.String("host", smtpConnection.Host))
	return smtpConnection, nil
}

func (eMailConnection *EMailConnection) SendEMail(email EMail) error {
	m := mail.NewMsg()
	if err := m.From(eMailConnection.From); err != nil {
		eMailConnection.Log.Error("Failed to set From address", zap.Error(err))
		return err
	}
	if err := m.To(email.To...); err != nil {
		eMailConnection.Log.Error("Failed to set To address", zap.Error(err))
		return err
	}
	m.Subject(email.Subject)
	m.SetBodyString(mail.TypeTextHTML, email.Body)

	if err := eMailConnection.Client.DialAndSend(m); err != nil {
		eMailConnection.Log.Error("Failed to send email", zap.Error(err))
		return err
	}

	eMailConnection.Log.Info("Email sent successfully")
	return nil
}
