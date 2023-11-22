package mongomanager

type Trial struct {
	SubscriptionID         string `bson:"subscriptionID" index:"unique"`
	ProductType            string `bson:"productType"`
	CustomerID             string `bson:"customerID,omitempty" index:"unique"`
	PaymentCardFingerprint string `bson:"paymentCardFingerprint,omitempty" index:"unique"`
	PaymentPayPalEMail     string `bson:"paymentPayPalEMail,omitempty" index:"unique"`
	PaymentSEPAFingerprint string `bson:"paymentSEPAFingerprint,omitempty" index:"unique"`
}
