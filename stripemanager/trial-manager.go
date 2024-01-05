package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v76"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) getTrialDaysForCustomer(c context.Context, quantity int64, paymentMethod *stripe.PaymentMethod, product *stripe.Product, customer *stripe.Customer) (int64, error) {
	if quantity != 1 {
		return 0, errors.New("Quantity is not 1 therefore no trial period is possible.")
	}
	err := paymentHandler.hadTrialBefore(c, paymentMethod, product, customer)
	if err != nil {
		return 0, err
	}
	iTrialPeriodDays := int64(0)
	stripe.Key = paymentHandler.StripeConnection.Key
	if product == nil {
		return iTrialPeriodDays, errors.New("Product is nil")
	}
	if customer == nil {
		return iTrialPeriodDays, errors.New("Customer is nil")
	}
	metaDataProduct := product.Metadata
	trialPeriodDays, ok := metaDataProduct["trialPeriodDays"]
	if !ok {
		return iTrialPeriodDays, errors.New("trialPeriodDays not found for product: " + product.ID)
	}
	iTrialPeriodDays, err = strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		paymentHandler.Log.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return iTrialPeriodDays, errors.New("Error converting trialPeriodDays")
	}
	return iTrialPeriodDays, nil
}

func (paymentHandler *PaymentHandler) hadTrialBefore(ctx context.Context, paymentMethod *stripe.PaymentMethod, product *stripe.Product, customer *stripe.Customer) error {
	metaDataProduct := product.Metadata
	productType, ok := metaDataProduct["productType"]
	if !ok {
		return errors.New("trialPeriodDays not found for product: " + product.ID)
	}
	cardFingerprint := ""
	if paymentMethod.Card != nil {
		cardFingerprint = paymentMethod.Card.Fingerprint
	}
	paypalPayerEmail := ""
	if paymentMethod.Paypal != nil {
		paypalPayerEmail = paymentMethod.Paypal.PayerEmail
	}
	sepaDebitFingerprint := ""
	if paymentMethod.SEPADebit != nil {
		sepaDebitFingerprint = paymentMethod.SEPADebit.Fingerprint
	}
	filter := mongomanager.Trial{
		ProductType:            productType,
		CustomerID:             customer.ID,
		PaymentCardFingerprint: cardFingerprint,
		PaymentPayPalEMail:     paypalPayerEmail,
		PaymentSEPAFingerprint: sepaDebitFingerprint,
	}
	trialSearch, err := paymentHandler.MongoConnection.GetTrial(ctx, filter)
	if err != nil {
		return err
	}
	if trialSearch.CustomerID != "" {
		return errors.New("CustomerID matched. Customer used trial before. CustomerID: " + trialSearch.CustomerID)
	} else if trialSearch.PaymentCardFingerprint != "" {
		return errors.New("PaymentCardFingerprint matched. Customer used trial before. PaymentCardFingerprint: " + trialSearch.PaymentCardFingerprint)
	} else if trialSearch.PaymentPayPalEMail != "" {
		return errors.New("PaymentPayPalEMail matched. Customer used trial before. PaymentPayPalEMail: " + trialSearch.PaymentPayPalEMail)
	} else if trialSearch.PaymentSEPAFingerprint != "" {
		return errors.New("PaymentSEPAFingerprint matched. Customer used trial before. PaymentSEPAFingerprint: " + trialSearch.PaymentSEPAFingerprint)
	} else {
		paymentHandler.Log.Debug("Customer did not match. Customer did not use trial before.", zap.String("Product", product.Name), zap.String("CustomerID", customer.ID))
	}
	return nil
}
