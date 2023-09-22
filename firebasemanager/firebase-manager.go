package firebasemanager

import (
	"context"
	"errors"
	"os"

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

func (firebaseConnection *FirebaseConnection) VerifyIDToken(ctx context.Context, jwtToken string) bool {
	ret := false
	if jwtToken == "" {
		firebaseConnection.log.Error("ID token is empty")
	} else {
		client, err := firebaseConnection.firebaseApp.Auth(ctx)
		if err != nil {
			firebaseConnection.log.Error("Error initializing app", zap.Error(err))
		} else {
			token, err := client.VerifyIDTokenAndCheckRevoked(ctx, jwtToken)
			if err != nil {
				firebaseConnection.log.Error("Error verifying ID token", zap.Error(err))
			} else {
				firebaseConnection.log.Debug("Token is valid.", zap.Any("UID:", token.UID))
				ret = true
			}
		}
	}
	return ret
}

func (firebaseConnection *FirebaseConnection) GetTokenDetails(ctx context.Context, jwtToken string) (tokenDetails TokenDetails, err error) {
	if jwtToken == "" {
		return TokenDetails{}, errors.New("Token is empty")
	}
	client, err := firebaseConnection.firebaseApp.Auth(ctx)
	if err != nil {
		firebaseConnection.log.Error("Error initializing app", zap.Error(err))
		return TokenDetails{}, errors.New("Error initializing app")
	}
	idToken, err := client.VerifyIDTokenAndCheckRevoked(ctx, jwtToken)
	if err != nil {
		firebaseConnection.log.Error("Error verifying ID token", zap.Error(err))
		return TokenDetails{}, errors.New("Error verifying ID token")
	}
	uid := idToken.UID
	if uid == "" {
		return TokenDetails{}, errors.New("UID is empty")
	}

	email, err := firebaseConnection.getEMailFromToken(idToken)
	if err != nil {
		return TokenDetails{}, errors.New("Error getting email from token")
	}
	token := TokenDetails{
		UID:   uid,
		EMail: email,
	}
	return token, nil
}

func (m *FirebaseConnection) getEMailFromToken(idToken *auth.Token) (string, error) {
	if idToken.Claims == nil {
		return "", errors.New("claims is nil")
	}
	email := idToken.Claims["email"].(string)
	if email == "" {
		return "", errors.New("email is empty")
	}
	return email, nil
}
