package config

// BillingConfig represents the full billing configuration
type BillingConfig struct {
	Version      int                    `yaml:"version" json:"version"`
	Providers    []string               `yaml:"providers" json:"providers"`
	Settings     *Settings              `yaml:"settings,omitempty" json:"settings,omitempty"`
	Entitlements map[string]Entitlement `yaml:"entitlements" json:"entitlements"`
	Plans        []Plan                 `yaml:"plans" json:"plans"`
	Addons       []Addon                `yaml:"addons" json:"addons"`
	Promotions   []Promotion            `yaml:"promotions" json:"promotions"`
}

// Settings contains global billing settings
type Settings struct {
	Currency  string `yaml:"currency,omitempty" json:"currency,omitempty"`
	TrialDays int    `yaml:"trial_days,omitempty" json:"trial_days,omitempty"`
	GraceDays int    `yaml:"grace_days,omitempty" json:"grace_days,omitempty"`
}

// Entitlement defines a feature or limit that can be granted
type Entitlement struct {
	Type        string `yaml:"type" json:"type"`
	Unit        string `yaml:"unit,omitempty" json:"unit,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Plan represents a pricing plan
type Plan struct {
	ID           string           `yaml:"id" json:"id"`
	Name         string           `yaml:"name" json:"name"`
	Description  string           `yaml:"description,omitempty" json:"description,omitempty"`
	Headline     string           `yaml:"headline,omitempty" json:"headline,omitempty"`
	Type         string           `yaml:"type,omitempty" json:"type,omitempty"`                   // personal, team, enterprise
	BillingModel string           `yaml:"billing_model,omitempty" json:"billing_model,omitempty"` // subscription (default), one_time
	Providers    []string         `yaml:"providers,omitempty" json:"providers,omitempty"`
	Public       *bool            `yaml:"public,omitempty" json:"public,omitempty"`
	Default      bool             `yaml:"default,omitempty" json:"default,omitempty"`
	TrialDays    int              `yaml:"trial_days,omitempty" json:"trial_days,omitempty"`
	Prices       map[string]Price `yaml:"prices" json:"prices"`
	Limits       map[string]any   `yaml:"limits,omitempty" json:"limits,omitempty"`
	Features     []string         `yaml:"features,omitempty" json:"features,omitempty"`
	UpgradesTo   []string         `yaml:"upgrades_to,omitempty" json:"upgrades_to,omitempty"`
	Metadata     map[string]any   `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// IsOneTime returns true if this plan uses one-time billing (e.g., lifetime deal)
func (p *Plan) IsOneTime() bool {
	return p.BillingModel == "one_time"
}

// EffectiveProviders returns the plan's providers, falling back to global providers
func (p *Plan) EffectiveProviders(globalProviders []string) []string {
	if len(p.Providers) > 0 {
		return p.Providers
	}
	return globalProviders
}

// HasProvider checks if this plan targets the given provider
func (p *Plan) HasProvider(provider string, globalProviders []string) bool {
	for _, pr := range p.EffectiveProviders(globalProviders) {
		if pr == provider {
			return true
		}
	}
	return false
}

// Price represents a price point for a plan (supports flat, per_unit, and tiered)
type Price struct {
	// Flat price
	Amount         int                `yaml:"amount,omitempty" json:"amount,omitempty"`
	CurrencyPrices map[string]int     `yaml:"currency_prices,omitempty" json:"currency_prices,omitempty"`

	// Per-unit price (usage-based)
	PerUnit  int    `yaml:"per_unit,omitempty" json:"per_unit,omitempty"`
	Unit     string `yaml:"unit,omitempty" json:"unit,omitempty"`
	Min      int    `yaml:"min,omitempty" json:"min,omitempty"`
	Max      int    `yaml:"max,omitempty" json:"max,omitempty"`
	Included int    `yaml:"included,omitempty" json:"included,omitempty"`

	// Tiered price
	Tiers []PriceTier `yaml:"tiers,omitempty" json:"tiers,omitempty"`
	Mode  string      `yaml:"mode,omitempty" json:"mode,omitempty"` // graduated, volume
}

// PriceTier represents a tier in tiered pricing
type PriceTier struct {
	UpTo   any `yaml:"up_to" json:"up_to"` // int or "unlimited"
	Amount int `yaml:"amount,omitempty" json:"amount,omitempty"`
	Flat   int `yaml:"flat,omitempty" json:"flat,omitempty"`
}

// PriceType returns the type of price: "flat", "per_unit", or "tiered"
func (p *Price) PriceType() string {
	if len(p.Tiers) > 0 {
		return "tiered"
	}
	if p.PerUnit > 0 {
		return "per_unit"
	}
	return "flat"
}

// GetTierUpTo returns the up_to value as int64, or -1 for unlimited
func (t *PriceTier) GetTierUpTo() int64 {
	switch v := t.UpTo.(type) {
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if v == "unlimited" {
			return -1
		}
	}
	return 0
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
