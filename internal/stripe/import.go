package stripe

import (
	"time"

	"raterunner/internal/config"
)

// ImportResult contains both the billing config and provider ID mapping
type ImportResult struct {
	Billing  *config.BillingConfig
	Provider *config.ProviderConfig
}

// Import fetches all products and prices from Stripe and converts to BillingConfig and ProviderConfig
func (c *Client) Import() (*ImportResult, error) {
	products, err := c.FetchProductsWithPrices()
	if err != nil {
		return nil, err
	}

	billing := &config.BillingConfig{
		Version:   1,
		Providers: []string{"stripe"},
		Plans:     make([]config.Plan, 0, len(products)),
	}

	provider := &config.ProviderConfig{
		Provider:    "stripe",
		Environment: string(c.env),
		SyncedAt:    time.Now().UTC().Format(time.RFC3339),
		Plans:       make(map[string]config.PlanIDs),
	}

	for _, prod := range products {
		if !prod.Active {
			continue
		}

		planID := planIDFromProduct(prod)
		plan := config.Plan{
			ID:     planID,
			Name:   prod.Name,
			Prices: make(map[string]config.Price),
		}

		// Track provider IDs
		planIDs := config.PlanIDs{
			ProductID: prod.ID,
			Prices:    make(map[string]string),
		}

		hasRecurring := false
		hasOneTime := false

		for _, p := range prod.Prices {
			if !p.Active {
				continue
			}
			if p.Interval == "" {
				// One-time price
				plan.Prices["one_time"] = config.Price{
					Amount: int(p.Amount),
				}
				planIDs.Prices["one_time"] = p.ID
				hasOneTime = true
			} else {
				// Recurring price
				plan.Prices[p.Interval] = config.Price{
					Amount: int(p.Amount),
				}
				planIDs.Prices[p.Interval] = p.ID
				hasRecurring = true
			}
		}

		// Set billing_model: prefer metadata, fallback to price detection
		if prod.BillingModel != "" {
			plan.BillingModel = prod.BillingModel
		} else if hasOneTime && !hasRecurring {
			plan.BillingModel = "one_time"
		}

		// Only add plans that have at least one price
		if len(plan.Prices) > 0 {
			billing.Plans = append(billing.Plans, plan)
			provider.Plans[planID] = planIDs
		}
	}

	return &ImportResult{
		Billing:  billing,
		Provider: provider,
	}, nil
}

// planIDFromProduct extracts plan ID from product metadata or generates from name
func planIDFromProduct(prod Product) string {
	if prod.PlanCode != "" {
		return prod.PlanCode
	}
	// Fallback: normalize product name
	return normalizeName(prod.Name)
}
