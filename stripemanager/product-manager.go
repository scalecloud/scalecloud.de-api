package stripemanager

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/product"
)

func (paymentHandler *PaymentHandler) GetProductTiers(c context.Context, prodType ProductType) (ProductTiersReply, error) {
	stripe.Key = paymentHandler.StripeConnection.Key
	productType := string(prodType)
	query := "active:'true' AND metadata['productType']:'" + productType + "'"
	params := &stripe.ProductSearchParams{
		SearchParams: stripe.SearchParams{
			Query: query,
		},
	}
	var productTiers []ProductTier
	result := product.Search(params)
	for result.Next() {
		product := result.Product()
		metaData := product.Metadata
		if metaData == nil {
			return ProductTiersReply{}, errors.New("product metadata not found")
		}
		storageAmount, ok := metaData["storageAmount"]
		if !ok {
			return ProductTiersReply{}, errors.New("storage amount not found")
		}
		iStorageAmount, err := strconv.Atoi(storageAmount)
		if err != nil {
			return ProductTiersReply{}, errors.New("error converting storage amount to int")
		}
		productTypeMeta, ok := metaData["productType"]
		if !ok {
			return ProductTiersReply{}, errors.New("ProductType not found")
		}
		if productTypeMeta != productType {
			return ProductTiersReply{}, errors.New("ProductType not matching")
		}
		storageUnit, ok := metaData["storageUnit"]
		if !ok {
			return ProductTiersReply{}, errors.New("storage unit not found")
		}
		trialPeriodDays, ok := metaData["trialPeriodDays"]
		if !ok {
			return ProductTiersReply{}, errors.New("trialPeriodDays not found")
		}
		iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
		if err != nil {
			return ProductTiersReply{}, errors.New("error converting trialPeriodDays")
		}
		price, err := paymentHandler.StripeConnection.GetPrice(c, product.ID)
		if err != nil {
			return ProductTiersReply{}, errors.New("price not found")
		}
		productTier := ProductTier{
			ProductType:   prodType,
			ProductID:     product.ID,
			Name:          product.Name,
			StorageAmount: iStorageAmount,
			StorageUnit:   storageUnit,
			TrialDays:     iTrialPeriodDays,
			PricePerMonth: price.UnitAmount,
		}
		productTiers = append(productTiers, productTier)
	}
	productTiers = sortByPrice(productTiers)

	reply := ProductTiersReply{
		ProductType:  prodType,
		ProductTiers: productTiers,
	}
	if len(productTiers) == 0 {
		return reply, errors.New("no product tiers found")
	}
	return reply, nil
}

func sortByPrice(productTiers []ProductTier) []ProductTier {
	sort.Slice(productTiers, func(i, j int) bool {
		return productTiers[i].PricePerMonth < productTiers[j].PricePerMonth
	})
	return productTiers
}

func (stripeConnection *StripeConnection) GetProduct(c context.Context, productID string) (*stripe.Product, error) {
	stripe.Key = stripeConnection.Key
	params := &stripe.ProductParams{}
	product, err := product.Get(productID, params)
	if err != nil {
		return nil, errors.New("product not found")
	}
	return product, nil
}
