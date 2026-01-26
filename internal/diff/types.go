package diff

// Status represents the sync status of a plan
type Status string

const (
	StatusOK      Status = "OK"
	StatusDiffers Status = "DIFFERS"
	StatusMissing Status = "MISSING"
)

// DiffResult contains the comparison results
type DiffResult struct {
	Environment string     `json:"environment"`
	ComparedAt  string     `json:"compared_at"`
	Plans       []PlanDiff `json:"plans"`
	Summary     Summary    `json:"summary"`
}

// PlanDiff represents the diff for a single plan
type PlanDiff struct {
	PlanID   string       `json:"plan_id"`
	PlanName string       `json:"plan_name"`
	Status   Status       `json:"status"`
	Details  string       `json:"details,omitempty"`
	Prices   []PriceDiff  `json:"prices,omitempty"`
}

// PriceDiff represents the diff for a single price
type PriceDiff struct {
	Interval    string `json:"interval"`
	LocalAmount int    `json:"local_amount"`
	StripeAmount int64  `json:"stripe_amount,omitempty"`
	Status      Status `json:"status"`
}

// Summary contains the summary statistics
type Summary struct {
	Total   int `json:"total"`
	Synced  int `json:"synced"`
	Missing int `json:"missing"`
	Differs int `json:"differs"`
}
