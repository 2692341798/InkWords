package textbook

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	pathpkg "path"
	"regexp"
	"strconv"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// QualityReport separates deterministic checks from human reader trials.
type QualityReport struct {
	ContractVersion           string                     `json:"contract_version"`
	Passed                    bool                       `json:"passed"`
	Failures                  []string                   `json:"failures,omitempty"`
	CriticalClaimCount        int                        `json:"critical_claim_count"`
	CoveredCriticalClaimCount int                        `json:"covered_critical_claim_count"`
	EvidenceReferenceCount    int                        `json:"evidence_reference_count"`
	ConceptStylePassed        bool                       `json:"concept_style_passed"`
	HandsOnStylePassed        bool                       `json:"hands_on_style_passed"`
	ManualReviewRequired      bool                       `json:"manual_review_required"`
	ManualReviewDimensions    []string                   `json:"manual_review_dimensions"`
	Advisories                []SoftQualityAdvisory      `json:"advisories,omitempty"`
	FailureDiagnostics        []QualityFailureDiagnostic `json:"failure_diagnostics,omitempty"`
}

var evidenceReference = regexp.MustCompile(`\[evidence:([^\]]+)\]`)
var fencedGoCodeBlock = regexp.MustCompile("(?s)```go[ \\t]*\\n(.*?)```")

var learningStageHeadings = []string{
	"### 先验激活",
	"### 问题体验",
	"### 全貌预告",
	"### 完整示范",
	"### 引导练习",
	"### 脚手架渐退",
	"### 独立迁移",
	"### 复述并关联",
	"### 延迟提取",
	"### 根据表现调整",
}

// RunSampleChapterQualityGates rejects common forms of anti-self-study prose.
func RunSampleChapterQualityGates(chapter SampleChapter, pack EvidencePack) QualityReport {
	failures := make([]string, 0)
	styleProbe := probeSampleStyleForProfile(chapter.Markdown, chapter.Profile)
	failures = append(failures, styleProbe.Failures...)
	if err := pack.Validate(); err != nil {
		failures = append(failures, "evidence_pack: "+err.Error())
	}
	if err := chapter.Scenario.Validate(); err != nil {
		failures = append(failures, "scenario_frame: "+err.Error())
	}
	if err := chapter.Understanding.Validate(); err != nil {
		failures = append(failures, "understanding_chain: "+err.Error())
	}
	if err := chapter.LearningArc.Validate(); err != nil {
		failures = append(failures, "learning_arc: "+err.Error())
	}
	if err := chapter.PracticeSet.Validate(chapter.EvidenceIDs); err != nil {
		failures = append(failures, "practice_set: "+err.Error())
	} else if !strings.Contains(chapter.Markdown, sharedtextbook.RenderPracticeSet(chapter.PracticeSet)) {
		failures = append(failures, "practice_set_manuscript_mismatch")
	}
	for _, heading := range headingsForProfile(chapter.Profile, []string{"## 先遇到一个真实问题", "## 先看全貌", "## 为什么要先登记路由", "## 跟着源码走三步", "## 按自然学习曲线练习", "## 动手", "## 用自己的话检查理解", "## 你现在应该能解释什么"}) {
		if !strings.Contains(chapter.Markdown, heading) {
			failures = append(failures, "missing_required_section: "+heading)
		}
	}
	failures = append(failures, validateLearningStageBodies(chapter.Markdown)...)
	for _, phrase := range []string{"显然", "很简单", "留给读者", "顾名思义"} {
		if strings.Contains(chapter.Markdown, phrase) {
			failures = append(failures, "anti_self_study_phrase: "+phrase)
		}
	}
	for _, acronym := range UndefinedRepeatedAcronyms(chapter.Markdown) {
		failures = append(failures, "undefined_repeated_acronym: "+acronym)
	}
	for _, term := range CircularTermDefinitions(chapter.Markdown) {
		failures = append(failures, "circular_term_definition: "+term)
	}
	for _, codeBlock := range MissingCodeBlockOrigins(chapter.Markdown) {
		failures = append(failures, "code_origin_missing: "+codeBlock)
	}
	switch chapter.Profile {
	case "", sharedtextbook.ChapterProfileConcept:
		failures = append(failures, validateGinTeachingImplementation(chapter.Markdown)...)
	case sharedtextbook.ChapterProfileHandsOn:
		failures = append(failures, validateGinIntegrationImplementation(chapter.Markdown)...)
	default:
		failures = append(failures, "unsupported_chapter_profile")
	}
	if !strings.Contains(chapter.Markdown, "**验证状态：未验证。**") || containsAny(chapter.Markdown, "运行结果如下", "截图如下", "终端输出如下", "实际输出为") {
		failures = append(failures, "unverified_runtime_evidence")
	}
	if !strings.Contains(chapter.Markdown, "若") || !strings.Contains(chapter.Markdown, "请") {
		failures = append(failures, "missing_recovery_path")
	}
	if len(chapter.Scenario.Mapping) < 3 || !strings.Contains(chapter.Scenario.Boundary, "不代表") {
		failures = append(failures, "analogy_boundary_missing")
	}
	if !strings.Contains(chapter.Understanding.Alternatives, "至少") && strings.Count(chapter.Understanding.Alternatives, "；") < 2 {
		failures = append(failures, "implementation_alternatives_missing")
	}
	if !containsAny(chapter.Markdown, "打开", "新建", "执行", "断点") {
		failures = append(failures, "missing_actionable_operation")
	}
	if missing := coldStartOperationGaps(chapter.Markdown); len(missing) > 0 {
		failures = append(failures, "cold_start_operation_incomplete: "+strings.Join(missing, ","))
	}
	if strings.Count(chapter.Markdown, "\n1. ") < 1 || strings.Count(chapter.Markdown, "\n5. ") < 1 {
		failures = append(failures, "insufficient_independent_practice")
	}
	knownEvidence := make(map[string]bool, len(pack.Evidence))
	for _, evidence := range pack.Evidence {
		knownEvidence[evidence.ID] = true
	}
	referenced := make(map[string]bool)
	for _, match := range evidenceReference.FindAllStringSubmatch(chapter.Markdown, -1) {
		referenced[match[1]] = true
		if !knownEvidence[match[1]] {
			failures = append(failures, "unsupported_claim: "+match[1])
		}
	}
	for _, required := range chapter.EvidenceIDs {
		if !referenced[required] {
			failures = append(failures, "missing_evidence_reference: "+required)
		}
	}
	for _, required := range []string{"gin-routergroup-get", "gin-routergroup-handle", "gin-engine-add-route", "gin-engine-serve-http", "gin-engine-handle-http-request", "gin-node-get-value"} {
		if !knownEvidence[required] {
			failures = append(failures, "gin_request_lifecycle_evidence_missing: "+required)
		}
	}
	criticalClaims, coveredCriticalClaims := validateCriticalClaims(chapter.Claims, knownEvidence, referenced, &failures)
	if chapter.ContentHash != digest(chapter.Markdown) {
		failures = append(failures, "content_hash_mismatch")
	}
	return QualityReport{ContractVersion: sharedtextbook.SampleQualityContractVersion, Passed: len(failures) == 0, Failures: failures, CriticalClaimCount: criticalClaims, CoveredCriticalClaimCount: coveredCriticalClaims, EvidenceReferenceCount: len(referenced), ConceptStylePassed: styleProbe.ConceptExplanationPassed, HandsOnStylePassed: styleProbe.HandsOnProcedurePassed, ManualReviewRequired: true, ManualReviewDimensions: manualReviewDimensions(), Advisories: AssessSampleChapterSoftQuality(chapter.Markdown), FailureDiagnostics: DiagnoseUndefinedAcronyms(chapter.Markdown)}
}

// validateGinTeachingImplementation distinguishes a from-scratch mechanism
// model from an integration example that merely imports Gin. This does not
// claim the code ran; course-runner remains the only authority for execution.
func validateGinTeachingImplementation(markdown string) []string {
	failures := make([]string, 0)
	blocks := fencedCodeBlock.FindAllString(markdown, -1)
	if len(blocks) < 2 {
		failures = append(failures, "teaching_implementation_requires_source_and_test")
	}
	joined := strings.Join(blocks, "\n")
	if strings.Contains(joined, "github.com/gin-gonic/gin") {
		failures = append(failures, "teaching_implementation_uses_production_library")
	}
	for _, symbol := range []string{"type routeNode struct", "addRoute", "findRoute"} {
		if !strings.Contains(joined, symbol) {
			failures = append(failures, "teaching_mechanism_symbol_missing: "+symbol)
		}
	}
	for _, functionName := range []string{"addRoute", "findRoute"} {
		if !functionParametersContain(joined, functionName, "method", "path") {
			failures = append(failures, "teaching_method_dimension_missing: "+functionName)
		}
	}
	if !strings.Contains(joined, "func Test") || !strings.Contains(markdown, "go test ./...") {
		failures = append(failures, "teaching_implementation_tests_missing")
	}
	if !strings.Contains(markdown, "go mod init") {
		failures = append(failures, "teaching_go_module_init_missing")
	}
	if !strings.Contains(markdown, "非生产用途") || !strings.Contains(markdown, "省略") {
		failures = append(failures, "teaching_implementation_boundary_missing")
	}
	failures = append(failures, validateTeachingGoSources(markdown)...)
	return failures
}

// validateTeachingGoSources performs source-only checks. It deliberately does
// not execute model output or claim runtime verification; course-runner remains
// the authority for go test and observable execution evidence.
func validateTeachingGoSources(markdown string) []string {
	matches := fencedGoCodeBlock.FindAllStringSubmatch(markdown, -1)
	if len(matches) != 2 {
		return []string{"teaching_go_files_invalid"}
	}
	failures := make([]string, 0)
	seenFiles := make(map[string]bool, len(matches))
	for index, match := range matches {
		filename := fmt.Sprintf("teaching_%d.go", index+1)
		file, err := parser.ParseFile(token.NewFileSet(), filename, match[1], parser.AllErrors)
		if err != nil {
			failures = append(failures, "teaching_go_parse_failed: "+filename)
			continue
		}
		filename = teachingGoFilename(file, index)
		if seenFiles[filename] {
			failures = append(failures, "teaching_go_file_layout_invalid: "+filename)
			continue
		}
		seenFiles[filename] = true
		if file.Name.Name != "main" {
			failures = append(failures, "teaching_go_package_invalid: "+filename)
		}
		failures = append(failures, unusedTeachingGoImports(filename, file)...)
	}
	for _, required := range []string{"main.go", "main_test.go"} {
		if !seenFiles[required] {
			failures = append(failures, "teaching_go_file_layout_invalid: missing "+required)
		}
	}
	return failures
}

func teachingGoFilename(file *ast.File, index int) string {
	hasMain, hasTest := false, false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		hasMain = hasMain || function.Name.Name == "main"
		hasTest = hasTest || (strings.HasPrefix(function.Name.Name, "Test") && function.Name.Name != "TestMain")
	}
	// Comments, strings and receiver methods cannot establish executable
	// file identity. Mixed declarations must not silently select one role.
	if hasTest && !hasMain {
		return "main_test.go"
	}
	if hasMain && !hasTest {
		return "main.go"
	}
	return fmt.Sprintf("teaching_%d.go", index+1)
}

func unusedTeachingGoImports(filename string, file *ast.File) []string {
	usedQualifiers := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if qualifier, ok := selector.X.(*ast.Ident); ok && qualifier.Obj == nil {
			usedQualifiers[qualifier.Name] = true
		}
		return true
	})

	failures := make([]string, 0)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			failures = append(failures, "teaching_go_import_invalid: "+filename)
			continue
		}
		qualifier := pathpkg.Base(importPath)
		if spec.Name != nil {
			qualifier = spec.Name.Name
		}
		switch qualifier {
		case "_":
			continue
		case ".":
			failures = append(failures, "teaching_go_dot_import_forbidden: "+filename+":"+importPath)
			continue
		}
		if !usedQualifiers[qualifier] {
			failures = append(failures, "teaching_go_import_unused: "+filename+":"+importPath)
		}
	}
	return failures
}

func functionParametersContain(code, functionName string, required ...string) bool {
	declaration := regexp.MustCompile(`func\s*(?:\([^)]*\)\s*)?` + regexp.QuoteMeta(functionName) + `\s*\(([^)]*)\)`)
	match := declaration.FindStringSubmatch(code)
	if len(match) != 2 {
		return false
	}
	for _, name := range required {
		if !strings.Contains(match[1], name) {
			return false
		}
	}
	return true
}

// validateCriticalClaims makes source-fact coverage auditable. A claim is covered
// only when every cited evidence id exists in the frozen pack and is visibly cited
// in the manuscript; prose alone cannot upgrade an assertion to verified.
func validateCriticalClaims(claims []sharedtextbook.Claim, knownEvidence, referenced map[string]bool, failures *[]string) (int, int) {
	critical, covered := 0, 0
	for _, claim := range claims {
		if err := claim.Validate(); err != nil {
			*failures = append(*failures, "invalid_claim: "+claim.ID)
			continue
		}
		if !claim.Critical {
			continue
		}
		critical++
		claimCovered := claim.Status == sharedtextbook.ClaimStatusVerified
		for _, evidenceID := range claim.EvidenceIDs {
			if !knownEvidence[evidenceID] {
				*failures = append(*failures, "claim_uses_unknown_evidence: "+claim.ID+":"+evidenceID)
				claimCovered = false
			}
			if !referenced[evidenceID] {
				*failures = append(*failures, "claim_evidence_not_cited: "+claim.ID+":"+evidenceID)
				claimCovered = false
			}
		}
		if claimCovered {
			covered++
		}
	}
	if critical == 0 {
		*failures = append(*failures, "missing_critical_claims")
	} else if covered != critical {
		*failures = append(*failures, "critical_claim_coverage_incomplete")
	}
	return critical, covered
}

func containsAny(content string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(content, phrase) {
			return true
		}
	}
	return false
}

// validateLearningStageBodies prevents a candidate from satisfying the natural
// learning-curve contract with a list of empty headings. It is intentionally a
// structural guard, not a claim that text length proves pedagogical quality.
func validateLearningStageBodies(markdown string) []string {
	failures := make([]string, 0)
	previous := -1
	bodies := make(map[string]string, len(learningStageHeadings))
	for _, heading := range learningStageHeadings {
		index := strings.Index(markdown, heading)
		if index < 0 {
			failures = append(failures, "missing_required_section: "+heading)
			continue
		}
		if index <= previous {
			failures = append(failures, "learning_stage_out_of_order: "+heading)
		}
		previous = index
		body := sectionBody(markdown, index+len(heading))
		bodies[heading] = body
		if len([]rune(strings.TrimSpace(body))) < 32 {
			failures = append(failures, "learning_stage_has_no_substance: "+heading)
		}
	}
	if !strings.Contains(bodies["### 引导练习"], "提示") {
		failures = append(failures, "guided_practice_missing_hint")
	}
	if !strings.Contains(bodies["### 脚手架渐退"], "提示") {
		failures = append(failures, "scaffolding_fade_missing_hint_removal")
	}
	if !containsAny(bodies["### 独立迁移"], "从零", "独立") {
		failures = append(failures, "independent_transfer_missing_independent_work")
	}
	return failures
}

func sectionBody(markdown string, start int) string {
	remaining := markdown[start:]
	nextSection := len(remaining)
	for _, marker := range []string{"\n### ", "\n## "} {
		if index := strings.Index(remaining, marker); index >= 0 && index < nextSection {
			nextSection = index
		}
	}
	return remaining[:nextSection]
}

// coldStartOperationGaps verifies only checkable runbook facts. It does not
// infer whether an operation really worked; runtime proof remains a separate
// verification artifact and must stay unverified until the runner records it.
func coldStartOperationGaps(markdown string) []string {
	index := strings.Index(markdown, "## 动手")
	if index < 0 {
		return []string{"section"}
	}
	body := sectionBody(markdown, index+len("## 动手"))
	requirements := []struct {
		name    string
		matches []string
	}{
		{name: "tool_or_version", matches: []string{"版本", "v1.", "VS Code", "GoLand", "Visual Studio"}},
		{name: "path_or_location", matches: []string{"文件夹", "目录", "路径", "终端进入"}},
		{name: "expected_result", matches: []string{"预期", "应该看到"}},
		{name: "recovery", matches: []string{"若", "如果"}},
	}
	missing := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		if !containsAny(body, requirement.matches...) {
			missing = append(missing, requirement.name)
		}
	}
	return missing
}

// CandidateRevision creates a contract-valid, explicitly unapproved revision envelope.
func CandidateRevision(chapter SampleChapter, pack EvidencePack, bookContractRevision, styleSheetRevision, evidencePackHash string) (sharedtextbook.ChapterRevision, error) {
	if !RunSampleChapterQualityGates(chapter, pack).Passed {
		return sharedtextbook.ChapterRevision{}, fmt.Errorf("sample chapter must pass quality gates before candidate envelope")
	}
	return sharedtextbook.ChapterRevision{ID: chapter.ID + "-candidate", ChapterID: chapter.ID, Kind: sharedtextbook.RevisionKindCandidate, Markdown: chapter.Markdown, DocumentHash: chapter.ContentHash, ContentHash: chapter.ContentHash, BookContractRevision: bookContractRevision, StyleSheetRevision: styleSheetRevision, EvidencePackHash: evidencePackHash}, nil
}
