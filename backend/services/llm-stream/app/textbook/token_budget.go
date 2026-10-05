package textbook

import sharedgeneration "inkwords-backend/shared/kernel/generation"

// TokenBudget is a provider-neutral preflight budget; estimates are explicitly estimates.
type TokenBudget = sharedgeneration.TokenBudget
type BudgetReport = sharedgeneration.BudgetReport

// CheckBudget estimates supplied fragments for diagnostics. Provider dispatch
// must use generation.CheckRequestBudget to include rendered message overhead.
func CheckBudget(budget TokenBudget, excerpts []string) (BudgetReport, error) {
	return sharedgeneration.CheckBudget(budget, excerpts)
}
