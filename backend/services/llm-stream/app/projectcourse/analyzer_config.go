package projectcourse

import (
	"context"
	"strings"

	"inkwords-backend/shared/kernel/projectcourse"
)

// ConfigAnalyzer 只提取配置文件的稳定文件级事实，不执行或解析其中的命令。
type ConfigAnalyzer struct{}

func (ConfigAnalyzer) Supports(file InventoryEntry) bool {
	lower := strings.ToLower(file.Path)
	return strings.HasSuffix(lower, "docker-compose.yml") || strings.HasSuffix(lower, "docker-compose.yaml") || strings.HasSuffix(lower, "dockerfile") || strings.HasSuffix(lower, "go.mod") || strings.HasSuffix(lower, "package.json")
}

func (a ConfigAnalyzer) Analyze(ctx context.Context, snapshot projectcourse.SourceSnapshot, files []InventoryEntry) (SemanticFacts, error) {
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
		if !a.Supports(file) {
			continue
		}
		facts.Symbols = append(facts.Symbols, SymbolRecord{Path: file.Path, Name: file.Path, Kind: SymbolType, StartLine: 1, EndLine: strings.Count(file.Content, "\n") + 1})
	}
	sortFacts(&facts)
	return facts, nil
}
