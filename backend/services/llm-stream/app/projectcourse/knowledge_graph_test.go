package projectcourse

import (
	"context"
	"testing"
	"time"

	"inkwords-backend/shared/kernel/projectcourse"
)

func TestGoAnalyzerExtractsSymbolsAndRelationsDeterministically(t *testing.T) {
	snapshot := projectcourse.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	files := []InventoryEntry{{Path: "backend/services/core-api/transport/routes.go", Role: RoleTransport, Disposition: DispositionCovered, Content: `package transport
import "fmt"
type Handler interface { Serve() }
func RegisterRoutes() { fmt.Println("route") }
`}}
	facts, err := (GoAnalyzer{}).Analyze(context.Background(), snapshot, files)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSymbol(facts.Symbols, "RegisterRoutes", SymbolRoute) {
		t.Fatal("route symbol was not extracted")
	}
	if !hasSymbol(facts.Symbols, "Handler", SymbolInterface) {
		t.Fatal("interface symbol was not extracted")
	}
	if !hasSymbol(facts.Symbols, "fmt", SymbolImport) {
		t.Fatal("import symbol was not extracted")
	}
	if len(facts.Relations) == 0 {
		t.Fatal("call relation was not extracted")
	}
}

func TestKnowledgeGraphKeepsSnapshotAndFileCoverage(t *testing.T) {
	snapshot := projectcourse.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	files := []InventoryEntry{{Path: "main.go", Role: RoleApplication, Disposition: DispositionCovered, Content: "package main"}}
	graph, err := BuildKnowledgeGraph(context.Background(), snapshot, files, []SemanticAnalyzer{GoAnalyzer{}})
	if err != nil {
		t.Fatal(err)
	}
	if graph.CommitSHA != snapshot.ResolvedCommitSHA || len(graph.Modules) != 1 {
		t.Fatalf("unexpected graph: %#v", graph)
	}
}

func TestLowPrecisionTypeScriptAndConfigAdaptersStayFileScoped(t *testing.T) {
	snapshot := projectcourse.SourceSnapshot{RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", CapturedAt: time.Unix(1, 0)}
	files := []InventoryEntry{
		{Path: "frontend/src/hooks/useProjectAnalyzer.ts", Content: "import { useMemo } from 'react'\nexport function useProjectAnalyzer() {}", Disposition: DispositionCovered},
		{Path: "docker-compose.yml", Content: "services:\n  api:", Disposition: DispositionIndexed},
	}
	tsFacts, err := (TypeScriptAnalyzer{}).Analyze(context.Background(), snapshot, files)
	if err != nil || len(tsFacts.Symbols) != 2 {
		t.Fatalf("unexpected TypeScript facts: %#v %v", tsFacts, err)
	}
	configFacts, err := (ConfigAnalyzer{}).Analyze(context.Background(), snapshot, files)
	if err != nil || len(configFacts.Symbols) != 1 || configFacts.Symbols[0].Path != "docker-compose.yml" {
		t.Fatalf("unexpected config facts: %#v %v", configFacts, err)
	}
}

func hasSymbol(symbols []SymbolRecord, name string, kind SymbolKind) bool {
	for _, symbol := range symbols {
		if symbol.Name == name && symbol.Kind == kind {
			return true
		}
	}
	return false
}
