package main

import (
	"github.com/scalecloud/scalecloud.de-api/apimanager"
	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripesecret"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func main() {
	logger.Info("Starting App.")
	firebasemanager.InitFirebase()
	mongomanager.InitMongo()
	stripesecret.InitStripe()
	apimanager.InitAPI()
	logger.Info("App ended.")
}
