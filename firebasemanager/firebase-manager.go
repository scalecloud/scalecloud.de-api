package firebasemanager

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"

	firebase "firebase.google.com/go/v4"
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
	firebaseAccountKey := "./keys/firebase-serviceAccountKey.json"
	if fileExists(firebaseAccountKey) {
		log.Info("keyfile exists. ", zap.String("file", firebaseAccountKey))
	} else {
		return nil, errors.New("required file does not exist: " + firebaseAccountKey)
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(firebaseAccountKey))
	if err != nil {
		return nil, err
	}
	return app, nil
}
