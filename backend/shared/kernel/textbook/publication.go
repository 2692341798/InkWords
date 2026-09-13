package textbook

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RightsItem keeps publication eligibility explicit for every reusable work.
type RightsItem struct {
	ID                string         `json:"id"`
	ProjectID         string         `json:"project_id"`
	BuildID           string         `json:"build_id"`
	SubjectRef        string         `json:"subject_ref"`
	WorkType          RightsWorkType `json:"work_type"`
	RightsBasis       string         `json:"rights_basis"`
	AllowedUse        string         `json:"allowed_use"`
	Attribution       string         `json:"attribution"`
	PublicationStatus RightsStatus   `json:"publication_status"`
}

func (item RightsItem) Validate() error {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.ProjectID) == "" || strings.TrimSpace(item.BuildID) == "" || strings.TrimSpace(item.SubjectRef) == "" || strings.TrimSpace(item.RightsBasis) == "" || strings.TrimSpace(item.AllowedUse) == "" || strings.TrimSpace(item.Attribution) == "" {
		return fmt.Errorf("rights item is incomplete")
	}
	if err := item.WorkType.Validate(); err != nil {
		return err
	}
	return item.PublicationStatus.Validate()
}

// RightsSubject is one exact object embedded in a frozen build that must have
// a matching immutable RightsItem before publication preflight may pass.
type RightsSubject struct {
	SubjectRef string         `json:"subject_ref"`
	WorkType   RightsWorkType `json:"work_type"`
}

func (subject RightsSubject) Validate() error {
	if strings.TrimSpace(subject.SubjectRef) == "" {
		return fmt.Errorf("rights subject reference is required")
	}
	return subject.WorkType.Validate()
}

// PublicationReviewStage names one of the eight editorial gates. Actor
// provenance remains explicit in separate human and delegated AI contracts.
// Automated detectors use AutomatedPublicationCheck and cannot satisfy these stages.
type PublicationReviewStage string

const (
	PublicationReviewDevelopmental PublicationReviewStage = "developmental"
	PublicationReviewTechnical     PublicationReviewStage = "technical"
	PublicationReviewSelfStudy     PublicationReviewStage = "self_study"
	PublicationReviewConsistency   PublicationReviewStage = "consistency"
	PublicationReviewCopyEditing   PublicationReviewStage = "copy_editing"
	PublicationReviewLayout        PublicationReviewStage = "layout"
	PublicationReviewRights        PublicationReviewStage = "rights"
	PublicationReviewReaderTrial   PublicationReviewStage = "reader_trial"
)

var requiredPublicationReviewStages = []PublicationReviewStage{
	PublicationReviewDevelopmental,
	PublicationReviewTechnical,
	PublicationReviewSelfStudy,
	PublicationReviewConsistency,
	PublicationReviewCopyEditing,
	PublicationReviewLayout,
	PublicationReviewRights,
	PublicationReviewReaderTrial,
}

func (stage PublicationReviewStage) Validate() error {
	return validateEnum(stage, requiredPublicationReviewStages, "publication review stage")
}

func (stage PublicationReviewStage) Label() string {
	switch stage {
	case PublicationReviewDevelopmental:
		return "发展性"
	case PublicationReviewTechnical:
		return "技术"
	case PublicationReviewSelfStudy:
		return "自学性"
	case PublicationReviewConsistency:
		return "全书一致性"
	case PublicationReviewCopyEditing:
		return "文字"
	case PublicationReviewLayout:
		return "版式"
	case PublicationReviewRights:
		return "权利与合规"
	case PublicationReviewReaderTrial:
		return "读者试学"
	default:
		return "未知"
	}
}

// RequiredPublicationReviewStages returns a copy so callers cannot mutate the
// publication contract's required order.
func RequiredPublicationReviewStages() []PublicationReviewStage {
	return append([]PublicationReviewStage(nil), requiredPublicationReviewStages...)
}

// HumanPublicationReview is immutable evidence that a person completed one
// review stage for one frozen build.
type HumanPublicationReview struct {
	ContractVersion string                 `json:"contract_version,omitempty"`
	ManifestHash    string                 `json:"manifest_hash,omitempty"`
	Revision        int                    `json:"revision,omitempty"`
	ReviewerKind    string                 `json:"reviewer_kind,omitempty"`
	Verdict         string                 `json:"verdict,omitempty"`
	Score           int                    `json:"score,omitempty"`
	Scope           string                 `json:"scope,omitempty"`
	EvidenceRefs    []string               `json:"evidence_refs,omitempty"`
	HardFailures    []string               `json:"hard_failures,omitempty"`
	ID              string                 `json:"id"`
	BuildID         string                 `json:"build_id"`
	Stage           PublicationReviewStage `json:"stage"`
	Reviewer        string                 `json:"reviewer"`
	Notes           string                 `json:"notes"`
	CompletedAt     time.Time              `json:"completed_at"`
	Automated       bool                   `json:"automated"`
}

func (review HumanPublicationReview) Validate() error {
	if strings.TrimSpace(review.ID) == "" || strings.TrimSpace(review.BuildID) == "" || strings.TrimSpace(review.Reviewer) == "" || len([]rune(strings.TrimSpace(review.Notes))) < 8 || review.CompletedAt.IsZero() || review.Automated {
		return fmt.Errorf("human publication review is incomplete")
	}
	if err := review.Stage.Validate(); err != nil {
		return err
	}
	return review.validateDecision()
}

// AutomatedPublicationCheck always names its detector. It is intentionally a
// separate type and cannot be imported as a human review.
type AutomatedPublicationCheck struct {
	ID       string        `json:"id"`
	Detector string        `json:"detector"`
	Status   QualityStatus `json:"status"`
}

func (check AutomatedPublicationCheck) Validate() error {
	if strings.TrimSpace(check.ID) == "" || strings.TrimSpace(check.Detector) == "" {
		return fmt.Errorf("automated publication check is incomplete")
	}
	return check.Status.Validate()
}

type PublicationPreflightInput struct {
	ManifestHash           string                       `json:"manifest_hash,omitempty"`
	DelegatedReviews       []DelegatedPublicationReview `json:"delegated_reviews,omitempty"`
	BuildID                string                       `json:"build_id"`
	RequiredRightsSubjects []RightsSubject              `json:"required_rights_subjects"`
	RightsItems            []RightsItem                 `json:"rights_items"`
	RightsAmendments       []RightsAmendment            `json:"rights_amendments,omitempty"`
	AutomatedChecks        []AutomatedPublicationCheck  `json:"automated_checks"`
	HumanReviews           []HumanPublicationReview     `json:"human_reviews"`
}

type PublicationPreflightResult struct {
	Passed   bool     `json:"passed"`
	Blockers []string `json:"blockers"`
}

// EvaluatePublicationPreflight is fail-closed. It only evaluates explicit,
// persisted evidence and never changes BookBuild status by itself.
func EvaluatePublicationPreflight(input PublicationPreflightInput) PublicationPreflightResult {
	blockers := make([]string, 0)
	buildID := strings.TrimSpace(input.BuildID)
	ledger, rightsErr := ResolveRightsLedger(buildID, input.ManifestHash, input.RightsItems, input.RightsAmendments)
	if rightsErr != nil {
		blockers = append(blockers, "权利补证历史不完整或对象冲突。")
		input.RightsItems = nil
	} else {
		input.RightsItems = ledger.EffectiveItems
	}
	if buildID == "" {
		blockers = append(blockers, "缺少冻结构建标识。")
	}
	if len(input.RightsItems) == 0 {
		blockers = append(blockers, "缺少权利清单。")
	}
	if len(input.RequiredRightsSubjects) == 0 {
		blockers = append(blockers, "冻结构建没有可核对的逐对象权利范围。")
	}
	coveredRights := make(map[string]RightsWorkType, len(input.RightsItems))
	for _, item := range input.RightsItems {
		if err := item.Validate(); err != nil || item.BuildID != buildID {
			blockers = append(blockers, "权利项 "+item.ID+" 不完整或不属于当前构建。")
			continue
		}
		if item.PublicationStatus != RightsStatusReady {
			blockers = append(blockers, "权利项 "+item.ID+" 尚未处于 ready 状态。")
			continue
		}
		coveredRights[item.SubjectRef] = item.WorkType
	}
	for _, subject := range input.RequiredRightsSubjects {
		if err := subject.Validate(); err != nil {
			blockers = append(blockers, "冻结构建包含无效的权利核对对象。")
			continue
		}
		if workType, ok := coveredRights[subject.SubjectRef]; !ok || workType != subject.WorkType {
			blockers = append(blockers, fmt.Sprintf("权利清单尚未覆盖 %s（%s）。", subject.SubjectRef, subject.WorkType))
		}
	}
	if len(input.AutomatedChecks) == 0 {
		blockers = append(blockers, "缺少自动出版完整性检查。")
	}
	for _, check := range input.AutomatedChecks {
		if err := check.Validate(); err != nil {
			blockers = append(blockers, "自动检查记录不完整。")
			continue
		}
		if check.Status != QualityStatusPass {
			blockers = append(blockers, fmt.Sprintf("自动检查 %s 未通过（检测器：%s）。", check.ID, check.Detector))
		}
	}
	completed, humanBlockers := evaluateHumanPublicationReviews(input)
	blockers = append(blockers, humanBlockers...)
	blockers = applyDelegatedPublicationReviews(input, completed, blockers)
	// A rights decision made before new evidence cannot attest that evidence.
	var amendedAt, reviewedAt time.Time
	for _, a := range input.RightsAmendments {
		if a.CompletedAt.After(amendedAt) {
			amendedAt = a.CompletedAt
		}
	}
	for _, review := range input.HumanReviews {
		if review.ContractVersion == HumanPublicationReviewFormat && review.Validate() == nil && review.BuildID == buildID && review.ManifestHash == input.ManifestHash && review.Stage == PublicationReviewRights && review.CompletedAt.After(reviewedAt) {
			reviewedAt = review.CompletedAt
		}
	}
	for _, review := range input.DelegatedReviews {
		if review.Validate() == nil && review.BuildID == buildID && review.ManifestHash == input.ManifestHash && review.Stage == PublicationReviewRights && review.CompletedAt.After(reviewedAt) {
			reviewedAt = review.CompletedAt
		}
	}
	if amendedAt.After(reviewedAt) {
		completed[PublicationReviewRights] = false
		blockers = append(blockers, "权利证据已补证，请重新审阅当前权利清单。")
	}
	for _, stage := range requiredPublicationReviewStages {
		if passed, recorded := completed[stage]; !passed && recorded {
			blockers = append(blockers, stage.Label()+"审校已有记录，但尚未通过。")
		} else if !passed {
			blockers = append(blockers, "缺少人工或用户委托 AI "+stage.Label()+"审校记录。")
		}
	}
	sort.Strings(blockers)
	return PublicationPreflightResult{Passed: len(blockers) == 0, Blockers: blockers}
}

// BookBuild pins exactly the approved manuscript revisions used by every export.
type BookBuild struct {
	ID                     string            `json:"id"`
	ProjectID              string            `json:"project_id"`
	BookContractRevisionID string            `json:"book_contract_revision_id"`
	StyleSheetRevisionID   string            `json:"style_sheet_revision_id"`
	ApprovedRevisionIDs    []string          `json:"approved_revision_ids"`
	ToolVersions           map[string]string `json:"tool_versions"`
	ManifestHash           string            `json:"manifest_hash"`
	Status                 BookBuildStatus   `json:"status"`
}

func (build BookBuild) Validate() error {
	if strings.TrimSpace(build.ID) == "" || strings.TrimSpace(build.ProjectID) == "" || strings.TrimSpace(build.BookContractRevisionID) == "" || strings.TrimSpace(build.StyleSheetRevisionID) == "" || len(build.ApprovedRevisionIDs) == 0 || len(build.ToolVersions) == 0 {
		return fmt.Errorf("book build identity and approved revisions are required")
	}
	if !isSHA256Digest(build.ManifestHash) {
		return fmt.Errorf("book build manifest hash is required")
	}
	return build.Status.Validate()
}
