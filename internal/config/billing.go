package config

// BillingConfig represents the full billing configuration
type BillingConfig struct {
	Version      int                    `yaml:"version" json:"version"`
	Providers    []string               `yaml:"providers" json:"providers"`
	Entitlements map[string]Entitlement `yaml:"entitlements" json:"entitlements"`
	Plans        []Plan                 `yaml:"plans" json:"plans"`
	Addons       []Addon                `yaml:"addons" json:"addons"`
	Promotions   []Promotion            `yaml:"promotions" json:"promotions"`
}

// Entitlement defines a feature or limit that can be granted
type Entitlement struct {
	Type string `yaml:"type" json:"type"`
	Unit string `yaml:"unit" json:"unit"`
}

// Plan represents a pricing plan
type Plan struct {
	ID     string           `yaml:"id" json:"id"`
	Name   string           `yaml:"name" json:"name"`
	Prices map[string]Price `yaml:"prices" json:"prices"`
	Limits map[string]any   `yaml:"limits" json:"limits"`
}

// Price represents a price point for a plan
type Price struct {
	Amount   int    `yaml:"amount" json:"amount"`
	Currency string `yaml:"currency,omitempty" json:"currency,omitempty"`
}

// Addon represents an add-on that can be purchased
type Addon struct {
	ID     string         `yaml:"id" json:"id"`
	Name   string         `yaml:"name" json:"name"`
	Price  Price          `yaml:"price" json:"price"`
	Grants map[string]any `yaml:"grants" json:"grants"`
}

// Promotion represents a promotional discount
type Promotion struct {
	Code             string            `yaml:"code" json:"code"`
	Description      string            `yaml:"description,omitempty" json:"description,omitempty"`
	Discount         PromotionDiscount `yaml:"discount" json:"discount"`
	Duration         any               `yaml:"duration,omitempty" json:"duration,omitempty"` // "once", "forever", or {months: N}
	AppliesTo        []string          `yaml:"applies_to,omitempty" json:"applies_to,omitempty"`
	NewCustomersOnly bool              `yaml:"new_customers_only,omitempty" json:"new_customers_only,omitempty"`
	MaxUses          int               `yaml:"max_uses,omitempty" json:"max_uses,omitempty"`
	Expires          string            `yaml:"expires,omitempty" json:"expires,omitempty"`
	Active           *bool             `yaml:"active,omitempty" json:"active,omitempty"`
}

// PromotionDiscount defines the discount amount
type PromotionDiscount struct {
	Percent int `yaml:"percent,omitempty" json:"percent,omitempty"`
	Fixed   int `yaml:"fixed,omitempty" json:"fixed,omitempty"`
}

// GetDurationMonths returns the number of months for repeating duration, 0 for "once", -1 for "forever"
func (p *Promotion) GetDurationMonths() int {
	if p.Duration == nil {
		return 0 // default: once
	}
	switch d := p.Duration.(type) {
	case string:
		if d == "forever" {
			return -1
		}
		return 0 // "once"
	case map[string]any:
		if months, ok := d["months"]; ok {
			if m, ok := months.(int); ok {
				return m
			}
			if m, ok := months.(float64); ok {
				return int(m)
			}
		}
	}
	return 0
}

// IsActive returns whether the promotion is active (defaults to true)
func (p *Promotion) IsActive() bool {
	if p.Active == nil {
		return true
	}
	return *p.Active
}
