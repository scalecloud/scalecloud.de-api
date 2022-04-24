package scalecloud

import (
	"os"

	"github.com/scalecloud/scalecloud.de-api/tree/main/firebase"
	"github.com/scalecloud/scalecloud.de-api/tree/main/mongo"
	"github.com/scalecloud/scalecloud.de-api/tree/main/stripe"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func Init() {
	logger.Info("Init scalecloud.de-api")
	firebase.InitFirebase()
	mongo.InitMongo()
	stripe.InitStripe()
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
