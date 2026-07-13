package projectcourse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"inkwords-backend/shared/kernel/projectcourse"
)

// PlanBlueprint builds a deterministic, evidence-linked first blueprint.
// It deliberately does not infer reasons, learning outcomes, or prose from model output.
func PlanBlueprint(courseID string, snapshot projectcourse.SourceSnapshot, graph KnowledgeGraph, audience projectcourse.AudienceLevel) (projectcourse.Blueprint, projectcourse.CoverageMatrix, error) {
	if err := snapshot.Validate(); err != nil {
		return projectcourse.Blueprint{}, projectcourse.CoverageMatrix{}, err
	}
	if strings.TrimSpace(courseID) == "" {
		return projectcourse.Blueprint{}, projectcourse.CoverageMatrix{}, fmt.Errorf("course id is required")
	}
	if graph.CommitSHA != snapshot.ResolvedCommitSHA {
		return projectcourse.Blueprint{}, projectcourse.CoverageMatrix{}, fmt.Errorf("knowledge graph SHA does not match snapshot")
	}
	if err := audience.Validate(); err != nil {
		return projectcourse.Blueprint{}, projectcourse.CoverageMatrix{}, err
	}

	chapters := []projectcourse.Chapter{{
		ID: "chapter-" + stableID("project-map"), Title: "项目地图与主链路", Sort: 1, Enabled: true,
		Type: projectcourse.ChapterProjectMap, EvidenceIDs: evidenceIDsForFiles(graph.Files),
	}}
	files := append([]InventoryEntry(nil), graph.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	previousID := chapters[0].ID
	for index, file := range files {
		if file.Disposition == DispositionExcluded || file.Role == RoleGenerated || file.Role == RoleBinary {
			continue
		}
		chapterType := chapterTypeForRole(file.Role)
		id := "chapter-" + stableID(file.Path)
		chapters = append(chapters, projectcourse.Chapter{
			ID: id, Title: chapterTitle(file), Sort: index + 2, Enabled: true, Type: chapterType,
			PrerequisiteIDs: []string{previousID}, EvidenceIDs: evidenceIDsForFiles([]InventoryEntry{file}),
		})
		previousID = id
	}
	blueprint := projectcourse.Blueprint{
		CourseID: courseID, BlueprintVersion: 1, CommitSHA: snapshot.ResolvedCommitSHA, AudienceLevel: audience,
		Volumes: []projectcourse.Volume{{ID: "volume-" + stableID("foundation"), Title: "项目理解与模块实践", Sort: 1, Chapters: chapters}},
	}
	coverage := buildCoverage(graph, chapters)
	if err := blueprint.Validate(); err != nil {
		return projectcourse.Blueprint{}, projectcourse.CoverageMatrix{}, err
	}
	return blueprint, coverage, nil
}

func chapterTypeForRole(role FileRole) projectcourse.ChapterType {
	switch role {
	case RoleConfiguration, RoleBuildDeploy:
		return projectcourse.ChapterTechnicalTheory
	case RoleTransport, RoleApplication:
		return projectcourse.ChapterMainFlow
	case RoleTest, RoleExample:
		return projectcourse.ChapterHandsOnLab
	case RoleDocumentation:
		return projectcourse.ChapterDesignTradeoff
	default:
		return projectcourse.ChapterModuleDeepDive
	}
}

func chapterTitle(file InventoryEntry) string { return "模块实践：" + file.Path }

func stableID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func evidenceIDsForFiles(files []InventoryEntry) []string {
	ids := make([]string, 0, len(files))
	for _, file := range files {
		if file.Disposition == DispositionExcluded {
			continue
		}
		ids = append(ids, "evidence-"+stableID(file.Path+"\x00"+file.ContentHash))
	}
	sort.Strings(ids)
	return ids
}

func buildCoverage(graph KnowledgeGraph, chapters []projectcourse.Chapter) projectcourse.CoverageMatrix {
	chapterIDs := make([]string, 0, len(chapters))
	for _, chapter := range chapters {
		chapterIDs = append(chapterIDs, chapter.ID)
	}
	items := make([]projectcourse.CoverageItem, 0, len(graph.Files))
	for _, file := range graph.Files {
		items = append(items, projectcourse.CoverageItem{ID: file.Path, Kind: "file", Label: file.Path, ChapterIDs: chapterIDs, Covered: file.Disposition != DispositionExcluded})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return projectcourse.CoverageMatrix{Files: items}
}
