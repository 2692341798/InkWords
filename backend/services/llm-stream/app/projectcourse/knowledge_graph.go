package projectcourse

import (
	"context"
	"fmt"
	"sort"

	"inkwords-backend/shared/kernel/projectcourse"
)

type ModuleRecord struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	Role      FileRole `json:"role"`
	SymbolIDs []string `json:"symbol_ids"`
}

type KnowledgeGraph struct {
	CommitSHA string           `json:"commit_sha"`
	Files     []InventoryEntry `json:"files"`
	Symbols   []SymbolRecord   `json:"symbols"`
	Relations []RelationRecord `json:"relations"`
	Modules   []ModuleRecord   `json:"modules"`
}

func BuildKnowledgeGraph(ctx context.Context, snapshot projectcourse.SourceSnapshot, files []InventoryEntry, analyzers []SemanticAnalyzer) (KnowledgeGraph, error) {
	if err := snapshot.Validate(); err != nil {
		return KnowledgeGraph{}, err
	}
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: append([]InventoryEntry(nil), files...)}
	for _, analyzer := range analyzers {
		facts, err := analyzer.Analyze(ctx, snapshot, files)
		if err != nil {
			return KnowledgeGraph{}, fmt.Errorf("semantic analysis: %w", err)
		}
		graph.Symbols = append(graph.Symbols, facts.Symbols...)
		graph.Relations = append(graph.Relations, facts.Relations...)
	}
	for _, file := range files {
		var symbolIDs []string
		for _, symbol := range graph.Symbols {
			if symbol.Path == file.Path {
				symbolIDs = append(symbolIDs, fmt.Sprintf("%s:%s:%d", symbol.Path, symbol.Name, symbol.StartLine))
			}
		}
		graph.Modules = append(graph.Modules, ModuleRecord{ID: file.Path, Path: file.Path, Role: file.Role, SymbolIDs: symbolIDs})
	}
	sortFacts(&SemanticFacts{Symbols: graph.Symbols, Relations: graph.Relations})
	sort.Slice(graph.Modules, func(i, j int) bool { return graph.Modules[i].Path < graph.Modules[j].Path })
	return graph, nil
}
