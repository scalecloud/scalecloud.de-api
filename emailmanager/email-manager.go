package emailmanager

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type smtpCredentials struct {
	Host     string `json:"host" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	From     string `json:"from" validate:"required,email"`
}

type EMailConnection struct {
	Dialer *gomail.Dialer
	From   string
	Log    *zap.Logger
}

type Email struct {
	To      []string
	Subject string
	Body    string
}

func InitEMailConnection(log *zap.Logger) (*EMailConnection, error) {
	log.Info("Init mail handler")
	smtpConnection, err := initDialer(log)
	if err != nil {
		return nil, err
	}

	dialer := gomail.NewDialer(smtpConnection.Host, smtpConnection.Port, smtpConnection.Username, smtpConnection.Password)

	eMailHandler := &EMailConnection{
		Dialer: dialer,
		From:   smtpConnection.From,
		Log:    log.Named("emailhandler"),
	}

	return eMailHandler, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func initDialer(log *zap.Logger) (*smtpCredentials, error) {
	credentialsFile := "./keys/smtp-credentials.json"
	if !fileExists(credentialsFile) {
		return nil, errors.New("required file does not exist: " + credentialsFile)
	}

	file, err := os.Open(credentialsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	smtpConnection := &smtpCredentials{}
	err = json.NewDecoder(file).Decode(smtpConnection)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	err = validate.Struct(smtpConnection)
	if err != nil {
		return nil, errors.New("Incomplete or invalid SMTP credentials: " + err.Error())
	}

	log.Info("SMTP connection read", zap.String("host", smtpConnection.Host))
	return smtpConnection, nil
}

func (eMailConnection *EMailConnection) SendEMail(email Email) error {
	m := gomail.NewMessage()
	m.SetHeader("From", eMailConnection.From)
	m.SetHeader("To", email.To...)
	m.SetHeader("Subject", email.Subject)
	m.SetBody("text/html", email.Body)

	if err := eMailConnection.Dialer.DialAndSend(m); err != nil {
		eMailConnection.Log.Error("Failed to send email", zap.Error(err))
		return err
	}

	eMailConnection.Log.Info("Email sent successfully")
	return nil
}
