package user

type ProfileData struct {
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	AvatarURL          string   `json:"avatar_url"`
	SubscriptionTier   int16    `json:"subscription_tier"`
	TokensUsed         int      `json:"tokens_used"`
	TokenLimit         int      `json:"token_limit"`
	ConnectedPlatforms []string `json:"connected_platforms"`
}

type UpdateProfileRequest struct {
	Username string `json:"username"`
}

type TechStackStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type StatsData struct {
	TokensUsed     int            `json:"tokens_used"`
	EstimatedCost  float64        `json:"estimated_cost"`
	TotalArticles  int64          `json:"total_articles"`
	TotalWords     int64          `json:"total_words"`
	TechStackStats []TechStackStat `json:"tech_stack_stats"`
}

type PromptSettingsResponse struct {
	Styles    []PromptStyleItem `json:"styles"`
	Defaults  map[string]string `json:"defaults"`
	Overrides map[string]string `json:"overrides"`
}

type PromptStyleItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type UpdatePromptSettingsRequest struct {
	Overrides map[string]string `json:"overrides"`
}
