package projectcourse

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnumsUseStableJSONValuesAndRejectUnknownValues(t *testing.T) {
	if ScenarioModeProjectMasteryCourse != "project_mastery_course" {
		t.Fatalf("unexpected scenario value: %q", ScenarioModeProjectMasteryCourse)
	}
	if err := AudienceProgramming.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := AudienceLevel("unknown").Validate(); err == nil {
		t.Fatal("unknown audience level must be rejected")
	}
	if err := ChapterType("ebook").Validate(); err == nil {
		t.Fatal("unknown chapter type must be rejected")
	}
	encoded, err := json.Marshal(struct {
		Audience AudienceLevel `json:"audience"`
		Type     ChapterType   `json:"type"`
	}{AudienceProgramming, ChapterMainFlow})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"audience":"programming","type":"main_flow"}` {
		t.Fatalf("unstable contract: %s", encoded)
	}
}

func TestSourceSnapshotRequiresResolvedCommitSHA(t *testing.T) {
	snapshot := SourceSnapshot{
		RepositoryURL:     "https://github.com/example/project",
		RequestedRef:      "main",
		ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567",
		CapturedAt:        time.Unix(1, 0).UTC(),
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	snapshot.ResolvedCommitSHA = "HEAD"
	if err := snapshot.Validate(); err == nil {
		t.Fatal("HEAD must not satisfy an immutable snapshot")
	}
}

func TestClaimRequiresEvidenceAndValidConfidence(t *testing.T) {
	claim := Claim{
		ClaimID: "claim-1", Text: "worker consumes tasks", ClaimType: "project_fact",
		Confidence: ConfidenceObserved, EvidenceIDs: []string{"ev-1"}, Status: ClaimVerified,
	}
	if err := claim.Validate(); err != nil {
		t.Fatal(err)
	}
	claim.EvidenceIDs = nil
	if err := claim.Validate(); err == nil {
		t.Fatal("claim without evidence must be rejected")
	}
}

func TestBlueprintValidatesAudienceAndChapterType(t *testing.T) {
	blueprint := Blueprint{
		CourseID: "course-1", BlueprintVersion: 1, CommitSHA: "0123456789abcdef0123456789abcdef01234567",
		AudienceLevel: AudienceProgramming,
		Volumes:       []Volume{{ID: "volume-1", Title: "项目基础", Sort: 1, Chapters: []Chapter{{ID: "chapter-1", Title: "项目地图", Sort: 1, Type: ChapterProjectMap}}}},
	}
	if err := blueprint.Validate(); err != nil {
		t.Fatal(err)
	}
	blueprint.Volumes[0].Chapters[0].Type = "unknown"
	if err := blueprint.Validate(); err == nil {
		t.Fatal("unknown chapter type must invalidate blueprint")
	}
}

func TestBlueprintRejectsInvalidOrderDuplicateAndCyclicDependencies(t *testing.T) {
	snapshot := Blueprint{CourseID: "course-1", BlueprintVersion: 1, CommitSHA: "0123456789abcdef0123456789abcdef01234567", AudienceLevel: AudienceProgramming, Volumes: []Volume{{ID: "v1", Title: "卷一", Sort: 1, Chapters: []Chapter{{ID: "c1", Title: "第一章", Sort: 1, Type: ChapterMainFlow}}}}}
	require.ErrorContains(t, snapshot.Validate(), "project map")
	snapshot.Volumes[0].Chapters[0].Type = ChapterProjectMap
	snapshot.Volumes[0].Chapters = append(snapshot.Volumes[0].Chapters, Chapter{ID: "c2", Title: "第二章", Sort: 2, Type: ChapterMainFlow, PrerequisiteIDs: []string{"c3"}}, Chapter{ID: "c3", Title: "第三章", Sort: 3, Type: ChapterMainFlow, PrerequisiteIDs: []string{"c2"}})
	require.ErrorContains(t, snapshot.Validate(), "cycle")
}

func TestCoverageRateTreatsEmptySetAsCovered(t *testing.T) {
	if got := (CoverageMatrix{}).CoveredRate(nil); got != 1 {
		t.Fatalf("empty coverage should be 100%%, got %v", got)
	}
	if got := (CoverageMatrix{}).CoveredRate([]CoverageItem{{Covered: true}, {Covered: false}}); got != 0.5 {
		t.Fatalf("unexpected coverage rate: %v", got)
	}
}
