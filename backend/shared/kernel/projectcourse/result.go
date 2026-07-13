package projectcourse

type GateReport struct {
	Name    string     `json:"name"`
	Result  GateResult `json:"result"`
	Message string     `json:"message,omitempty"`
}

type CourseResult struct {
	CourseID         string       `json:"course_id"`
	BlueprintVersion int          `json:"blueprint_version"`
	CommitSHA        string       `json:"commit_sha"`
	Status           CourseStatus `json:"status"`
	Gates            []GateReport `json:"gates"`
}
