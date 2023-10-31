package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
)

func (paymentHandler *PaymentHandler) GetPaymentMethodOverview(c context.Context, tokenDetails firebasemanager.TokenDetails) (PaymentMethodOverviewReply, error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return PaymentMethodOverviewReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	cus, err := getCustomerByID(c, customerID)
	if err != nil {
		return PaymentMethodOverviewReply{}, err
	}
	if cus.InvoiceSettings == nil {
		return PaymentMethodOverviewReply{}, errors.New("InvoiceSettings not found")
	}
	defaultPaymentMethod := cus.InvoiceSettings.DefaultPaymentMethod
	if defaultPaymentMethod == nil {
		return PaymentMethodOverviewReply{}, errors.New("DefaultPaymentMethod not found")
	}
	defaultPaymentID := defaultPaymentMethod.ID
	if defaultPaymentID == "" {
		return PaymentMethodOverviewReply{}, errors.New("DefaultPaymentMethodID is empty")
	}
	pm, err := paymentHandler.StripeConnection.getPaymentMethod(c, defaultPaymentID)
	if err != nil {
		return PaymentMethodOverviewReply{}, err
	}
	if string(pm.Type) == string(stripe.PaymentMethodType(stripe.PaymentMethodTypeCard)) {
		brand := string(pm.Card.Brand)
		reply := PaymentMethodOverviewReply{
			Type: string(pm.Type),
			PaymentMethodOverviewCard: PaymentMethodOverviewCard{
				Brand:    brand,
				Last4:    pm.Card.Last4,
				ExpMonth: uint64(pm.Card.ExpMonth),
				ExpYear:  uint64(pm.Card.ExpYear),
			},
		}
		return reply, nil
	} else if string(pm.Type) == string(stripe.PaymentMethodType(stripe.PaymentMethodTypeSEPADebit)) {
		reply := PaymentMethodOverviewReply{
			Type: string(pm.SEPADebit.BankCode),
			PaymentMethodOverviewSEPADebit: PaymentMethodOverviewSEPADebit{
				BankCode: pm.SEPADebit.BankCode,
				Branch:   pm.SEPADebit.BranchCode,
				Country:  pm.SEPADebit.Country,
				Last4:    pm.SEPADebit.Last4,
			},
		}
		return reply, nil
	} else if string(pm.Type) == string(stripe.PaymentMethodType(stripe.PaymentMethodTypePaypal)) {
		reply := PaymentMethodOverviewReply{
			Type: string(pm.Type),
			PaymentMethodOverviewPayPal: PaymentMethodOverviewPayPal{
				Email: pm.Paypal.PayerEmail,
			},
		}
		return reply, nil
	}
	return PaymentMethodOverviewReply{}, errors.New("Payment method not found")
}
