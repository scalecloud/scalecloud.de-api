package firebase

import (
	"context"
	"log"

	"go.uber.org/zap"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var logger, _ = zap.NewProduction()

func InitFirebase() {
	logger.Info("Init firebase")
	verifyIDToken(context.Background(), InitializeAppDefault(), "potato")
}

func InitializeAppDefault() *firebase.App {

	opt := option.WithCredentialsFile("./keys/firebase-serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		logger.Error("Error initializing app", zap.Error(err))
		return nil
	}
	logger.Info("Firebase app initialized successfully")

	return app
}

func verifyIDToken(ctx context.Context, app *firebase.App, idToken string) bool {
	// [START verify_id_token_golang]
	client, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Fatalf("error verifying ID token: %v\n", err)
	}

	log.Printf("Verified ID token: %v\n", token)
	// [END verify_id_token_golang]

	return true
}
