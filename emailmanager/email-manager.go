package emailmanager

import (
	"encoding/json"
	"errors"
	"os"

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

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func readSMTPCredentials(log *zap.Logger) (*smtpCredentials, error) {
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
