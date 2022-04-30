package firebase

import (
	"context"
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

	token := ""
	uid := ""
	ctx := context.Background()
	app := InitializeAppDefault(ctx)
	if verifyIDToken(ctx, app, token, uid) {
		logger.Info("Token is valid.")
	} else {
		logger.Info("Token is not valid.")
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

func verifyIDToken(ctx context.Context, app *firebase.App, awtToken string, uid string) bool {
	ret := false
	if awtToken == "" {
		logger.Error("ID token is empty")
	} else if uid == "" {
		logger.Error("UID is empty")
	} else {
		client, err := app.Auth(ctx)
		if err != nil {
			logger.Error("Error initializing app", zap.Error(err))
		} else {
			token, err := client.VerifyIDTokenAndCheckRevoked(ctx, awtToken)
			if err != nil {
				logger.Error("Error verifying ID token", zap.Error(err))
			} else if token.UID != uid {
				logger.Error("UID does not match with Token", zap.Any("Token:", token))
			} else if token.UID == uid {
				logger.Info("Token is valid and matches UID.", zap.Any("Token:", token))
				ret = true
			}
		}
	}
	return ret
}
