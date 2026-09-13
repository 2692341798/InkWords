package textbook

import (
	"sort"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// ClaimCandidate is an evidence-backed wording for one fact key before a chapter is generated.
// Key identity is decided upstream and must stay stable across source versions.
type ClaimCandidate struct {
	Key        string
	Text       string
	SourceRole sharedtextbook.SourceRole
	EvidenceID string
}

// SourceConflict records a factual disagreement that needs a human resolution.
// It intentionally does not select a winner or synthesize a replacement statement.
type SourceConflict struct {
	Key         string
	EvidenceIDs []string
}

// DetectSourceConflicts reports only primary-versus-official disagreements.
// User references are not eligible inputs for textbook facts, and disagreements
// within a single source role are handled by the source-curation workflow.
func DetectSourceConflicts(candidates []ClaimCandidate) []SourceConflict {
	groups := map[string]map[sharedtextbook.SourceRole]map[string][]string{}
	for _, candidate := range candidates {
		if candidate.Key == "" || candidate.Text == "" || candidate.EvidenceID == "" {
			continue
		}
		if candidate.SourceRole != sharedtextbook.SourceRolePrimary && candidate.SourceRole != sharedtextbook.SourceRoleOfficial {
			continue
		}
		if groups[candidate.Key] == nil {
			groups[candidate.Key] = map[sharedtextbook.SourceRole]map[string][]string{}
		}
		if groups[candidate.Key][candidate.SourceRole] == nil {
			groups[candidate.Key][candidate.SourceRole] = map[string][]string{}
		}
		groups[candidate.Key][candidate.SourceRole][candidate.Text] = append(groups[candidate.Key][candidate.SourceRole][candidate.Text], candidate.EvidenceID)
	}

	conflicts := make([]SourceConflict, 0)
	for key, byRole := range groups {
		primary := byRole[sharedtextbook.SourceRolePrimary]
		official := byRole[sharedtextbook.SourceRoleOfficial]
		if len(primary) == 0 || len(official) == 0 || !hasDifferentText(primary, official) {
			continue
		}
		conflicts = append(conflicts, SourceConflict{Key: key, EvidenceIDs: sortedEvidenceIDs(primary, official)})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Key < conflicts[j].Key })
	return conflicts
}

func hasDifferentText(primary, official map[string][]string) bool {
	texts := map[string]bool{}
	for text := range primary {
		texts[text] = true
	}
	for text := range official {
		texts[text] = true
	}
	return len(texts) > 1
}

func sortedEvidenceIDs(groups ...map[string][]string) []string {
	seen := map[string]bool{}
	for _, group := range groups {
		for _, ids := range group {
			for _, id := range ids {
				seen[id] = true
			}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
