package projectcourse

import "github.com/google/uuid"

type CreateInput struct {
	UserID            uuid.UUID
	RepositoryURL     string
	RequestedRef      string
	ResolvedCommitSHA string
	AudienceLevel     string
}

type BlueprintUpdate struct {
	ExpectedVersion int
	BlueprintJSON   []byte
	CoverageJSON    []byte
	ChapterUpdates  []ChapterUpdate
}

type ChapterUpdate struct {
	ChapterID string `json:"chapter_id"`
	Title     string `json:"title"`
	Sort      int    `json:"sort"`
	Enabled   bool   `json:"enabled"`
}
