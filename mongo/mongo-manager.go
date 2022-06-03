package mongo

import (
	"os"

	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

const x509 = "keys/mongodb-atlas.pem"
const connectionString = "keys/mongodb-atlas-connection-string.txt"

func InitMongo() {
	logger.Info("Init Mongo")
	if fileExists(connectionString) {
		logger.Debug("connectionString exists. ", zap.String("file", connectionString))
	} else {
		logger.Error("connectionString does not exist. ", zap.String("file", connectionString))
		os.Exit(1)
	}
	if fileExists(x509) {
		logger.Debug("x509 exists. ", zap.String("file", x509))
	} else {
		logger.Error("x509 does not exist. ", zap.String("file", x509))
		os.Exit(1)
	}
	initMongoStripe()
}
