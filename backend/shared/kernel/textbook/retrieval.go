package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// RetrievalCandidate is one locally explainable source chunk. It contains no
// model-derived signal so equal source snapshots and a query always rank alike.
type RetrievalCandidate struct {
	Snapshot SourceSnapshot `json:"snapshot"`
	Document SourceDocument `json:"document"`
	Chunk    SourceChunk    `json:"chunk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
}

// RetrievalPlan is an immutable audit record for one bounded source lookup.
// Candidates are retained so an author can see why a selected excerpt won.
type RetrievalPlan struct {
	Query      string               `json:"query"`
	Candidates []RetrievalCandidate `json:"candidates"`
	Selected   []RetrievalCandidate `json:"selected"`
	InputHash  string               `json:"input_hash"`
}

// RetrieveEvidence ranks chunks deterministically. Primary-source facts get a
// small priority boost; this is a tie-breaker, not permission to hide a conflict.
func RetrieveEvidence(query string, candidates []RetrievalCandidate, limit int) (RetrievalPlan, error) {
	terms := normalizedRetrievalTerms(query)
	if len(terms) == 0 || limit < 1 {
		return RetrievalPlan{}, fmt.Errorf("retrieval needs a query and positive limit")
	}
	for index := range candidates {
		candidate := &candidates[index]
		if err := validateRetrievalCandidate(*candidate); err != nil {
			return RetrievalPlan{}, err
		}
		candidate.Score, candidate.Reasons = scoreRetrievalCandidate(terms, *candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Chunk.ID < candidates[j].Chunk.ID
	})
	selected := make([]RetrievalCandidate, 0, limit)
	for _, candidate := range candidates {
		if candidate.Score > 0 && len(selected) < limit {
			selected = append(selected, candidate)
		}
	}
	if len(selected) == 0 {
		return RetrievalPlan{}, fmt.Errorf("no evidence matches the retrieval query")
	}
	parts := []string{strings.Join(terms, " ")}
	for _, candidate := range candidates {
		parts = append(parts, candidate.Snapshot.ID+":"+candidate.Document.ID+":"+candidate.Chunk.ID+":"+candidate.Chunk.TextHash+":"+fmt.Sprint(candidate.Score)+":"+strings.Join(candidate.Reasons, ","))
	}
	return RetrievalPlan{Query: query, Candidates: candidates, Selected: selected, InputHash: retrievalDigest(strings.Join(parts, "\n"))}, nil
}

// BuildGenerationEvidencePack converts a reviewed retrieval plan into the only
// source window generation is allowed to receive. It never includes all chunks.
func BuildGenerationEvidencePack(plan RetrievalPlan) (GenerationEvidencePack, error) {
	if len(plan.Selected) == 0 || strings.TrimSpace(plan.InputHash) == "" {
		return GenerationEvidencePack{}, fmt.Errorf("retrieval plan has no selected evidence")
	}
	pack := GenerationEvidencePack{Excerpts: map[string]string{}}
	seenSnapshots := map[string]bool{}
	for _, candidate := range plan.Selected {
		if err := validateRetrievalCandidate(candidate); err != nil {
			return GenerationEvidencePack{}, err
		}
		if !seenSnapshots[candidate.Snapshot.ID] {
			seenSnapshots[candidate.Snapshot.ID] = true
			switch candidate.Snapshot.Role {
			case SourceRolePrimary:
				if pack.PrimarySnapshot.ID == "" {
					pack.PrimarySnapshot = candidate.Snapshot
				} else {
					pack.PrimarySnapshots = append(pack.PrimarySnapshots, candidate.Snapshot)
				}
			case SourceRoleOfficial:
				pack.OfficialSources = append(pack.OfficialSources, candidate.Snapshot)
			default:
				return GenerationEvidencePack{}, fmt.Errorf("retrieval selected unapproved source role")
			}
		}
		evidenceID := "evidence-" + candidate.Chunk.ID
		pack.Evidence = append(pack.Evidence, EvidenceRef{ID: evidenceID, SnapshotID: candidate.Snapshot.ID, DocumentID: candidate.Document.ID, ChunkID: candidate.Chunk.ID, Locator: candidate.Chunk.Locator, ContentHash: candidate.Chunk.TextHash, Confidence: EvidenceConfidenceDocumented, SourceRole: candidate.Snapshot.Role})
		pack.Excerpts[evidenceID] = candidate.Chunk.SearchText
	}
	if err := pack.Validate(); err != nil {
		return GenerationEvidencePack{}, fmt.Errorf("build evidence pack: %w", err)
	}
	return pack, nil
}

func validateRetrievalCandidate(candidate RetrievalCandidate) error {
	if err := candidate.Snapshot.Validate(); err != nil {
		return err
	}
	if err := candidate.Document.Validate(); err != nil {
		return err
	}
	if err := candidate.Chunk.Validate(); err != nil {
		return err
	}
	if candidate.Document.SnapshotID != candidate.Snapshot.ID || candidate.Chunk.DocumentID != candidate.Document.ID {
		return fmt.Errorf("retrieval candidate provenance mismatch")
	}
	return nil
}

func scoreRetrievalCandidate(terms []string, candidate RetrievalCandidate) (int, []string) {
	score, reasons := 0, []string{}
	haystacks := []struct {
		value, reason string
		weight        int
	}{{candidate.Chunk.SearchText, "正文关键词", 4}, {candidate.Chunk.Locator.Symbol, "源码符号", 6}, {strings.Join(candidate.Chunk.HeadingPath, " "), "标题路径", 3}, {candidate.Document.ArtifactPath + " " + candidate.Document.CanonicalLocator, "文件路径", 2}}
	for _, term := range terms {
		for _, field := range haystacks {
			if strings.Contains(strings.ToLower(field.value), term) {
				score += field.weight
				reasons = append(reasons, field.reason+":"+term)
			}
		}
	}
	if score > 0 && candidate.Snapshot.Role == SourceRolePrimary {
		score += 2
		reasons = append(reasons, "主资料优先")
	}
	return score, reasons
}

func normalizedRetrievalTerms(query string) []string {
	seen := map[string]bool{}
	var terms []string
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if len(term) > 1 && !seen[term] {
			seen[term] = true
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms
}

// RetrievalQueryTerms shares literal query normalization with database candidate
// filtering, keeping the bounded fetch consistent with local ranking.
func RetrievalQueryTerms(query string) []string { return normalizedRetrievalTerms(query) }

func retrievalDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
