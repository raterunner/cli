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
		// Create new product
		params := &stripe.ProductParams{
			Name: stripe.String(plan.Name),
			Metadata: map[string]string{
				"plan_code": plan.ID,
			},
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
		if err := c.syncPrice(productID, plan.ID, interval, localPrice.Amount, existingPrices, result); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) syncPrice(productID, planID, interval string, amount int, existingPrices []ProductPrice, result *SyncResult) error {
	// Check if price with exact amount already exists
	for _, p := range existingPrices {
		if p.Interval == interval && p.Amount == int64(amount) && p.Active {
			return nil // Price already exists
		}
	}

	// Check for conflicting price
	for _, p := range existingPrices {
		if p.Interval == interval && p.Active && p.Amount != int64(amount) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("plan '%s' %s: price differs (local=%d, stripe=%d), archiving old and creating new",
					planID, interval, amount, p.Amount))

			// Archive old price
			_, err := price.Update(p.ID, &stripe.PriceParams{
				Active: stripe.Bool(false),
			})
			if err != nil {
				return fmt.Errorf("failed to archive old price %s: %w", p.ID, err)
			}
			result.PricesArchived++
		}
	}

	// Create new price
	params := &stripe.PriceParams{
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(int64(amount)),
		Currency:   stripe.String("usd"),
	}

	if interval != "" && interval != "one_time" {
		var stripeInterval stripe.PriceRecurringInterval
		switch interval {
		case "monthly":
			stripeInterval = stripe.PriceRecurringIntervalMonth
		case "yearly":
			stripeInterval = stripe.PriceRecurringIntervalYear
		default:
			return fmt.Errorf("unsupported interval: %s", interval)
		}
		params.Recurring = &stripe.PriceRecurringParams{
			Interval: stripe.String(string(stripeInterval)),
		}
	}

	_, err := price.New(params)
	if err != nil {
		return fmt.Errorf("failed to create price: %w", err)
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
