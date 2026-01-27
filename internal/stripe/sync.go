package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"

	"raterunner/internal/config"
)

// SyncResult contains the results of the sync operation
type SyncResult struct {
	ProductsCreated int
	PricesCreated   int
	PricesArchived  int
	Warnings        []string
}

// Sync creates or updates all plans from a billing config in Stripe
func (c *Client) Sync(cfg *config.BillingConfig) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch existing products once
	existingProducts, err := c.FetchProductsWithPrices()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing products: %w", err)
	}

	for _, plan := range cfg.Plans {
		if err := c.syncPlan(plan, existingProducts, result); err != nil {
			return result, fmt.Errorf("failed to sync plan '%s': %w", plan.ID, err)
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
