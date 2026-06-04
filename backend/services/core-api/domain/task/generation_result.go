package task

// GenerationResult describes the structured generation task result persisted in
// job_tasks.result_json and consumed by core-api.
type GenerationResult struct {
	ResultVersion   int                   `json:"result_version"`
	TaskType        string                `json:"task_type"`
	TaskSubtype     string                `json:"task_subtype"`
	PersistenceMode string                `json:"persistence_mode"`
	FinalStatus     string                `json:"final_status"`
	Usage           GenerationResultUsage `json:"usage"`
	Payload         map[string]any        `json:"payload"`
}

// GenerationResultUsage carries token accounting facts that belong to core-api.
type GenerationResultUsage struct {
	EstimatedTokens int `json:"estimated_tokens"`
}
