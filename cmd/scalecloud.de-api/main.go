package main

import (
	"github.com/scalecloud/scalecloud.de-api/api"
	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/scalecloud/scalecloud.de-api/mongo"
	"github.com/scalecloud/scalecloud.de-api/stripe"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting App.")
	firebase.InitFirebase()
	mongo.InitMongo()
	stripe.InitStripe()
	api.InitAPI()
	logger.Info("App ended.")
}
