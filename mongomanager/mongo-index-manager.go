package mongomanager

func (mongoConnection *MongoConnection) EnsureIndexes() error {
	err := mongoConnection.ensureSeatIndex()
	if err != nil {
		return err
	}
	err = mongoConnection.ensureNewsletterIndex()
	if err != nil {
		return err
	}
	mongoConnection.Log.Info("all required indexes are present")
	return nil
}
