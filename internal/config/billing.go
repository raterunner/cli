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
	Code     string           `yaml:"code" json:"code"`
	Discount PromotionDiscount `yaml:"discount" json:"discount"`
	Duration PromotionDuration `yaml:"duration" json:"duration"`
}

// PromotionDiscount defines the discount amount
type PromotionDiscount struct {
	Percent int `yaml:"percent,omitempty" json:"percent,omitempty"`
	Amount  int `yaml:"amount,omitempty" json:"amount,omitempty"`
}

// PromotionDuration defines how long the discount applies
type PromotionDuration struct {
	Months int `yaml:"months,omitempty" json:"months,omitempty"`
}
