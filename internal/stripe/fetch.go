package stripe

import (
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// Product represents a Stripe product with its prices
type Product struct {
	ID           string
	Name         string
	PlanCode     string // from metadata
	BillingModel string // from metadata: "subscription" or "one_time"
	Active       bool
	Prices       []ProductPrice
}

// ProductPrice represents a Stripe price
type ProductPrice struct {
	ID       string
	Interval string // "month", "year", or "" for one-time
	Amount   int64
	Currency string
	Active   bool
}

// FetchProducts retrieves all active products from Stripe
func (c *Client) FetchProducts() ([]Product, error) {
	var products []Product

	params := &stripe.ProductListParams{}
	params.Filters.AddFilter("limit", "", "100")
	params.Filters.AddFilter("active", "", "true")

	iter := product.List(params)
	for iter.Next() {
		p := iter.Product()

		prod := Product{
			ID:     p.ID,
			Name:   p.Name,
			Active: p.Active,
		}

		// Check for plan_code in metadata
		if planCode, ok := p.Metadata["plan_code"]; ok {
			prod.PlanCode = planCode
		}

		// Check for billing_model in metadata
		if billingModel, ok := p.Metadata["billing_model"]; ok {
			prod.BillingModel = billingModel
		}

		products = append(products, prod)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	return products, nil
}

// FetchPricesForProduct retrieves all prices for a given product
func (c *Client) FetchPricesForProduct(productID string) ([]ProductPrice, error) {
	var prices []ProductPrice

	params := &stripe.PriceListParams{}
	params.Filters.AddFilter("product", "", productID)
	params.Filters.AddFilter("limit", "", "100")

	iter := price.List(params)
	for iter.Next() {
		p := iter.Price()

		pp := ProductPrice{
			ID:       p.ID,
			Amount:   p.UnitAmount,
			Currency: string(p.Currency),
			Active:   p.Active,
		}

		// Determine interval
		if p.Recurring != nil {
			switch p.Recurring.Interval {
			case stripe.PriceRecurringIntervalMonth:
				pp.Interval = "monthly"
			case stripe.PriceRecurringIntervalYear:
				pp.Interval = "yearly"
			default:
				pp.Interval = string(p.Recurring.Interval)
			}
		}

		prices = append(prices, pp)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list prices for product %s: %w", productID, err)
	}

	return prices, nil
}

// FetchProductsWithPrices retrieves all products with their prices
func (c *Client) FetchProductsWithPrices() ([]Product, error) {
	products, err := c.FetchProducts()
	if err != nil {
		return nil, err
	}

	for i := range products {
		prices, err := c.FetchPricesForProduct(products[i].ID)
		if err != nil {
			return nil, err
		}
		products[i].Prices = prices
	}

	return products, nil
}

// MatchProduct finds an active Stripe product that matches the given plan ID.
// It first checks for plan_code metadata match, then falls back to name matching.
// Only active products are considered - archived products are ignored.
func MatchProduct(products []Product, planID, planName string) *Product {
	// Primary: match by plan_code metadata (active only)
	for i := range products {
		if products[i].Active && products[i].PlanCode == planID {
			return &products[i]
		}
	}

	// Fallback: match by normalized name (active only)
	normalizedPlanID := normalizeName(planID)
	for i := range products {
		if products[i].Active && normalizeName(products[i].Name) == normalizedPlanID {
			return &products[i]
		}
	}

	return nil
}

// normalizeName converts a name to a normalized form for comparison
func normalizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace common separators with underscore
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	// Remove common suffixes
	name = strings.TrimSuffix(name, "_plan")
	return name
}
