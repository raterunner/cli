package stripe

import (
	"raterunner/internal/config"
)

// Import fetches all products and prices from Stripe and converts to BillingConfig
func (c *Client) Import() (*config.BillingConfig, error) {
	products, err := c.FetchProductsWithPrices()
	if err != nil {
		return nil, err
	}

	cfg := &config.BillingConfig{
		Version:   1,
		Providers: []string{"stripe"},
		Plans:     make([]config.Plan, 0, len(products)),
	}

	for _, prod := range products {
		if !prod.Active {
			continue
		}

		plan := config.Plan{
			ID:     planIDFromProduct(prod),
			Name:   prod.Name,
			Prices: make(map[string]config.Price),
		}

		for _, p := range prod.Prices {
			if !p.Active {
				continue
			}
			if p.Interval == "" {
				continue // Skip one-time prices for now
			}
			plan.Prices[p.Interval] = config.Price{
				Amount: int(p.Amount),
			}
		}

		// Only add plans that have at least one price
		if len(plan.Prices) > 0 {
			cfg.Plans = append(cfg.Plans, plan)
		}
	}

	return cfg, nil
}

// planIDFromProduct extracts plan ID from product metadata or generates from name
func planIDFromProduct(prod Product) string {
	if prod.PlanCode != "" {
		return prod.PlanCode
	}
	// Fallback: normalize product name
	return normalizeName(prod.Name)
}
