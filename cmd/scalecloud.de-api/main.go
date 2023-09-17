package main

import (
	"github.com/scalecloud/scalecloud.de-api/apimanager"
	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripe/secret"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting App.")
	firebasemanager.InitFirebase()
	mongomanager.InitMongo()
	secret.InitStripe()
	apimanager.InitAPI()
	logger.Info("App ended.")
}
