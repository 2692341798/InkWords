package projectcourse

import (
	"context"
	"strings"

	"inkwords-backend/shared/kernel/projectcourse"
)

// TypeScriptAnalyzer 是首期的低精度文件级适配器。
// 在引入成熟 TS/TSX AST 解析器前，只记录明显的 import/export 行，不生成调用关系。
type TypeScriptAnalyzer struct{}

func (TypeScriptAnalyzer) Supports(file InventoryEntry) bool {
	lower := strings.ToLower(file.Path)
	return (strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".jsx")) && file.Disposition != DispositionExcluded
}

func (a TypeScriptAnalyzer) Analyze(ctx context.Context, snapshot projectcourse.SourceSnapshot, files []InventoryEntry) (SemanticFacts, error) {
	if err := snapshot.Validate(); err != nil {
		return SemanticFacts{}, err
	}
	var facts SemanticFacts
	for _, file := range files {
		select {
		case <-ctx.Done():
			return SemanticFacts{}, ctx.Err()
		default:
		}
		if !a.Supports(file) || file.Content == "" {
			continue
		}
		for lineNumber, line := range strings.Split(file.Content, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export ") {
				kind := SymbolImport
				if strings.HasPrefix(trimmed, "export ") {
					kind = SymbolFunction
				}
				facts.Symbols = append(facts.Symbols, SymbolRecord{Path: file.Path, Name: trimmed, Kind: kind, StartLine: lineNumber + 1, EndLine: lineNumber + 1})
			}
		}
	}
	sortFacts(&facts)
	return facts, nil
}
