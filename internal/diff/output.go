package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// OutputTable writes the diff result as a formatted table
func OutputTable(w io.Writer, result *DiffResult) {
	fmt.Fprintf(w, "Environment: %s\n", result.Environment)
	fmt.Fprintf(w, "Compared at: %s\n", result.ComparedAt)
	fmt.Fprintln(w)

	// Header
	fmt.Fprintf(w, "%-20s %10s  %s\n", "PLAN", "STATUS", "DETAILS")
	fmt.Fprintln(w, strings.Repeat("-", 60))

	// Plans
	for _, plan := range result.Plans {
		status := formatStatus(plan.Status)
		fmt.Fprintf(w, "%-20s %10s", plan.PlanID, status)
		if plan.Details != "" {
			fmt.Fprintf(w, "  %s", plan.Details)
		}
		fmt.Fprintln(w)
	}

	// Summary
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Summary: %d total, %d synced, %d missing, %d differs\n",
		result.Summary.Total,
		result.Summary.Synced,
		result.Summary.Missing,
		result.Summary.Differs,
	)
}

// OutputJSON writes the diff result as JSON
func OutputJSON(w io.Writer, result *DiffResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// formatStatus formats the status with brackets
func formatStatus(s Status) string {
	return fmt.Sprintf("[%s]", s)
}
