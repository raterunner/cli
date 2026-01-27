package stripe

import (
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/coupon"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// TruncateResult contains the results of the truncate operation
type TruncateResult struct {
	ProductsArchived int
	PricesArchived   int
	CouponsDeleted   int
}

// Truncate archives all products, prices, and deletes coupons in the Stripe account.
// This only works in sandbox environment.
func (c *Client) Truncate() (*TruncateResult, error) {
	if c.env != Sandbox {
		return nil, fmt.Errorf("truncate is only allowed in sandbox environment")
	}

	result := &TruncateResult{}

	// First, archive all prices (must be done before products)
	priceParams := &stripe.PriceListParams{}
	priceParams.Filters.AddFilter("limit", "", "100")
	priceParams.Filters.AddFilter("active", "", "true")

	priceIter := price.List(priceParams)
	for priceIter.Next() {
		p := priceIter.Price()
		_, err := price.Update(p.ID, &stripe.PriceParams{
			Active: stripe.Bool(false),
		})
		if err != nil {
			return result, fmt.Errorf("failed to archive price %s: %w", p.ID, err)
		}
		result.PricesArchived++
	}
	if err := priceIter.Err(); err != nil {
		return result, fmt.Errorf("failed to list prices: %w", err)
	}

	// Then archive all products
	prodParams := &stripe.ProductListParams{}
	prodParams.Filters.AddFilter("limit", "", "100")
	prodParams.Filters.AddFilter("active", "", "true")

	prodIter := product.List(prodParams)
	for prodIter.Next() {
		p := prodIter.Product()
		_, err := product.Update(p.ID, &stripe.ProductParams{
			Active: stripe.Bool(false),
		})
		if err != nil {
			return result, fmt.Errorf("failed to archive product %s: %w", p.ID, err)
		}
		result.ProductsArchived++
	}
	if err := prodIter.Err(); err != nil {
		return result, fmt.Errorf("failed to list products: %w", err)
	}

	// Delete all coupons
	couponParams := &stripe.CouponListParams{}
	couponParams.Filters.AddFilter("limit", "", "100")

	couponIter := coupon.List(couponParams)
	for couponIter.Next() {
		c := couponIter.Coupon()
		_, err := coupon.Del(c.ID, nil)
		if err != nil {
			return result, fmt.Errorf("failed to delete coupon %s: %w", c.ID, err)
		}
		result.CouponsDeleted++
	}
	if err := couponIter.Err(); err != nil {
		return result, fmt.Errorf("failed to list coupons: %w", err)
	}

	return result, nil
}
