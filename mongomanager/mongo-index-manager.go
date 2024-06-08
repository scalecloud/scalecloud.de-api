package mongomanager

func (mongoConnection *MongoConnection) EnsureIndexes() error {
	err := mongoConnection.ensureSeatIndex()
	if err != nil {
		return err
	}
	mongoConnection.Log.Info("All required indexes are present.")
	return nil
}
