package generation

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// TokenBudget is a provider-neutral input/output budget shared by preflight and workers.
type TokenBudget struct {
	MaxInput       int
	ReservedOutput int
}

// BudgetReport keeps estimates visibly separate from provider-reported usage.
type BudgetReport struct {
	EstimatedInput      int
	AllowedInput        int
	RequiresCompression bool
	Advice              string
}

// CheckBudget applies the conservative estimator to the supplied request parts.
func CheckBudget(budget TokenBudget, parts []string) (BudgetReport, error) {
	estimated := 0
	for _, part := range parts {
		estimated += ConservativeTokenEstimate(part)
	}
	return checkEstimatedBudget(budget, estimated)
}

func checkEstimatedBudget(budget TokenBudget, estimated int) (BudgetReport, error) {
	if budget.MaxInput < 1 || budget.ReservedOutput < 0 || budget.ReservedOutput >= budget.MaxInput {
		return BudgetReport{}, fmt.Errorf("invalid token budget")
	}
	allowed := budget.MaxInput - budget.ReservedOutput
	report := BudgetReport{EstimatedInput: estimated, AllowedInput: allowed}
	if estimated > allowed {
		report.RequiresCompression = true
		report.Advice = "生成输入超过预算：请缩小检索范围、精简输出结构，或明确提高本章预算；系统不会静默截断。"
	}
	return report, nil
}

// RequestBudgetEstimateMethod names the worker estimate, distinct from API
// preflight's frozen-task-plus-reserve estimate and from provider token usage.
const RequestBudgetEstimateMethod = "utf8_rendered_messages_schema_reserve_v1"

// RequestFramingReserveTokens allows for native schema wrappers and chat
// framing. It is an estimate allowance, not a tokenizer or billing guarantee.
const RequestFramingReserveTokens = 256

// CheckRequestBudget includes all rendered messages, JSON escaping, evidence
// identity/locators, and a second schema allowance for adapters that additionally
// send native structured-output configuration. The unused native allowance is
// intentionally retained for JSON-mode providers such as DeepSeek.
func CheckRequestBudget(budget TokenBudget, request Request) (BudgetReport, error) {
	messages, err := BuildInputMessages(request)
	if err != nil {
		return BudgetReport{}, err
	}
	encoded, err := json.Marshal(messages)
	if err != nil {
		return BudgetReport{}, fmt.Errorf("encode generation input estimate: %w", err)
	}
	estimated := ConservativeTokenEstimate(string(encoded)) + ConservativeTokenEstimate(string(request.ResponseSchema)) + RequestFramingReserveTokens
	return checkEstimatedBudget(budget, estimated)
}

// ConservativeTokenEstimate counts non-ASCII runes individually and groups ASCII by four.
// It is deliberately an estimate, never provider billing evidence.
func ConservativeTokenEstimate(value string) int {
	ascii := 0
	nonASCII := 0
	for _, runeValue := range value {
		if runeValue <= utf8.RuneSelf {
			ascii++
			continue
		}
		nonASCII++
	}
	return (ascii+3)/4 + nonASCII
}
