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
		paymentHandler.Log.Info("Quantity is not 1 therefore no trial period is possible.", zap.Int64("Quantity", quantity))
		return -1, nil
	}
	err := paymentHandler.hadTrialBefore(c, paymentMethod, product, customer)
	if err != nil {
		paymentHandler.Log.Info("Customer had trial before. No trial period is possible.", zap.Error(err))
		return -1, nil
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	metaDataProduct := product.Metadata
	trialPeriodDays, ok := metaDataProduct["trialPeriodDays"]
	if !ok {
		return 0, errors.New("trialPeriodDays not found for product: " + product.ID)
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		paymentHandler.Log.Error("Error converting trialPeriodDays to int", zap.Error(err))
		return 0, errors.New("Error converting trialPeriodDays")
	}
	return iTrialPeriodDays, nil
}

func (paymentHandler *PaymentHandler) hadTrialBefore(ctx context.Context, paymentMethod *stripe.PaymentMethod, product *stripe.Product, customer *stripe.Customer) error {
	if paymentMethod == nil {
		return errors.New("PaymentMethod is nil")
	}
	if product == nil {
		return errors.New("Product is nil")
	}
	if customer == nil {
		return errors.New("Customer is nil")
	}
	metaDataProduct := product.Metadata
	if metaDataProduct == nil {
		return errors.New("Product metadata not found")
	}
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
	if trialSearch == (mongomanager.Trial{}) {
		paymentHandler.Log.Info("Customer did not use trial before.", zap.String("CustomerID", customer.ID), zap.String("ProductType", productType))
		return nil
	} else if trialSearch.CustomerID != "" {
		return errors.New("CustomerID matched. Customer used trial before. CustomerID: " + trialSearch.CustomerID + " ProductType:" + productType)
	} else if trialSearch.PaymentCardFingerprint != "" {
		return errors.New("PaymentCardFingerprint matched. Customer used trial before. PaymentCardFingerprint: " + trialSearch.PaymentCardFingerprint + " ProductType:" + productType)
	} else if trialSearch.PaymentPayPalEMail != "" {
		return errors.New("PaymentPayPalEMail matched. Customer used trial before. PaymentPayPalEMail: " + trialSearch.PaymentPayPalEMail + " ProductType:" + productType)
	} else if trialSearch.PaymentSEPAFingerprint != "" {
		return errors.New("PaymentSEPAFingerprint matched. Customer used trial before. PaymentSEPAFingerprint: " + trialSearch.PaymentSEPAFingerprint + " ProductType:" + productType)
	} else {
		paymentHandler.Log.Warn("No match found for trial search. This should not happen.", zap.Any("TrialSearch", trialSearch))
	}
	return nil
}
