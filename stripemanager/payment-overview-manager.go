package stripemanager

import (
	"context"
	"errors"
	"strings"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
)

func (paymentHandler *PaymentHandler) GetPaymentMethodOverview(c context.Context, tokenDetails firebasemanager.TokenDetails) (PaymentMethodOverviewReply, error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return PaymentMethodOverviewReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	cus, err := GetCustomerByID(c, customerID)
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
	pm, err := paymentHandler.StripeConnection.GetPaymentMethod(c, defaultPaymentID)
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
			Type: string(pm.Type),
			PaymentMethodOverviewSEPADebit: PaymentMethodOverviewSEPADebit{
				Country: pm.SEPADebit.Country,
				Last4:   pm.SEPADebit.Last4,
			},
		}
		return reply, nil
	} else if string(pm.Type) == string(stripe.PaymentMethodType(stripe.PaymentMethodTypePaypal)) {
		reply := PaymentMethodOverviewReply{
			Type: string(pm.Type),
			PaymentMethodOverviewPayPal: PaymentMethodOverviewPayPal{
				Email: maskEMail(pm.Paypal.PayerEmail),
			},
		}
		return reply, nil
	}
	return PaymentMethodOverviewReply{}, errors.New("Payment method not found")
}

func maskEMail(email string) string {
	ret := "****"
	if len(email) > 4 && strings.Contains(email, "@") {
		emailSplit := strings.Split(email, "@")
		addr := emailSplit[0]
		domain := emailSplit[1]
		if len(addr) > 3 {
			addr = addr[:3] + "****"
		} else if len(addr) > 2 {
			addr = addr[:2] + "****"
		} else {
			addr = addr[:1] + "****"
		}
		ret = addr + "@" + domain
	}
	return ret
}
