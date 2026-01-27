package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/coupon"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/promotioncode"

	"raterunner/internal/config"
)

// SyncResult contains the results of the sync operation
type SyncResult struct {
	ProductsCreated  int
	PricesCreated    int
	PricesArchived   int
	AddonsCreated    int
	CouponsCreated   int
	PromosCreated    int
	Warnings         []string
}

// Sync creates or updates all plans from a billing config in Stripe
func (c *Client) Sync(cfg *config.BillingConfig) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch existing products once
	existingProducts, err := c.FetchProductsWithPrices()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing products: %w", err)
	}

	// Sync plans
	for _, plan := range cfg.Plans {
		if err := c.syncPlan(plan, existingProducts, result); err != nil {
			return result, fmt.Errorf("failed to sync plan '%s': %w", plan.ID, err)
		}
	}

	// Sync addons
	for _, addon := range cfg.Addons {
		if err := c.syncAddon(addon, existingProducts, result); err != nil {
			return result, fmt.Errorf("failed to sync addon '%s': %w", addon.ID, err)
		}
	}

	// Sync promotions
	for _, promo := range cfg.Promotions {
		if err := c.syncPromotion(promo, result); err != nil {
			return result, fmt.Errorf("failed to sync promotion '%s': %w", promo.Code, err)
		}
	}

	return result, nil
}

func (c *Client) syncPlan(plan config.Plan, existingProducts []Product, result *SyncResult) error {
	existingProduct := MatchProduct(existingProducts, plan.ID, plan.Name)

	var productID string
	var existingPrices []ProductPrice

	if existingProduct != nil {
		productID = existingProduct.ID
		existingPrices = existingProduct.Prices

		// Check if name needs update
		if existingProduct.Name != plan.Name {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("plan '%s': product name differs (local='%s', stripe='%s'), not updating",
					plan.ID, plan.Name, existingProduct.Name))
		}
	} else {
		// Create new product with full metadata
		params := &stripe.ProductParams{
			Name: stripe.String(plan.Name),
			Metadata: map[string]string{
				"plan_code": plan.ID,
			},
		}

		// Add description
		if plan.Description != "" {
			params.Description = stripe.String(plan.Description)
		}

		// Add headline to metadata
		if plan.Headline != "" {
			params.Metadata["headline"] = plan.Headline
		}

		// Add plan type to metadata
		if plan.Type != "" {
			params.Metadata["plan_type"] = plan.Type
		}

		// Add marketing features
		if len(plan.Features) > 0 {
			params.MarketingFeatures = make([]*stripe.ProductMarketingFeatureParams, len(plan.Features))
			for i, f := range plan.Features {
				params.MarketingFeatures[i] = &stripe.ProductMarketingFeatureParams{
					Name: stripe.String(f),
				}
			}
		}

		// Add custom metadata
		for k, v := range plan.Metadata {
			if str, ok := v.(string); ok {
				params.Metadata[k] = str
			}
		}

		newProduct, err := product.New(params)
		if err != nil {
			return fmt.Errorf("failed to create product: %w", err)
		}
		productID = newProduct.ID
		result.ProductsCreated++
	}

	// Sync prices
	for interval, localPrice := range plan.Prices {
		if err := c.syncPriceAdvanced(productID, plan.ID, interval, localPrice, plan.TrialDays, existingPrices, result); err != nil {
			return err
		}
	}

	return nil
}

// syncPriceAdvanced creates prices supporting flat, per_unit, and tiered pricing
func (c *Client) syncPriceAdvanced(productID, planID, interval string, localPrice config.Price, trialDays int, existingPrices []ProductPrice, result *SyncResult) error {
	priceType := localPrice.PriceType()

	// For flat prices, check if exact price already exists
	if priceType == "flat" {
		for _, p := range existingPrices {
			if p.Interval == interval && p.Amount == int64(localPrice.Amount) && p.Active {
				return nil // Price already exists
			}
		}

		// Archive conflicting prices
		for _, p := range existingPrices {
			if p.Interval == interval && p.Active && p.Amount != int64(localPrice.Amount) {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("plan '%s' %s: price differs (local=%d, stripe=%d), archiving old and creating new",
						planID, interval, localPrice.Amount, p.Amount))

				_, err := price.Update(p.ID, &stripe.PriceParams{
					Active: stripe.Bool(false),
				})
				if err != nil {
					return fmt.Errorf("failed to archive old price %s: %w", p.ID, err)
				}
				result.PricesArchived++
			}
		}
	}

	// Build price params
	params := &stripe.PriceParams{
		Product:  stripe.String(productID),
		Currency: stripe.String("usd"),
	}

	// Set price based on type
	switch priceType {
	case "flat":
		params.UnitAmount = stripe.Int64(int64(localPrice.Amount))

	case "per_unit":
		params.UnitAmount = stripe.Int64(int64(localPrice.PerUnit))
		params.BillingScheme = stripe.String("per_unit")
		// Transform quantity is handled at subscription level

	case "tiered":
		params.BillingScheme = stripe.String("tiered")
		if localPrice.Mode == "volume" {
			params.TiersMode = stripe.String("volume")
		} else {
			params.TiersMode = stripe.String("graduated")
		}

		params.Tiers = make([]*stripe.PriceTierParams, len(localPrice.Tiers))
		for i, tier := range localPrice.Tiers {
			tierParam := &stripe.PriceTierParams{}

			upTo := tier.GetTierUpTo()
			if upTo == -1 {
				tierParam.UpToInf = stripe.Bool(true)
			} else {
				tierParam.UpTo = stripe.Int64(upTo)
			}

			// Always set UnitAmount (even if 0 for free tiers)
			// Stripe requires at least one of UnitAmount or FlatAmount per tier
			if tier.Flat > 0 {
				tierParam.FlatAmount = stripe.Int64(int64(tier.Flat))
			} else {
				// Use UnitAmount (can be 0 for free tiers)
				tierParam.UnitAmount = stripe.Int64(int64(tier.Amount))
			}

			params.Tiers[i] = tierParam
		}
	}

	// Set recurring interval
	if interval != "" && interval != "one_time" {
		recurring := &stripe.PriceRecurringParams{}

		switch interval {
		case "monthly":
			recurring.Interval = stripe.String(string(stripe.PriceRecurringIntervalMonth))
		case "quarterly":
			recurring.Interval = stripe.String(string(stripe.PriceRecurringIntervalMonth))
			recurring.IntervalCount = stripe.Int64(3)
		case "yearly":
			recurring.Interval = stripe.String(string(stripe.PriceRecurringIntervalYear))
		default:
			return fmt.Errorf("unsupported interval: %s", interval)
		}

		// Add trial period if specified
		if trialDays > 0 {
			recurring.TrialPeriodDays = stripe.Int64(int64(trialDays))
		}

		// For per-unit pricing: use "licensed" (quantity set at subscription time)
		// "metered" requires a Meter object (for usage reporting)
		if priceType == "per_unit" {
			recurring.UsageType = stripe.String("licensed")
		}

		params.Recurring = recurring
	}

	_, err := price.New(params)
	if err != nil {
		return fmt.Errorf("failed to create %s price: %w", priceType, err)
	}
	result.PricesCreated++

	return nil
}

func (c *Client) syncAddon(addon config.Addon, existingProducts []Product, result *SyncResult) error {
	// Addons are products with one-time prices
	existingProduct := MatchProduct(existingProducts, addon.ID, addon.Name)

	var productID string

	if existingProduct != nil {
		productID = existingProduct.ID
		// Check if one-time price with correct amount exists
		for _, p := range existingProduct.Prices {
			if p.Interval == "" && p.Amount == int64(addon.Price.Amount) && p.Active {
				return nil // Addon already exists with correct price
			}
		}
	} else {
		// Create new product for addon
		params := &stripe.ProductParams{
			Name: stripe.String(addon.Name),
			Metadata: map[string]string{
				"addon_code": addon.ID,
				"type":       "addon",
			},
		}

		newProduct, err := product.New(params)
		if err != nil {
			return fmt.Errorf("failed to create addon product: %w", err)
		}
		productID = newProduct.ID
		result.AddonsCreated++
	}

	// Create one-time price for addon
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(int64(addon.Price.Amount)),
		Currency:   stripe.String("usd"),
	}

	_, err := price.New(priceParams)
	if err != nil {
		return fmt.Errorf("failed to create addon price: %w", err)
	}
	result.PricesCreated++

	return nil
}

func (c *Client) syncPromotion(promo config.Promotion, result *SyncResult) error {
	if !promo.IsActive() {
		return nil // Skip inactive promotions
	}

	// Create coupon in Stripe
	couponParams := &stripe.CouponParams{
		ID: stripe.String(promo.Code), // Use code as coupon ID
	}

	// Set discount type
	if promo.Discount.Percent > 0 {
		couponParams.PercentOff = stripe.Float64(float64(promo.Discount.Percent))
	} else if promo.Discount.Fixed > 0 {
		couponParams.AmountOff = stripe.Int64(int64(promo.Discount.Fixed))
		couponParams.Currency = stripe.String("usd")
	}

	// Set duration
	months := promo.GetDurationMonths()
	switch {
	case months == 0:
		couponParams.Duration = stripe.String("once")
	case months == -1:
		couponParams.Duration = stripe.String("forever")
	default:
		couponParams.Duration = stripe.String("repeating")
		couponParams.DurationInMonths = stripe.Int64(int64(months))
	}

	// Set max redemptions if specified
	if promo.MaxUses > 0 {
		couponParams.MaxRedemptions = stripe.Int64(int64(promo.MaxUses))
	}

	_, err := coupon.New(couponParams)
	if err != nil {
		// Check if coupon already exists
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceAlreadyExists {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("coupon '%s' already exists, skipping", promo.Code))
			return nil
		}
		return fmt.Errorf("failed to create coupon: %w", err)
	}
	result.CouponsCreated++

	// Create promotion code (the actual code customers enter)
	promoParams := &stripe.PromotionCodeParams{
		Coupon: stripe.String(promo.Code),
		Code:   stripe.String(promo.Code),
	}

	if promo.NewCustomersOnly {
		promoParams.Restrictions = &stripe.PromotionCodeRestrictionsParams{
			FirstTimeTransaction: stripe.Bool(true),
		}
	}

	_, err = promotioncode.New(promoParams)
	if err != nil {
		// Promotion code might already exist
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceAlreadyExists {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("promotion code '%s' already exists, skipping", promo.Code))
			return nil
		}
		return fmt.Errorf("failed to create promotion code: %w", err)
	}
	result.PromosCreated++

	return nil
}
