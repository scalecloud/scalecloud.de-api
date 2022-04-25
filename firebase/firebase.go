package firebase

import (
	"context"

	"go.uber.org/zap"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var logger, _ = zap.NewProduction()

func InitFirebase() {
	logger.Info("Init firebase")
}

func InitializeAppDefault() *firebase.App {

	opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		logger.Error("Error initializing app", zap.Error(err))
		return nil
	}

	return app
}
