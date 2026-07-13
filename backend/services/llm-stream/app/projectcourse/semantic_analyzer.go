package projectcourse

import (
	"context"

	"inkwords-backend/shared/kernel/projectcourse"
)

type SymbolKind string

const (
	SymbolPackage   SymbolKind = "package"
	SymbolImport    SymbolKind = "import"
	SymbolFunction  SymbolKind = "function"
	SymbolMethod    SymbolKind = "method"
	SymbolType      SymbolKind = "type"
	SymbolInterface SymbolKind = "interface"
	SymbolRoute     SymbolKind = "route"
	SymbolConsumer  SymbolKind = "consumer"
)

type SymbolRecord struct {
	Path      string     `json:"path"`
	Name      string     `json:"name"`
	Kind      SymbolKind `json:"kind"`
	StartLine int        `json:"start_line"`
	EndLine   int        `json:"end_line"`
}

type RelationKind string

const (
	RelationImports   RelationKind = "imports"
	RelationCalls     RelationKind = "calls"
	RelationRouteTo   RelationKind = "route_to_handler"
	RelationPublishes RelationKind = "publishes_event"
	RelationConsumes  RelationKind = "consumes_event"
)

type RelationRecord struct {
	FromPath   string       `json:"from_path"`
	FromSymbol string       `json:"from_symbol"`
	ToPath     string       `json:"to_path,omitempty"`
	ToSymbol   string       `json:"to_symbol"`
	Kind       RelationKind `json:"kind"`
}

type SemanticFacts struct {
	Symbols   []SymbolRecord   `json:"symbols"`
	Relations []RelationRecord `json:"relations"`
}

type SemanticAnalyzer interface {
	Supports(file InventoryEntry) bool
	Analyze(ctx context.Context, snapshot projectcourse.SourceSnapshot, files []InventoryEntry) (SemanticFacts, error)
}
