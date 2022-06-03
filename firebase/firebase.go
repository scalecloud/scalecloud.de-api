package firebase

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

const keyFile = "./keys/firebase-serviceAccountKey.json"

var logger, _ = zap.NewProduction()

func InitFirebase() {
	logger.Info("Init firebase")
	if fileExists(keyFile) {
		logger.Info("Keyfile exists. ", zap.String("file", keyFile))
	} else {
		logger.Error("Keyfile does not exist. ", zap.String("file", keyFile))
		os.Exit(1)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func InitializeAppDefault(ctx context.Context) *firebase.App {

	opt := option.WithCredentialsFile(keyFile)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		logger.Error("Error creating new Firebase app.", zap.Error(err))
	}
	return app
}

func VerifyIDToken(ctx context.Context, jwtToken string) bool {
	ret := false
	if jwtToken == "" {
		logger.Error("ID token is empty")
	} else {
		app := InitializeAppDefault(ctx)
		client, err := app.Auth(ctx)
		if err != nil {
			logger.Error("Error initializing app", zap.Error(err))
		} else {
			token, err := client.VerifyIDTokenAndCheckRevoked(ctx, jwtToken)
			if err != nil {
				logger.Error("Error verifying ID token", zap.Error(err))
			} else {
				logger.Debug("Token is valid.", zap.Any("UID:", token.UID))
				ret = true
			}
		}
	}
	return ret
}

func GetTokenDetails(ctx context.Context, jwtToken string) (tokenDetails TokenDetails, err error) {
	if jwtToken == "" {
		logger.Error("Token is empty")
		return TokenDetails{}, errors.New("Token is empty")
	}
	app := InitializeAppDefault(ctx)
	client, err := app.Auth(ctx)
	if err != nil {
		logger.Error("Error initializing app", zap.Error(err))
		return TokenDetails{}, errors.New("Error initializing app")
	}
	idToken, err := client.VerifyIDTokenAndCheckRevoked(ctx, jwtToken)
	if err != nil {
		logger.Error("Error verifying ID token", zap.Error(err))
		return TokenDetails{}, errors.New("Error verifying ID token")
	}
	uid := idToken.UID
	if uid == "" {
		logger.Error("UID is empty")
		return TokenDetails{}, errors.New("UID is empty")
	}
	email := "test@test.de"
	if email == "" {
		logger.Error("Email is empty")
		return TokenDetails{}, errors.New("Email is empty")
	}
	token := TokenDetails{
		UID:   uid,
		Email: email,
	}
	return token, nil
}
