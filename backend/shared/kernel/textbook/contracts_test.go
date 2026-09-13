package textbook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAudienceAndChapterProfileRejectUnknownValues(t *testing.T) {
	require.NoError(t, AudienceFoundation.Validate())
	require.NoError(t, ChapterProfileConcept.Validate())
	require.Error(t, AudienceLevel("expert_only").Validate())
	require.Error(t, ChapterProfile("essay").Validate())
}

func TestSampleHumanReviewRequiresEveryDimensionAtLeastThree(t *testing.T) {
	contract := CurrentSampleHumanReviewContract()
	require.Equal(t, SampleHumanReviewContractVersion, contract.ContractVersion)
	require.Equal(t, SampleManualReviewDimensions(), contract.Dimensions)
	require.Equal(t, 3, contract.MinimumScore)
	require.Equal(t, 4, contract.MaximumScore)
	require.Equal(t, 8, contract.NoteMinRunes)
	require.Equal(t, 2000, contract.NoteMaxRunes)

	scores := make([]DimensionScore, 0, len(SampleManualReviewDimensions()))
	for _, dimension := range SampleManualReviewDimensions() {
		scores = append(scores, DimensionScore{Dimension: dimension, Score: 3})
	}
	review := SampleHumanReview{ContractVersion: SampleHumanReviewContractVersion, DimensionScores: scores}
	require.NoError(t, review.Validate())

	review.DimensionScores[0].Score = 2
	require.ErrorContains(t, review.Validate(), "at least 3/4")
	review.DimensionScores[0].Score = 3
	review.DimensionScores[0].Dimension = "未定义维度"
	require.ErrorContains(t, review.Validate(), "dimensions")
}

func TestSourceSnapshotAndEvidenceRequireImmutableLocations(t *testing.T) {
	snapshot := SourceSnapshot{
		ID:              "snapshot-gin",
		SourceID:        "source-gin",
		Kind:            SourceKindGitRepository,
		Role:            SourceRolePrimary,
		Locator:         "https://github.com/gin-gonic/gin",
		ResolvedVersion: "73726dc606796a025971fe451f0aa6f1b9b847f6",
		ContentHash:     "sha256:0123456789abcdef",
		CapturedAt:      time.Unix(1, 0).UTC(),
	}
	require.NoError(t, snapshot.Validate())

	evidence := EvidenceRef{
		ID:          "evidence-routergroup-use",
		SnapshotID:  snapshot.ID,
		DocumentID:  "document-routergroup",
		Locator:     EvidenceLocator{Path: "routergroup.go", Symbol: "(*RouterGroup).Use", StartLine: 65, EndLine: 68},
		ContentHash: "sha256:abcdef0123456789",
		Confidence:  EvidenceConfidenceObserved,
		SourceRole:  SourceRolePrimary,
	}
	require.NoError(t, evidence.Validate())
	claim := Claim{ID: "claim-router", Text: "路由组会把处理函数交给 Engine 注册", Kind: "source_fact", Critical: true, EvidenceIDs: []string{evidence.ID}, Status: ClaimStatusVerified}
	require.NoError(t, claim.Validate())

	snapshot.ResolvedVersion = "main"
	require.Error(t, snapshot.Validate())
	evidence.Locator.StartLine = 0
	require.Error(t, evidence.Validate())
	claim.EvidenceIDs = nil
	require.Error(t, claim.Validate())
}

func TestBookContractAndStyleSheetFreezeReaderAndWritingRules(t *testing.T) {
	contract := BookContract{
		RevisionID:     "book-contract-1",
		ProjectID:      "project-1",
		RevisionNumber: 1,
		ContentHash:    "sha256:book-contract",
		Reader: ReaderModel{
			Audience:             AudienceFoundation,
			KnownKnowledge:       []string{"能打开终端"},
			ForbiddenAssumptions: []string{"不了解 HTTP 路由"},
			LearningOutcomes:     []string{"能够解释请求如何抵达处理函数"},
		},
		Promise:            "用可运行证据讲清 Gin 请求链路。",
		ChapterProfiles:    []ChapterProfile{ChapterProfileConcept, ChapterProfileHandsOn},
		TerminologyVersion: "terms-1",
		PublicationProfile: "personal_learning",
	}
	require.NoError(t, contract.Validate())

	sheet := StyleSheet{
		RevisionID:       "style-sheet-1",
		ProjectID:        contract.ProjectID,
		RevisionNumber:   1,
		ContentHash:      "sha256:style-sheet",
		Language:         "zh-CN",
		TerminologyRules: []string{"先用白话解释，再给精确术语。"},
		CodeRules:        []string{"代码块必须说明来源或验证状态。"},
		CitationRules:    []string{"关键事实必须引用固定快照。"},
		ForbiddenPhrases: []string{"显然", "留给读者"},
	}
	require.NoError(t, sheet.Validate())

	contract.ChapterProfiles = nil
	require.Error(t, contract.Validate())
	sheet.CitationRules = nil
	require.Error(t, sheet.Validate())
}

func TestSourceDocumentsAndChunksKeepStructuredProvenance(t *testing.T) {
	document := SourceDocument{
		ID:               "document-routergroup",
		SnapshotID:       "snapshot-gin",
		CanonicalLocator: "https://github.com/gin-gonic/gin/blob/73726dc606796a025971fe451f0aa6f1b9b847f6/routergroup.go",
		Title:            "routergroup.go",
		MediaType:        "text/x-go",
		ContentHash:      "sha256:routergroup",
		ArtifactPath:     "sources/gin-v1.12.0/routergroup.go",
	}
	require.NoError(t, document.Validate())

	chunk := SourceChunk{
		ID:          "chunk-routergroup-use",
		DocumentID:  document.ID,
		Ordinal:     1,
		HeadingPath: []string{"RouterGroup", "Use"},
		Locator:     EvidenceLocator{Path: "routergroup.go", Symbol: "(*RouterGroup).Use", StartLine: 65, EndLine: 68},
		TextHash:    "sha256:routergroup-use",
		SearchText:  "Use adds middleware to the group",
	}
	require.NoError(t, chunk.Validate())

	chunk.Ordinal = 0
	require.Error(t, chunk.Validate())
}

func TestBlueprintRejectsCyclesAndReferenceFirstTeaching(t *testing.T) {
	blueprint := Blueprint{
		RevisionID:           "blueprint-1",
		ProjectID:            "project-1",
		RevisionNumber:       1,
		ContentHash:          "sha256:blueprint",
		BookContractRevision: "book-contract-1",
		StyleSheetRevision:   "style-sheet-1",
		Volumes: []BlueprintVolume{{
			ID:    "volume-1",
			Title: "从请求到处理函数",
			Sort:  1,
			Chapters: []BlueprintChapter{
				{ID: "chapter-routing", Title: "请求如何被路由", Sort: 1, Profile: ChapterProfileConcept, EvidenceIDs: []string{"gin-routergroup-get"}},
				{ID: "chapter-middleware", Title: "中间件链", Sort: 2, Profile: ChapterProfileHandsOn, PrerequisiteIDs: []string{"chapter-routing"}, EvidenceIDs: []string{"gin-routergroup-use"}},
			},
		}},
	}
	require.NoError(t, blueprint.Validate())

	blueprint.Volumes[0].Chapters[0].Profile = ChapterProfileReference
	require.Error(t, blueprint.Validate())
	blueprint.Volumes[0].Chapters[0].Profile = ChapterProfileConcept
	blueprint.Volumes[0].Chapters[0].PrerequisiteIDs = []string{"chapter-middleware"}
	require.Error(t, blueprint.Validate())
}

func TestBlueprintGenerationReadinessRequiresCriticalClaimsBoundToChapterEvidence(t *testing.T) {
	blueprint := Blueprint{
		RevisionID: "blueprint-1", ProjectID: "project-1", RevisionNumber: 1, ContentHash: "sha256:blueprint",
		BookContractRevision: "book-contract-1", StyleSheetRevision: "style-sheet-1",
		Volumes: []BlueprintVolume{{ID: "volume-1", Title: "基础", Sort: 1, Chapters: []BlueprintChapter{{
			ID: "chapter-1", Title: "路由", Sort: 1, Profile: ChapterProfileConcept, EvidenceIDs: []string{"chunk-router"},
		}}}},
	}
	require.NoError(t, blueprint.Validate(), "authors may save an incomplete outline draft")
	require.ErrorContains(t, blueprint.ValidateGenerationReadiness(), "has no critical claim requirements")

	blueprint.Volumes[0].Chapters[0].CriticalClaims = []ClaimRequirement{{ID: "route-registration", Label: "解释路由登记", EvidenceIDs: []string{"chunk-other"}}}
	require.ErrorContains(t, blueprint.ValidateGenerationReadiness(), "references evidence outside the chapter")

	blueprint.Volumes[0].Chapters[0].CriticalClaims[0].EvidenceIDs = []string{"chunk-router"}
	require.NoError(t, blueprint.ValidateGenerationReadiness())
}

func TestScenarioFrameAndUnderstandingChainRejectIncompleteTeachingClaims(t *testing.T) {
	scenario := ScenarioFrame{
		Situation:       "服务收到请求，需要按路径找到处理函数。",
		Trigger:         "没有路由规则时，请求只能得到 404。",
		Consequence:     "读者无法理解处理函数为何没有被调用。",
		Analogy:         "路由表像按地址投递的分拣规则。",
		Mapping:         []AnalogyMapping{{AnalogyElement: "分拣规则", TechnicalElement: "路径与 HTTP 方法匹配"}},
		Mechanism:       "Gin 将方法树中的路径节点与处理函数链关联。",
		Boundary:        "分拣类比不解释参数节点压缩和中间件调用顺序。",
		DemonstrationID: "artifact-router-demo",
	}
	require.NoError(t, scenario.Validate())
	scenario.Boundary = ""
	require.Error(t, scenario.Validate())

	chain := UnderstandingChain{
		Need:         "没有路由时，HTTP 请求无法到达业务代码。",
		Problem:      "同一服务必须按方法和路径分派请求。",
		Alternatives: "可以使用标准库 ServeMux、第三方路由器或手写分派。",
		Usage:        "注册 GET 路由并启动 Engine。",
		Mechanism:    "路由组拼接路径和处理函数链，再交给 Engine 的方法树。",
		Observation:  "通过日志和 200/404 响应观察匹配结果。",
		Tradeoffs:    "路由框架不能替代认证、限流和业务错误处理。",
		AssessmentID: []string{"explain-router", "diagnose-404"},
	}
	require.NoError(t, chain.Validate())
	chain.Observation = ""
	require.Error(t, chain.Validate())
}

func TestLearningArcRequiresOrderedScaffoldingAndRecovery(t *testing.T) {
	arc := DefaultLearningArc()
	for index := range arc.Stages {
		arc.Stages[index].Objective = "完成当前学习阶段"
		arc.Stages[index].SuccessEvidence = []string{"evidence-stage"}
		arc.Stages[index].RecoveryRoute = "返回上一个可验证步骤"
	}
	require.NoError(t, arc.Validate())

	arc.Stages[5], arc.Stages[6] = arc.Stages[6], arc.Stages[5]
	require.Error(t, arc.Validate())
}

func TestRevisionQualityAndPublishingContractsProtectApprovalBoundaries(t *testing.T) {
	revision := ChapterRevision{
		ID:                   "revision-1",
		ChapterID:            "chapter-1",
		Kind:                 RevisionKindCandidate,
		Markdown:             "# Gin 路由\n",
		DocumentHash:         "sha256:document",
		ContentHash:          "sha256:content",
		BookContractRevision: "book-contract-1",
		StyleSheetRevision:   "style-sheet-1",
		EvidencePackHash:     "sha256:evidence",
	}
	require.NoError(t, revision.Validate())
	artifact := CodeArtifact{
		ID:           "artifact-1",
		RevisionID:   revision.ID,
		Kind:         CodeArtifactTeachingImplementation,
		Language:     "go",
		Entrypoint:   "main.go",
		ManifestHash: "sha256:artifact",
		ArtifactHash: "sha256:artifact-tree",
		Limitations:  []string{"只演示单一路由，不代表生产错误处理。"},
		Status:       ArtifactStatusUnverified,
	}
	require.NoError(t, artifact.Validate())

	assessment := QualityAssessment{
		ID:              "assessment-1",
		ScopeType:       QualityScopeChapterRevision,
		ScopeID:         revision.ID,
		ContractVersion: "book-contract-1",
		Detector:        "evidence_gate",
		Status:          QualityStatusPass,
		DimensionScores: []DimensionScore{{Dimension: "evidence", Score: 4}},
	}
	require.NoError(t, assessment.Validate())

	build := BookBuild{
		ID:                     "build-1",
		ProjectID:              "project-1",
		BookContractRevisionID: "book-contract-1",
		StyleSheetRevisionID:   "style-sheet-1",
		ApprovedRevisionIDs:    []string{revision.ID},
		ToolVersions:           map[string]string{"canonical_ast": CanonicalBookASTFormat},
		ManifestHash:           "sha256:manifest",
		Status:                 BookBuildDraft,
	}
	require.NoError(t, build.Validate())

	rights := RightsItem{
		ID:                "rights-1",
		ProjectID:         build.ProjectID,
		BuildID:           build.ID,
		SubjectRef:        "code:artifact-1",
		WorkType:          RightsWorkTypeCode,
		RightsBasis:       "MIT license in primary source snapshot",
		AllowedUse:        "excerpt_with_attribution",
		Attribution:       "gin-gonic/gin v1.12.0",
		PublicationStatus: RightsStatusReady,
	}
	require.NoError(t, rights.Validate())

	build.ManifestHash = ""
	require.Error(t, build.Validate())
	rights.RightsBasis = ""
	require.Error(t, rights.Validate())
	artifact.Limitations = nil
	require.Error(t, artifact.Validate())
}

func TestTextbookEventIsVersionedAndIdempotent(t *testing.T) {
	event := TextbookEvent{
		EventVersion: 1,
		EventID:      "event-1",
		EventType:    EventCandidateRevisionCreated,
		ProjectID:    "project-1",
		Stage:        "chapter_generation",
		InputHash:    "sha256:input",
		OutputHash:   "sha256:output",
	}
	require.NoError(t, event.Validate())
	event.InputHash = ""
	require.Error(t, event.Validate())
}
