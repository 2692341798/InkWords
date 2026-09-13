// Package textbook contains stable, cross-process contracts for the local
// self-study textbook workflow. It deliberately has no transport, persistence,
// queue, filesystem, or model-provider dependencies.
package textbook

import "fmt"

type AudienceLevel string

const (
	AudienceFoundation    AudienceLevel = "foundation"
	AudienceProgramming   AudienceLevel = "programming"
	AudienceStackFamiliar AudienceLevel = "stack_familiar"
)

func (value AudienceLevel) Validate() error {
	return validateEnum(value, []AudienceLevel{AudienceFoundation, AudienceProgramming, AudienceStackFamiliar}, "audience level")
}

type ChapterProfile string

const (
	ChapterProfileConcept           ChapterProfile = "concept"
	ChapterProfileHandsOn           ChapterProfile = "hands_on"
	ChapterProfileSourceWalkthrough ChapterProfile = "source_walkthrough"
	ChapterProfileProjectIteration  ChapterProfile = "project_iteration"
	ChapterProfileTroubleshooting   ChapterProfile = "troubleshooting"
	ChapterProfileIntegrationReview ChapterProfile = "integration_review"
	ChapterProfileReference         ChapterProfile = "reference"
)

func (value ChapterProfile) Validate() error {
	return validateEnum(value, []ChapterProfile{
		ChapterProfileConcept,
		ChapterProfileHandsOn,
		ChapterProfileSourceWalkthrough,
		ChapterProfileProjectIteration,
		ChapterProfileTroubleshooting,
		ChapterProfileIntegrationReview,
		ChapterProfileReference,
	}, "chapter profile")
}

type SourceKind string

const (
	SourceKindGitRepository SourceKind = "git_repository"
	SourceKindOfficialWeb   SourceKind = "official_web"
	SourceKindPDF           SourceKind = "pdf"
	SourceKindDOCX          SourceKind = "docx"
	SourceKindMarkdown      SourceKind = "markdown"
	SourceKindText          SourceKind = "text"
	SourceKindZIP           SourceKind = "zip"
)

func (value SourceKind) Validate() error {
	return validateEnum(value, []SourceKind{
		SourceKindGitRepository,
		SourceKindOfficialWeb,
		SourceKindPDF,
		SourceKindDOCX,
		SourceKindMarkdown,
		SourceKindText,
		SourceKindZIP,
	}, "source kind")
}

type SourceRole string

const (
	SourceRolePrimary       SourceRole = "primary"
	SourceRoleOfficial      SourceRole = "official_supporting"
	SourceRoleUserReference SourceRole = "user_reference"
)

func (value SourceRole) Validate() error {
	return validateEnum(value, []SourceRole{SourceRolePrimary, SourceRoleOfficial, SourceRoleUserReference}, "source role")
}

type EvidenceConfidence string

const (
	EvidenceConfidenceDocumented EvidenceConfidence = "documented"
	EvidenceConfidenceObserved   EvidenceConfidence = "observed"
	EvidenceConfidenceInferred   EvidenceConfidence = "inferred"
)

func (value EvidenceConfidence) Validate() error {
	return validateEnum(value, []EvidenceConfidence{EvidenceConfidenceDocumented, EvidenceConfidenceObserved, EvidenceConfidenceInferred}, "evidence confidence")
}

type ClaimStatus string

const (
	ClaimStatusVerified     ClaimStatus = "verified"
	ClaimStatusUnsupported  ClaimStatus = "unsupported"
	ClaimStatusContradicted ClaimStatus = "contradicted"
)

func (value ClaimStatus) Validate() error {
	return validateEnum(value, []ClaimStatus{ClaimStatusVerified, ClaimStatusUnsupported, ClaimStatusContradicted}, "claim status")
}

type RevisionKind string

const (
	RevisionKindDraft     RevisionKind = "draft"
	RevisionKindCandidate RevisionKind = "candidate"
	RevisionKindApproved  RevisionKind = "approved"
)

func (value RevisionKind) Validate() error {
	return validateEnum(value, []RevisionKind{RevisionKindDraft, RevisionKindCandidate, RevisionKindApproved}, "revision kind")
}

type CodeArtifactKind string

const (
	CodeArtifactTeachingImplementation CodeArtifactKind = "teaching_implementation"
	CodeArtifactIntegrationExample     CodeArtifactKind = "integration_example"
	CodeArtifactUpstreamWalkthrough    CodeArtifactKind = "upstream_walkthrough"
)

func (value CodeArtifactKind) Validate() error {
	return validateEnum(value, []CodeArtifactKind{CodeArtifactTeachingImplementation, CodeArtifactIntegrationExample, CodeArtifactUpstreamWalkthrough}, "code artifact kind")
}

type ArtifactStatus string

const (
	ArtifactStatusDraft      ArtifactStatus = "draft"
	ArtifactStatusUnverified ArtifactStatus = "unverified"
	ArtifactStatusVerified   ArtifactStatus = "verified"
	ArtifactStatusBlocked    ArtifactStatus = "blocked"
)

func (value ArtifactStatus) Validate() error {
	return validateEnum(value, []ArtifactStatus{ArtifactStatusDraft, ArtifactStatusUnverified, ArtifactStatusVerified, ArtifactStatusBlocked}, "artifact status")
}

type QualityScope string

const (
	QualityScopeChapterRevision QualityScope = "chapter_revision"
	QualityScopeBookBuild       QualityScope = "book_build"
	QualityScopeStyleProbe      QualityScope = "style_probe"
)

func (value QualityScope) Validate() error {
	return validateEnum(value, []QualityScope{QualityScopeChapterRevision, QualityScopeBookBuild, QualityScopeStyleProbe}, "quality scope")
}

type QualityStatus string

const (
	QualityStatusPass     QualityStatus = "pass"
	QualityStatusSoftFail QualityStatus = "soft_fail"
	QualityStatusHardFail QualityStatus = "hard_fail"
)

func (value QualityStatus) Validate() error {
	return validateEnum(value, []QualityStatus{QualityStatusPass, QualityStatusSoftFail, QualityStatusHardFail}, "quality status")
}

type RightsWorkType string

const (
	RightsWorkTypeProse      RightsWorkType = "prose"
	RightsWorkTypeCode       RightsWorkType = "code"
	RightsWorkTypeImage      RightsWorkType = "image"
	RightsWorkTypeScreenshot RightsWorkType = "screenshot"
	RightsWorkTypeFont       RightsWorkType = "font"
	RightsWorkTypeTrademark  RightsWorkType = "trademark"
	RightsWorkTypeData       RightsWorkType = "data"
)

func (value RightsWorkType) Validate() error {
	return validateEnum(value, []RightsWorkType{
		RightsWorkTypeProse,
		RightsWorkTypeCode,
		RightsWorkTypeImage,
		RightsWorkTypeScreenshot,
		RightsWorkTypeFont,
		RightsWorkTypeTrademark,
		RightsWorkTypeData,
	}, "rights work type")
}

type RightsStatus string

const (
	RightsStatusPending RightsStatus = "pending"
	RightsStatusReady   RightsStatus = "ready"
	RightsStatusBlocked RightsStatus = "blocked"
)

func (value RightsStatus) Validate() error {
	return validateEnum(value, []RightsStatus{RightsStatusPending, RightsStatusReady, RightsStatusBlocked}, "rights status")
}

type BookBuildStatus string

const (
	BookBuildDraft                BookBuildStatus = "draft"
	BookBuildReadyForReview       BookBuildStatus = "ready_for_review"
	BookBuildPublicationCandidate BookBuildStatus = "publication_candidate"
	BookBuildBlocked              BookBuildStatus = "blocked"
)

func (value BookBuildStatus) Validate() error {
	return validateEnum(value, []BookBuildStatus{BookBuildDraft, BookBuildReadyForReview, BookBuildPublicationCandidate, BookBuildBlocked}, "book build status")
}

type LearningStage string

const (
	LearningStageActivatePriorKnowledge     LearningStage = "activate_prior_knowledge"
	LearningStageExperienceProblem          LearningStage = "experience_problem"
	LearningStagePreviewWhole               LearningStage = "preview_whole"
	LearningStageWorkedExample              LearningStage = "worked_example"
	LearningStageGuidedPractice             LearningStage = "guided_practice"
	LearningStageFadeScaffolding            LearningStage = "fade_scaffolding"
	LearningStageIndependentTransfer        LearningStage = "independent_transfer"
	LearningStageExplainAndConnect          LearningStage = "explain_and_connect"
	LearningStageSpacedInterleavedRetrieval LearningStage = "spaced_interleaved_retrieval"
	LearningStageAdaptFromEvidence          LearningStage = "adapt_from_evidence"
)

func (value LearningStage) Validate() error {
	return validateEnum(value, defaultLearningStageOrder(), "learning stage")
}

type LearningTaskMode string

const (
	LearningTaskExplain   LearningTaskMode = "explain"
	LearningTaskComplete  LearningTaskMode = "complete"
	LearningTaskReproduce LearningTaskMode = "reproduce"
	LearningTaskTransfer  LearningTaskMode = "transfer"
	LearningTaskDiagnose  LearningTaskMode = "diagnose"
	LearningTaskRetain    LearningTaskMode = "retain"
)

func (value LearningTaskMode) Validate() error {
	return validateEnum(value, []LearningTaskMode{LearningTaskExplain, LearningTaskComplete, LearningTaskReproduce, LearningTaskTransfer, LearningTaskDiagnose, LearningTaskRetain}, "learning task mode")
}

type EventType string

const (
	EventCandidateRevisionCreated EventType = "candidate_revision_created"
	EventCandidateRevisionApplied EventType = "candidate_revision_applied"
	EventQualityAssessmentCreated EventType = "quality_assessment_created"
	EventVerificationCompleted    EventType = "verification_completed"
	EventBookBuildCreated         EventType = "book_build_created"
)

func (value EventType) Validate() error {
	return validateEnum(value, []EventType{
		EventCandidateRevisionCreated,
		EventCandidateRevisionApplied,
		EventQualityAssessmentCreated,
		EventVerificationCompleted,
		EventBookBuildCreated,
	}, "textbook event type")
}

func validateEnum[T comparable](value T, allowed []T, name string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("unknown %s %v", name, value)
}

func defaultLearningStageOrder() []LearningStage {
	return []LearningStage{
		LearningStageActivatePriorKnowledge,
		LearningStageExperienceProblem,
		LearningStagePreviewWhole,
		LearningStageWorkedExample,
		LearningStageGuidedPractice,
		LearningStageFadeScaffolding,
		LearningStageIndependentTransfer,
		LearningStageExplainAndConnect,
		LearningStageSpacedInterleavedRetrieval,
		LearningStageAdaptFromEvidence,
	}
}
