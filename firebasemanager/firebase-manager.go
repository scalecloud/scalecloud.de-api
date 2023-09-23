package firebasemanager

import (
	"context"
	"errors"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type FirebaseConnection struct {
	firebaseApp *firebase.App
	log         *zap.Logger
}

func InitFirebaseConnection(ctx context.Context, log *zap.Logger) (*FirebaseConnection, error) {
	log.Info("Init firebase")
	firebaseApp, err := initFirebaseApp(ctx, log)
	if err != nil {
		return nil, err
	}
	firebaseManager := &FirebaseConnection{
		log:         log.Named("firebasemanager"),
		firebaseApp: firebaseApp,
	}

	return firebaseManager, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func initFirebaseApp(ctx context.Context, log *zap.Logger) (*firebase.App, error) {
	keyFile := "./keys/firebase-serviceAccountKey.json"
	if fileExists(keyFile) {
		log.Info("Keyfile exists. ", zap.String("file", keyFile))
	} else {
		return nil, errors.New("Keyfile does not exist")
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(keyFile))
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (firebaseConnection *FirebaseConnection) VerifyIDToken(ctx context.Context, jwtToken string) error {
	client, err := firebaseConnection.firebaseApp.Auth(ctx)
	if err != nil {
		return err
	}
	token, err := client.VerifyIDTokenAndCheckRevoked(ctx, jwtToken)
	if err != nil {
		return err
	}
	firebaseConnection.log.Debug("Token is valid.", zap.Any("UID:", token.UID))
	return nil
}

func (firebaseConnection *FirebaseConnection) GetTokenDetails(c *gin.Context) (tokenDetails TokenDetails, err error) {
	jwtToken, err := GetBearerToken(c)
	if err != nil {
		return TokenDetails{}, err
	}
	client, err := firebaseConnection.firebaseApp.Auth(c)
	if err != nil {
		return TokenDetails{}, err
	}
	idToken, err := client.VerifyIDToken(c, jwtToken)
	if err != nil {
		return TokenDetails{}, err
	}
	uid := idToken.UID
	if uid == "" {
		return TokenDetails{}, errors.New("UID is empty")
	}

	email, err := firebaseConnection.getEMailFromToken(idToken)
	if err != nil {
		return TokenDetails{}, err
	}
	token := TokenDetails{
		UID:   uid,
		EMail: email,
	}
	return token, nil
}

func GetBearerToken(c *gin.Context) (string, error) {
	token := c.Request.Header.Get("Authorization")
	if token == "" {
		return "", errors.New("Authorization header is missing")
	}
	return token, nil

}

func (m *FirebaseConnection) getEMailFromToken(idToken *auth.Token) (string, error) {
	if idToken.Claims == nil {
		return "", errors.New("claims is nil")
	}
	email := idToken.Claims["email"].(string)
	if email == "" {
		return "", errors.New("E-Mail is empty")
	}
	return email, nil
}
