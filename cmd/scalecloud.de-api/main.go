package main

import (
	"github.com/scalecloud/scalecloud.de-api/api"
	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting App.")
	firebasemanager.InitFirebase()
	mongomanager.InitMongo()
	stripemanager.InitStripe()
	api.InitAPI()
	logger.Info("App ended.")
}
