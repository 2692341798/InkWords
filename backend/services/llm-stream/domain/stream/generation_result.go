package stream

import "encoding/json"

// TaskResultEnvelope 表示 generation 任务成功后写入 job_tasks.result_json 的统一外层结构。
type TaskResultEnvelope struct {
	ResultVersion   int             `json:"result_version"`
	TaskType        string          `json:"task_type"`
	TaskSubtype     string          `json:"task_subtype"`
	PersistenceMode string          `json:"persistence_mode"`
	FinalStatus     string          `json:"final_status"`
	Usage           TaskResultUsage `json:"usage"`
	Payload         map[string]any  `json:"payload"`
}

// TaskResultUsage 表示任务结果中附带的用量摘要。
type TaskResultUsage struct {
	EstimatedTokens int `json:"estimated_tokens"`
}

// GenerateSingleTaskResultInput 表示单篇生成成功后交给 core-api 的业务事实快照。
type GenerateSingleTaskResultInput struct {
	BlogID          string
	Title           string
	Content         string
	SourceType      string
	WordCount       int
	TechStacks      []string
	EstimatedTokens int
}

// ContinueTaskResultInput 表示续写任务成功后交给 core-api 的业务事实快照。
type ContinueTaskResultInput struct {
	BlogID          string
	AppendedContent string
	FinalContent    string
	EstimatedTokens int
}

// SeriesParentTaskResult 表示系列任务中父博客最终需要持久化的业务事实。
type SeriesParentTaskResult struct {
	BlogID  string `json:"blog_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// SeriesChapterTaskResult 表示系列任务中单个章节最终需要持久化的业务事实。
type SeriesChapterTaskResult struct {
	BlogID       string   `json:"blog_id"`
	ChapterSort  int      `json:"chapter_sort"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	WordCount    int      `json:"word_count"`
	TechStacks   []string `json:"tech_stacks"`
	Status       string   `json:"status"`
	ErrorMessage string   `json:"error_message"`
}

// GenerateSeriesTaskResultInput 表示系列生成完成后交给 core-api 的父子博客快照。
type GenerateSeriesTaskResultInput struct {
	ParentBlogID    string
	ParentTitle     string
	ParentContent   string
	EstimatedTokens int
	Chapters        []SeriesChapterTaskResult
}

// BuildGenerateSingleTaskResult 构造 generate_single 的 task_only 结果契约。
func BuildGenerateSingleTaskResult(input GenerateSingleTaskResultInput) ([]byte, error) {
	envelope := TaskResultEnvelope{
		ResultVersion:   1,
		TaskType:        "generation",
		TaskSubtype:     "generate_single",
		PersistenceMode: "task_only",
		FinalStatus:     "succeeded",
		Usage:           TaskResultUsage{EstimatedTokens: input.EstimatedTokens},
		Payload: map[string]any{
			"blog_id":     input.BlogID,
			"title":       input.Title,
			"content":     input.Content,
			"source_type": input.SourceType,
			"word_count":  input.WordCount,
			"tech_stacks": input.TechStacks,
		},
	}

	return json.Marshal(envelope)
}

// BuildContinueTaskResult 构造 continue 的 task_only 结果契约。
func BuildContinueTaskResult(input ContinueTaskResultInput) ([]byte, error) {
	envelope := TaskResultEnvelope{
		ResultVersion:   1,
		TaskType:        "generation",
		TaskSubtype:     "continue",
		PersistenceMode: "task_only",
		FinalStatus:     "succeeded",
		Usage:           TaskResultUsage{EstimatedTokens: input.EstimatedTokens},
		Payload: map[string]any{
			"blog_id":          input.BlogID,
			"appended_content": input.AppendedContent,
			"final_content":    input.FinalContent,
		},
	}

	return json.Marshal(envelope)
}

// BuildGenerateSeriesTaskResult 构造 generate_series 的 task_only 结果契约。
func BuildGenerateSeriesTaskResult(input GenerateSeriesTaskResultInput) ([]byte, error) {
	envelope := TaskResultEnvelope{
		ResultVersion:   1,
		TaskType:        "generation",
		TaskSubtype:     "generate_series",
		PersistenceMode: "task_only",
		FinalStatus:     "succeeded",
		Usage:           TaskResultUsage{EstimatedTokens: input.EstimatedTokens},
		Payload: map[string]any{
			"parent_blog": SeriesParentTaskResult{
				BlogID:  input.ParentBlogID,
				Title:   input.ParentTitle,
				Content: input.ParentContent,
			},
			"chapters": input.Chapters,
		},
	}

	return json.Marshal(envelope)
}
