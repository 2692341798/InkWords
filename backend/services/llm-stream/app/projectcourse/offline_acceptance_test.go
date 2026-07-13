package projectcourse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type offlineFixtureManifest struct {
	Entries []struct {
		Path string `json:"path"`
	} `json:"entries"`
}

// TestOfflineInkWordsAcceptance exercises the deterministic course path without
// a model, network, Docker, or execution of the target repository. It is a
// repeatable contract check, not evidence of real-model or sandbox dogfood.
func TestOfflineInkWordsAcceptance(t *testing.T) {
	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "inkwords-fixture", "manifest.json"))
	require.NoError(t, err)
	var fixture offlineFixtureManifest
	require.NoError(t, json.Unmarshal(manifestBytes, &fixture))

	snapshot := sharedkernel.SourceSnapshot{
		RepositoryURL:     "https://github.com/2692341798/InkWords",
		RequestedRef:      "main",
		ResolvedCommitSHA: "f14bd1dbc1e568a2335341dd4df0f6c0574bee35",
		CapturedAt:        time.Unix(1, 0).UTC(),
	}
	graph := KnowledgeGraph{CommitSHA: snapshot.ResolvedCommitSHA, Files: fixtureInventory(fixture.Entries)}

	var baseline sharedkernel.Blueprint
	for _, audience := range []sharedkernel.AudienceLevel{
		sharedkernel.AudienceFoundation,
		sharedkernel.AudienceProgramming,
		sharedkernel.AudienceStackFamiliar,
	} {
		blueprint, coverage, planErr := PlanBlueprint("offline-course", snapshot, graph, audience)
		require.NoError(t, planErr)
		require.Equal(t, 1.0, coverage.CoveredRate(coverage.Files))
		if baseline.CourseID == "" {
			baseline = blueprint
		} else {
			require.Equal(t, blueprintEvidenceIDs(baseline), blueprintEvidenceIDs(blueprint))
		}

		documents := make([]ChapterDocument, 0)
		for _, volume := range blueprint.Volumes {
			for _, chapter := range volume.Chapters {
				if !chapter.Enabled {
					continue
				}
				pack, packErr := BuildEvidencePack(snapshot, chapter.ID, chapter.EvidenceIDs, graph, []OfficialSource{{Technology: "fixture", URL: "https://go.dev/doc/", Content: "fixture official source", ContentHash: contentHash([]byte("fixture official source"))}})
				require.NoError(t, packErr)
				document, report, buildErr := buildOfflineChapter(chapter, pack)
				require.NoError(t, buildErr)
				require.Equal(t, sharedkernel.GatePass, report.Result, chapter.ID)
				documents = append(documents, document)
			}
		}
		require.NoError(t, ValidateChapterSet(documents, blueprintDependencies(blueprint)))
	}
}

func fixtureInventory(entries []struct {
	Path string `json:"path"`
}) []InventoryEntry {
	result := make([]InventoryEntry, 0, len(entries))
	for _, entry := range entries {
		content := "package fixture\n\n// " + entry.Path + "\nfunc Fixture() {}"
		if filepath.Ext(entry.Path) == ".md" || filepath.Ext(entry.Path) == ".yml" {
			content = "# fixture " + entry.Path + "\nservices:\n  app:\n    image: fixture"
		}
		if filepath.Ext(entry.Path) == ".ts" {
			content = "export const fixture = '" + entry.Path + "'"
		}
		result = append(result, InventoryEntry{Path: entry.Path, Role: fixtureRole(entry.Path), Disposition: DispositionCovered, ContentHash: contentHash([]byte(content)), Size: len(content), Content: content})
	}
	return result
}

func fixtureRole(path string) FileRole {
	switch {
	case filepath.Ext(path) == ".md":
		return RoleDocumentation
	case filepath.Ext(path) == ".yml":
		return RoleConfiguration
	case filepath.Ext(path) == ".ts":
		return RoleApplication
	case filepath.Ext(path) == ".go" && (len(path) >= 5 && path[len(path)-5:] == "_test.go"):
		return RoleTest
	case path == "scripts":
		return RoleBuildDeploy
	default:
		return RoleApplication
	}
}

func buildOfflineChapter(chapter sharedkernel.Chapter, pack EvidencePack) (ChapterDocument, QualityGateReport, error) {
	required, _, requiresLab, ok := chapterContractFor(chapter.Type)
	if !ok {
		return ChapterDocument{}, QualityGateReport{}, nil
	}
	markdown := "# " + chapter.Title + "\n"
	for _, section := range required {
		markdown += "## " + section + "\n离线夹具验收内容。\n"
	}
	var lab *sharedkernel.LabManifest
	if requiresLab {
		manifest, err := BuildLabManifest("go", "1.25", []LabPatch{{Path: "main.go", Content: "package main"}}, [][]LabPatch{{{Path: "main.go", Content: "package main\nfunc main() {}"}}}, []LabPatch{{Path: "main_test.go", Content: "package main"}}, []LabPatch{{Path: "variant_test.go", Content: "package main"}})
		if err != nil {
			return ChapterDocument{}, QualityGateReport{}, err
		}
		lab = &manifest
	}
	claim := sharedkernel.Claim{ClaimID: "claim-" + stableID(chapter.ID), Text: "夹具章节证据存在", ClaimType: "project_fact", Confidence: sharedkernel.ConfidenceObserved, EvidenceIDs: []string{pack.SourceEvidence[0].EvidenceID}, Status: sharedkernel.ClaimVerified}
	return BuildChapterDocument(chapter, pack, ClaimPlan{Claims: []sharedkernel.Claim{claim}, EvidenceIDs: []string{pack.SourceEvidence[0].EvidenceID}}, markdown, lab, true)
}

func blueprintEvidenceIDs(blueprint sharedkernel.Blueprint) [][]string {
	result := make([][]string, 0)
	for _, volume := range blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			result = append(result, chapter.EvidenceIDs)
		}
	}
	return result
}

func blueprintDependencies(blueprint sharedkernel.Blueprint) map[string][]string {
	result := make(map[string][]string)
	for _, volume := range blueprint.Volumes {
		for _, chapter := range volume.Chapters {
			result[chapter.ID] = chapter.PrerequisiteIDs
		}
	}
	return result
}
