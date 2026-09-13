package textbook

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestBuildGinRequestLifecycleSampleIsEvidenceBoundAndSelfStudyReady(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	require.Equal(t, ginRequestLifecycleChapterID, chapter.ID)
	require.Equal(t, sharedtextbook.AudienceFoundation, chapter.Audience)
	require.Equal(t, "deterministic_fixture", chapter.GenerationMode)
	require.Equal(t, "unverified", chapter.RuntimeVerification)
	require.Contains(t, chapter.Markdown, "**验证状态：未验证。**")
	require.NotContains(t, chapter.Markdown, "运行结果如下")

	report := RunSampleChapterQualityGates(chapter, pack)
	require.True(t, report.Passed, report.Failures)
	require.Equal(t, 5, report.CriticalClaimCount)
	require.Equal(t, report.CriticalClaimCount, report.CoveredCriticalClaimCount)
	require.True(t, report.ConceptStylePassed)
	require.True(t, report.HandsOnStylePassed)
	require.NoError(t, chapter.Scenario.Validate())
	require.NoError(t, chapter.Understanding.Validate())
	require.NoError(t, chapter.LearningArc.Validate())

	revision, err := CandidateRevision(chapter, pack, "book-contract-1", "style-sheet-1", "sha256:fixture-evidence")
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.RevisionKindCandidate, revision.Kind)
	require.NoError(t, revision.Validate())
}

func TestBuildGinRequestLifecycleSampleAdaptsContentAndPracticeToAudience(t *testing.T) {
	pack := ginEvidencePack(t)
	cases := []struct {
		name       string
		audience   sharedtextbook.AudienceLevel
		marker     string
		promptMark string
	}{
		{name: "foundation", audience: sharedtextbook.AudienceFoundation, marker: "不要求你先会 Go 或 Gin", promptMark: "先画出两条调用链"},
		{name: "programming", audience: sharedtextbook.AudienceProgramming, marker: "假设你已经会读 Go 函数、map 和单元测试", promptMark: "表驱动测试"},
		{name: "stack familiar", audience: sharedtextbook.AudienceStackFamiliar, marker: "假设你已经能创建 Gin 路由并使用调试器", promptMark: "生产源码符号"},
	}
	hashes := map[string]bool{}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			chapter, err := BuildGinRequestLifecycleSampleForAudience(pack, test.audience)
			require.NoError(t, err)
			require.Equal(t, test.audience, chapter.Audience)
			require.Contains(t, chapter.Markdown, test.marker)
			require.Contains(t, sharedtextbook.RenderPracticeSet(chapter.PracticeSet), test.promptMark)
			require.False(t, hashes[chapter.ContentHash], "each audience needs distinct manuscript content")
			hashes[chapter.ContentHash] = true

			report := RunSampleChapterQualityGates(chapter, pack)
			require.True(t, report.Passed, report.Failures)
		})
	}
}

func TestStyleProbeRequiresBothConceptExplanationAndHandsOnProcedure(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, "## 动手", "## 操作留空", 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.HandsOnStylePassed)
	require.True(t, report.ConceptStylePassed)
	require.Contains(t, report.Failures, "hands_on_procedure_missing: ## 动手")
}

func TestSampleQualityRejectsMissingOrDetachedPractice(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.PracticeSet.Tasks[0].ExpectedAnswer += "补充后的答案。"
	report := RunSampleChapterQualityGates(chapter, pack)
	require.Contains(t, report.Failures, "practice_set_manuscript_mismatch")
	chapter.PracticeSet.Tasks = nil
	report = RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, strings.Join(report.Failures, "\n"), "practice_set:")
}

func TestStyleProbeAcceptsCreatingAColdStartProjectWithoutLiteralOpen(t *testing.T) {
	markdown := strings.Join([]string{
		"## 先遇到一个真实问题",
		"## 先看全貌",
		"## 为什么要先登记路由",
		"## 跟着源码走三步",
		"## 动手",
		"在空文件夹新建 main.go，执行 go test ./...；预期测试通过。若失败，请检查 Go 版本。",
	}, "\n")

	report := ProbeSampleStyle(markdown)

	require.True(t, report.ConceptExplanationPassed)
	require.True(t, report.HandsOnProcedurePassed, report.Failures)
}

func TestSampleChapterQualityGatesRejectsCriticalClaimWithoutVisibleEvidence(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.ReplaceAll(chapter.Markdown, "[evidence:gin-routergroup-handle]", "")
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "claim_evidence_not_cited: gin-handle-combines-chain:gin-routergroup-handle")
	require.Contains(t, report.Failures, "critical_claim_coverage_incomplete")
}

func TestSampleChapterQualityGatesRejectMissingEvidenceAndAntiSelfStudyPhrases(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.EvidenceIDs = append(chapter.EvidenceIDs, "gin-routergroup-use")
	chapter.Markdown += "\n这很简单。"
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "missing_evidence_reference: gin-routergroup-use")
	require.Contains(t, report.Failures, "anti_self_study_phrase: 很简单")
}

func TestSampleChapterQualityGatesRejectsRepeatedUndefinedAcronym(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, "HTTP（超文本传输协议）", "HTTP", 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "undefined_repeated_acronym: HTTP")
}

func TestSampleChapterQualityGatesRejectsCircularTermDefinition(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown += "\n\n路由是路由规则本身。"
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "circular_term_definition: 路由")
}

func TestSampleChapterQualityGatesRejectsCodeBlockWithoutOrigin(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, "**代码来源：教学实现（main.go）。**", "**代码来源：未知。**", 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "code_origin_missing: code_block_1")
}

func TestSampleChapterQualityGatesRejectsGinIntegrationExampleMasqueradingAsTeachingImplementation(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, `package main`, `package main

import "github.com/gin-gonic/gin"`, 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "teaching_implementation_uses_production_library")
}

func TestSampleChapterQualityGatesRejectsRegistrationOnlyEvidenceForRequestMatching(t *testing.T) {
	pack := ginEvidencePack(t)
	pack.Evidence = pack.Evidence[:3]
	for evidenceID := range pack.Excerpts {
		if evidenceID == "gin-engine-serve-http" || evidenceID == "gin-engine-handle-http-request" || evidenceID == "gin-node-get-value" {
			delete(pack.Excerpts, evidenceID)
		}
	}
	chapter, err := BuildGinRequestLifecycleSample(ginEvidencePack(t))
	require.NoError(t, err)
	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "gin_request_lifecycle_evidence_missing: gin-engine-handle-http-request")
	require.Contains(t, report.Failures, "gin_request_lifecycle_evidence_missing: gin-node-get-value")
}

func TestSampleChapterQualityGatesRejectsInventedRuntimeOutputAndMissingPractice(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.ReplaceAll(strings.ReplaceAll(chapter.Markdown, "\n1. ", "\n首项 "), "\n5. ", "\n末项 ") + "\n终端输出如下：服务已启动。"
	chapter.ContentHash = digest(chapter.Markdown)
	report := RunSampleChapterQualityGates(chapter, pack)
	require.Contains(t, report.Failures, "unverified_runtime_evidence")
	require.Contains(t, report.Failures, "insufficient_independent_practice")
}

func TestSampleChapterQualityGatesRejectsEmptyOrOutOfOrderLearningStages(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, "### 完整示范：先跟着一行配置走完\n\n请按顺序读 GET → handle → addRoute 三个小节，并把每一步用“输入是什么、交给谁、解决什么”复述一遍。示范使用的是固定 Gin 快照中的三个片段，不是凭空画出的调用图。", "### 完整示范：先跟着一行配置走完\n\n好。", 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "learning_stage_has_no_substance: ### 完整示范")
}

func TestSampleChapterQualityGatesRejectsColdStartStepsWithoutExpectedResultOrRecovery(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.ReplaceAll(strings.ReplaceAll(chapter.Markdown, "预期", "结果"), "若", "当")
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "cold_start_operation_incomplete: expected_result,recovery")
}

func TestSampleChapterQualityGatesRejectsTeachingRouterWithoutMethodDimension(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, "addRoute(method, path string", "addRoute(path string", 1)
	chapter.Markdown = strings.Replace(chapter.Markdown, "findRoute(method, path string", "findRoute(path string", 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "teaching_method_dimension_missing: addRoute")
	require.Contains(t, report.Failures, "teaching_method_dimension_missing: findRoute")
}

func TestSampleChapterQualityGatesRejectsColdStartWithoutGoModuleInitialization(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, "go mod init example.com/gin-route-demo", "创建 go.mod", 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "teaching_go_module_init_missing")
}

func TestSampleChapterQualityGatesRejectsUnusedTeachingTestImport(t *testing.T) {
	pack := ginEvidencePack(t)
	chapter, err := BuildGinRequestLifecycleSample(pack)
	require.NoError(t, err)
	chapter.Markdown = strings.Replace(chapter.Markdown, `import "testing"`, `import (
	"net/http/httptest"
	"testing"
)`, 1)
	chapter.ContentHash = digest(chapter.Markdown)

	report := RunSampleChapterQualityGates(chapter, pack)
	require.False(t, report.Passed)
	require.Contains(t, report.Failures, "teaching_go_import_unused: main_test.go:net/http/httptest")
}

func TestFakeSampleGeneratorPinsContractAndCreatesOnlyCandidate(t *testing.T) {
	request := sampleGenerationRequest(t)
	request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}

	generation, err := FakeSampleGenerator{}.Generate(context.Background(), request)
	require.NoError(t, err)
	require.True(t, generation.Quality.Passed)
	require.Equal(t, sharedtextbook.RevisionKindCandidate, generation.Candidate.Kind)
	require.Equal(t, request.BookContract.RevisionID, generation.Candidate.BookContractRevision)
	require.Equal(t, request.StyleSheet.RevisionID, generation.Candidate.StyleSheetRevision)
	require.NotEqual(t, "", generation.Candidate.EvidencePackHash)
}

func TestFakeSampleGeneratorCoversAllApprovedAudiencesWithoutRelabeling(t *testing.T) {
	for _, audience := range []sharedtextbook.AudienceLevel{
		sharedtextbook.AudienceFoundation,
		sharedtextbook.AudienceProgramming,
		sharedtextbook.AudienceStackFamiliar,
	} {
		t.Run(string(audience), func(t *testing.T) {
			request := sampleGenerationRequest(t)
			request.Audience = audience
			request.BookContract.Reader.Audience = audience
			request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}

			generation, err := FakeSampleGenerator{}.Generate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, audience, generation.Chapter.Audience)
			require.Equal(t, generation.Chapter.Markdown, generation.Candidate.Markdown)
			require.True(t, generation.Quality.Passed, generation.Quality.Failures)
		})
	}
}

func ginEvidencePack(t *testing.T) EvidencePack {
	t.Helper()
	primary := sharedtextbook.SourceSnapshot{ID: "snapshot-gin-v1.12.0", SourceID: "gin-v1.12.0", Kind: sharedtextbook.SourceKindGitRepository, Role: sharedtextbook.SourceRolePrimary, Locator: "https://github.com/gin-gonic/gin", ResolvedVersion: "73726dc606796a025971fe451f0aa6f1b9b847f6", ContentHash: "sha256:gin-fixture", CapturedAt: time.Unix(1, 0).UTC()}
	ginSnapshot := primary
	ginSnapshot.ID = "snapshot-gin-go-v1.12.0"
	ginSnapshot.ContentHash = "sha256:gin-go-fixture"
	treeSnapshot := primary
	treeSnapshot.ID = "snapshot-tree-go-v1.12.0"
	treeSnapshot.ContentHash = "sha256:tree-go-fixture"
	official := sharedtextbook.SourceSnapshot{ID: "snapshot-gin-docs", SourceID: "gin-official-docs", Kind: sharedtextbook.SourceKindOfficialWeb, Role: sharedtextbook.SourceRoleOfficial, Locator: "https://gin-gonic.com/en/docs/", ResolvedVersion: "2026-09-02", ContentHash: "sha256:gin-docs-fixture", CapturedAt: time.Unix(1, 0).UTC()}
	makeEvidence := func(snapshot sharedtextbook.SourceSnapshot, id, documentID, path, symbol string, line, end int) sharedtextbook.EvidenceRef {
		return sharedtextbook.EvidenceRef{ID: id, SnapshotID: snapshot.ID, DocumentID: documentID, Locator: sharedtextbook.EvidenceLocator{Path: path, Symbol: symbol, StartLine: line, EndLine: end}, ContentHash: "sha256:" + id, Confidence: sharedtextbook.EvidenceConfidenceObserved, SourceRole: sharedtextbook.SourceRolePrimary}
	}
	evidence := []sharedtextbook.EvidenceRef{
		makeEvidence(primary, "gin-routergroup-get", "routergroup-go", "routergroup.go", "(*RouterGroup).GET", 115, 117),
		makeEvidence(primary, "gin-routergroup-handle", "routergroup-go", "routergroup.go", "(*RouterGroup).handle", 86, 88),
		makeEvidence(primary, "gin-engine-add-route", "routergroup-go", "routergroup.go", "(*Engine).addRoute", 364, 366),
		makeEvidence(ginSnapshot, "gin-engine-serve-http", "gin-go", "gin.go", "(*Engine).ServeHTTP", 661, 675),
		makeEvidence(ginSnapshot, "gin-engine-handle-http-request", "gin-go", "gin.go", "(*Engine).handleHTTPRequest", 690, 724),
		makeEvidence(treeSnapshot, "gin-node-get-value", "tree-go", "tree.go", "(*node).getValue", 418, 452),
	}
	return EvidencePack{PrimarySnapshot: primary, PrimarySnapshots: []sharedtextbook.SourceSnapshot{ginSnapshot, treeSnapshot}, OfficialSources: []sharedtextbook.SourceSnapshot{official}, Evidence: evidence, Excerpts: map[string]string{
		"gin-routergroup-get":            "return group.handle(http.MethodGet, relativePath, handlers)",
		"gin-routergroup-handle":         "handlers = group.combineHandlers(handlers)",
		"gin-engine-add-route":           "root.addRoute(path, handlers)",
		"gin-engine-serve-http":          "engine.handleHTTPRequest(c)",
		"gin-engine-handle-http-request": "按请求方法选择路由树，调用 root.getValue(rPath, ...)，把匹配 handlers 交给 Context.Next",
		"gin-node-get-value":             "getValue 比较当前节点前缀并沿匹配的子节点继续查找",
	}}
}
