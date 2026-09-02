// Command test_project_course_e2e performs the read-only, pre-generation part
// of Project Course acceptance against a pinned repository ref. It never runs
// target-repository commands; it only fetches source text and builds inventory
// plus the knowledge graph.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	projectcourse "inkwords-backend/services/llm-stream/app/projectcourse"
	"inkwords-backend/shared/platform/parser"
)

type acceptanceSummary struct {
	RepositoryURL     string    `json:"repository_url"`
	RequestedRef      string    `json:"requested_ref"`
	ResolvedCommitSHA string    `json:"resolved_commit_sha"`
	CapturedAt        time.Time `json:"captured_at"`
	FileCount         int       `json:"file_count"`
	SymbolCount       int       `json:"symbol_count"`
	RelationCount     int       `json:"relation_count"`
	ExecutedTarget    bool      `json:"executed_target_repository"`
}

func main() {
	repository := flag.String("repository", "https://github.com/2692341798/InkWords", "repository URL")
	ref := flag.String("ref", "main", "immutable ref to resolve")
	flag.Parse()

	analyzer := projectcourse.GitRepositoryAnalyzer{Fetcher: parser.NewGitFetcher()}
	analysis, err := analyzer.Analyze(context.Background(), *repository, *ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project course read-only acceptance failed: %v\n", err)
		os.Exit(1)
	}
	summary := acceptanceSummary{
		RepositoryURL: *repository, RequestedRef: *ref,
		ResolvedCommitSHA: analysis.Snapshot.ResolvedCommitSHA,
		CapturedAt:        analysis.Snapshot.CapturedAt,
		FileCount:         len(analysis.Graph.Files), SymbolCount: len(analysis.Graph.Symbols),
		RelationCount: len(analysis.Graph.Relations), ExecutedTarget: false,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		fmt.Fprintf(os.Stderr, "write acceptance summary: %v\n", err)
		os.Exit(1)
	}
}
