package projectcourse

import (
	"fmt"
	"sort"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

// BuildEvidencePack resolves blueprint evidence IDs against the immutable graph.
// It never invents a path or line range when the graph does not contain one.
func BuildEvidencePack(snapshot sharedkernel.SourceSnapshot, chapterID string, evidenceIDs []string, graph KnowledgeGraph, official []OfficialSource) (EvidencePack, error) {
	if err := snapshot.Validate(); err != nil {
		return EvidencePack{}, err
	}
	if strings.TrimSpace(chapterID) == "" {
		return EvidencePack{}, fmt.Errorf("chapter id is required")
	}
	byID := make(map[string]InventoryEntry, len(graph.Files))
	for _, file := range graph.Files {
		byID["evidence-"+stableID(file.Path+"\x00"+file.ContentHash)] = file
	}
	byPath := make(map[string][]SymbolRecord)
	for _, symbol := range graph.Symbols {
		byPath[symbol.Path] = append(byPath[symbol.Path], symbol)
	}
	for path := range byPath {
		sort.Slice(byPath[path], func(i, j int) bool {
			if byPath[path][i].StartLine != byPath[path][j].StartLine {
				return byPath[path][i].StartLine < byPath[path][j].StartLine
			}
			return byPath[path][i].Name < byPath[path][j].Name
		})
	}

	pack := EvidencePack{ChapterID: chapterID, OfficialSources: append([]OfficialSource(nil), official...)}
	seen := make(map[string]bool, len(evidenceIDs))
	for _, evidenceID := range evidenceIDs {
		file, ok := byID[evidenceID]
		if !ok {
			return EvidencePack{}, fmt.Errorf("evidence %q is not present in knowledge graph", evidenceID)
		}
		if file.Disposition == DispositionExcluded {
			return EvidencePack{}, fmt.Errorf("evidence %q points to excluded file %q", evidenceID, file.Path)
		}
		if seen[evidenceID] {
			continue
		}
		seen[evidenceID] = true
		start, end := 1, lineCount(file.Content)
		if end < start {
			end = 1
		}
		for _, symbol := range byPath[file.Path] {
			if symbol.StartLine > 0 {
				start, end = symbol.StartLine, symbol.EndLine
				break
			}
		}
		pack.SourceEvidence = append(pack.SourceEvidence, sharedkernel.EvidenceRef{
			EvidenceID: evidenceID, CommitSHA: snapshot.ResolvedCommitSHA, Path: file.Path,
			StartLine: start, EndLine: end, ContentHash: file.ContentHash,
		})
	}
	sort.Slice(pack.SourceEvidence, func(i, j int) bool { return pack.SourceEvidence[i].EvidenceID < pack.SourceEvidence[j].EvidenceID })
	if err := pack.Validate(); err != nil {
		return EvidencePack{}, err
	}
	return pack, nil
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}
