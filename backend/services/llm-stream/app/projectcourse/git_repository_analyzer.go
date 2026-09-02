package projectcourse

import (
	"context"
	"fmt"
	"strings"
	"time"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
	"inkwords-backend/shared/platform/parser"
)

type GitRepositoryAnalyzer struct{ Fetcher *parser.GitFetcher }

func (a GitRepositoryAnalyzer) Analyze(ctx context.Context, repositoryURL, requestedRef string) (RepositoryAnalysis, error) {
	if a.Fetcher == nil {
		return RepositoryAnalysis{}, fmt.Errorf("git fetcher is not configured")
	}
	select {
	case <-ctx.Done():
		return RepositoryAnalysis{}, ctx.Err()
	default:
	}
	result, err := a.Fetcher.FetchSnapshot(repositoryURL, "/", requestedRef, nil)
	if err != nil {
		return RepositoryAnalysis{}, err
	}
	snapshot := sharedkernel.SourceSnapshot{RepositoryURL: repositoryURL, RequestedRef: requestedRef, ResolvedCommitSHA: result.ResolvedCommitSHA, CapturedAt: time.Now().UTC()}
	if err := snapshot.Validate(); err != nil {
		return RepositoryAnalysis{}, err
	}
	inputs := inventoryInputsFromChunks(result.Chunks)
	files, err := BuildInventory(inputs, InventoryOptions{MaxContentBytes: 2_000_000})
	if err != nil {
		return RepositoryAnalysis{}, err
	}
	graph, err := BuildKnowledgeGraph(ctx, snapshot, files, []SemanticAnalyzer{GoAnalyzer{}, TypeScriptAnalyzer{}, ConfigAnalyzer{}})
	if err != nil {
		return RepositoryAnalysis{}, err
	}
	return RepositoryAnalysis{Snapshot: snapshot, Graph: graph}, nil
}

func inventoryInputsFromChunks(chunks []parser.FileChunk) []InventoryInput {
	var inputs []InventoryInput
	for _, chunk := range chunks {
		var currentPath string
		var content strings.Builder
		for _, line := range strings.SplitAfter(chunk.Content, "\n") {
			if filePath, ok := parseFileHeader(line); ok {
				if currentPath != "" {
					inputs = append(inputs, InventoryInput{Path: currentPath, Content: []byte(content.String())})
				}
				currentPath = filePath
				content.Reset()
				continue
			}
			if currentPath != "" {
				content.WriteString(line)
			}
		}
		if currentPath != "" {
			inputs = append(inputs, InventoryInput{Path: currentPath, Content: []byte(content.String())})
		}
	}
	return inputs
}

func parseFileHeader(line string) (string, bool) {
	const prefix = "--- File: "
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if !strings.HasSuffix(line, " ---") {
		return "", false
	}
	filePath := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, prefix), " ---"))
	return filePath, filePath != ""
}
