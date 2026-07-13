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
}
