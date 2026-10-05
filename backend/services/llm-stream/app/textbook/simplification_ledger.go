package textbook

import (
	"fmt"
	"strings"
)

// Simplification records a deliberate beginner-friendly approximation and its correction point.
type Simplification struct {
	EarlyExplanation string
	Boundary         string
	CorrectedIn      string
}

func ValidateSimplificationLedger(items []Simplification) error {
	for _, item := range items {
		if strings.TrimSpace(item.EarlyExplanation) == "" || strings.TrimSpace(item.Boundary) == "" || strings.TrimSpace(item.CorrectedIn) == "" {
			return fmt.Errorf("each simplification requires explanation, boundary and correction point")
		}
	}
	return nil
}
