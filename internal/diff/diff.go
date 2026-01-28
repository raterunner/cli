package diff

import (
	"fmt"
	"strings"
	"time"

	"raterunner/internal/config"
	"raterunner/internal/stripe"
)

// Compare compares the local billing config with Stripe products
func Compare(cfg *config.BillingConfig, products []stripe.Product, env string) *DiffResult {
	result := &DiffResult{
		Environment: env,
		ComparedAt:  time.Now().Format("2006-01-02 15:04:05"),
		Plans:       make([]PlanDiff, 0, len(cfg.Plans)),
	}

	for _, plan := range cfg.Plans {
		// Skip plans not targeting Stripe
		if !plan.HasProvider("stripe", cfg.Providers) {
			continue
		}
		planDiff := comparePlan(plan, products)
		result.Plans = append(result.Plans, planDiff)

		switch planDiff.Status {
		case StatusOK:
			result.Summary.Synced++
		case StatusMissing:
			result.Summary.Missing++
		case StatusDiffers:
			result.Summary.Differs++
		}
		result.Summary.Total++
	}

	return result
}

// comparePlan compares a single plan with Stripe products
func comparePlan(plan config.Plan, products []stripe.Product) PlanDiff {
	diff := PlanDiff{
		PlanID:   plan.ID,
		PlanName: plan.Name,
	}

	// Find matching Stripe product
	product := stripe.MatchProduct(products, plan.ID, plan.Name)
	if product == nil {
		diff.Status = StatusMissing
		diff.Details = "Not in Stripe"
		return diff
	}

	// Compare prices
	var priceDiffs []PriceDiff
	var differDetails []string

	for interval, localPrice := range plan.Prices {
		priceDiff := PriceDiff{
			Interval:    interval,
			LocalAmount: localPrice.Amount,
		}

		// Find matching Stripe price
		stripePrice := findPrice(product.Prices, interval)
		if stripePrice == nil {
			priceDiff.Status = StatusMissing
			differDetails = append(differDetails, fmt.Sprintf("%s: missing in Stripe", interval))
		} else {
			priceDiff.StripeAmount = stripePrice.Amount
			if int64(localPrice.Amount) == stripePrice.Amount {
				priceDiff.Status = StatusOK
			} else {
				priceDiff.Status = StatusDiffers
				differDetails = append(differDetails, fmt.Sprintf("%s: local=%d stripe=%d", interval, localPrice.Amount, stripePrice.Amount))
			}
		}

		priceDiffs = append(priceDiffs, priceDiff)
	}

	diff.Prices = priceDiffs

	if len(differDetails) > 0 {
		diff.Status = StatusDiffers
		diff.Details = strings.Join(differDetails, ", ")
	} else {
		diff.Status = StatusOK
	}

	return diff
}

// findPrice finds a price by interval
func findPrice(prices []stripe.ProductPrice, interval string) *stripe.ProductPrice {
	for i := range prices {
		if prices[i].Interval == interval && prices[i].Active {
			return &prices[i]
		}
	}
	return nil
}

// HasDifferences returns true if there are any differences
func (r *DiffResult) HasDifferences() bool {
	return r.Summary.Missing > 0 || r.Summary.Differs > 0
}
