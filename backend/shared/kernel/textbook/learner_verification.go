package textbook

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// LearnerVerificationPlanFormat identifies an execution-input proposal. A valid
// plan is neither execution permission nor a successful VerificationRun.
const LearnerVerificationPlanFormat = "inkwords.learner-verification-plan.v1"

// LearnerGoTestProfile names the accepted fixed offline Go-test policy. A live
// executor may accept it only after its fail-closed isolation preflight passes.
const LearnerGoTestProfile = "inkwords.learner-go-test-offline.v2"

// LearnerSandboxProfileDigest binds evidence to the reviewed outer Docker
// seccomp profile. The inner filter and prlimit implementation remain covered
// by the runner image digest.
const LearnerSandboxProfileDigest = "sha256:f6ed706bb4af6e5b602f12d86ceaf254a365ab2f0f7cabf0a9f56243044683fa"

var learnerGoVersion = regexp.MustCompile(`^go[1-9][0-9]*\.[0-9]+\.[0-9]+$`)

// LearnerRunnerIdentity must come from trusted runtime configuration and tool
// inspection, not from a submitted file, browser field or model response.
type LearnerRunnerIdentity struct {
	ImageDigest          string `json:"image_digest"`
	ToolchainVersion     string `json:"toolchain_version"`
	SandboxProfileDigest string `json:"sandbox_profile_digest"`
}

type LearnerExecutionPolicy struct {
	Command             []string `json:"command"`
	Environment         []string `json:"environment"`
	Network             string   `json:"network"`
	UID                 int      `json:"uid"`
	GID                 int      `json:"gid"`
	MemoryBytes         int64    `json:"memory_bytes"`
	PIDs                int      `json:"pids"`
	FileBytes           int64    `json:"file_bytes"`
	OutputBytes         int      `json:"output_bytes"`
	CPUSeconds          int      `json:"cpu_seconds"`
	TimeoutMillis       int      `json:"timeout_millis"`
	MissingModulePolicy string   `json:"missing_module_policy"`
}

func DefaultLearnerGoTestPolicy() LearnerExecutionPolicy {
	return LearnerExecutionPolicy{
		Command:     []string{"go", "test", "-count=1", "-mod=readonly", "-trimpath", "./..."},
		Environment: []string{"CGO_ENABLED=0", "GOCACHE=/tmp/go-cache", "GOFLAGS=-p=1", "GOMAXPROCS=1", "GOMODCACHE=/tmp/gomod-cache", "GOPATH=/tmp/go-path", "GOPROXY=off", "GOROOT=/usr/local/go", "GOSUMDB=off", "GOTELEMETRY=off", "GO_TELEMETRY_CHILD=2", "GOTOOLCHAIN=local", "GOWORK=off", "HOME=/tmp", "PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"},
		Network:     "unshared", UID: 65534, GID: 65534, MemoryBytes: 402653184, PIDs: 64, FileBytes: 67108864, OutputBytes: MaxLearnerVerificationOutputBytes, CPUSeconds: 30, TimeoutMillis: 30000, MissingModulePolicy: "inject-inkwords-local-module-v1",
	}
}

func (identity LearnerRunnerIdentity) Validate() error {
	if !isFullSHA256Digest(identity.ImageDigest) || !learnerGoVersion.MatchString(identity.ToolchainVersion) || identity.SandboxProfileDigest != LearnerSandboxProfileDigest {
		return fmt.Errorf("learner runner identity is invalid")
	}
	return nil
}

// LearnerVerificationPlan binds a saved learner answer and exact approved task
// to one proposed execution environment. It carries no paths, shell strings,
// credentials, runtime results or manuscript CodeArtifact identity.
type LearnerVerificationPlan struct {
	Format              string                 `json:"format"`
	Profile             string                 `json:"profile"`
	WorkspaceID         string                 `json:"workspace_id"`
	ObjectiveID         string                 `json:"objective_id"`
	AttemptID           string                 `json:"attempt_id"`
	SessionID           string                 `json:"session_id"`
	ChapterID           string                 `json:"chapter_id"`
	RevisionID          string                 `json:"revision_id"`
	TaskID              string                 `json:"task_id"`
	Skill               LearningTaskMode       `json:"skill"`
	PracticeContentHash string                 `json:"practice_content_hash"`
	SnapshotHash        string                 `json:"snapshot_hash"`
	FilesHash           string                 `json:"files_hash"`
	DerivedFiles        []LearnerCodeFile      `json:"derived_files,omitempty"`
	ExecutionFilesHash  string                 `json:"execution_files_hash"`
	TaskHash            string                 `json:"task_hash"`
	Runner              LearnerRunnerIdentity  `json:"runner"`
	Policy              LearnerExecutionPolicy `json:"policy"`
	InputHash           string                 `json:"input_hash,omitempty"`
}

// NewLearnerVerificationPlan performs pure identity checks. The caller must
// resolve both arguments from their owners; hashes alone never prove approval.
// It never opens files, executes commands, stages code or contacts a service.
func NewLearnerVerificationPlan(artifact LearnerArtifact, approved LearningProjection, runner LearnerRunnerIdentity) (LearnerVerificationPlan, error) {
	var plan LearnerVerificationPlan
	if artifact.Validate() != nil || approved.Validate() != nil || approved.Format != "inkwords.learning-projection.v2" || approved.PracticeSet == nil || artifact.RevisionID != approved.RevisionID || artifact.PracticeContentHash != approved.ContentHash {
		return plan, fmt.Errorf("learner verification requires matching frozen practice")
	}
	chapter, err := uuid.Parse(approved.ChapterID)
	if err != nil || chapter == uuid.Nil || chapter.String() != approved.ChapterID || runner.Validate() != nil {
		return plan, fmt.Errorf("learner verification requires a fixed chapter and runner identity")
	}
	var task *PracticeTask
	for i := range approved.PracticeSet.Tasks {
		candidate := &approved.PracticeSet.Tasks[i]
		if candidate.ID == artifact.TaskID && candidate.Mode == artifact.Skill {
			task = candidate
			break
		}
	}
	if task == nil {
		return plan, fmt.Errorf("learner verification task is not in the frozen practice")
	}
	return newLearnerVerificationPlanForTask(artifact, approved.ChapterID, *task, runner)
}

func newLearnerVerificationPlanForTask(artifact LearnerArtifact, chapterID string, task PracticeTask, runner LearnerRunnerIdentity) (LearnerVerificationPlan, error) {
	var plan LearnerVerificationPlan
	if err := artifact.Validate(); err != nil || task.ID != artifact.TaskID || task.Mode != artifact.Skill {
		return plan, fmt.Errorf("learner verification requires matching frozen task and files")
	}
	chapter, err := uuid.Parse(chapterID)
	if err != nil || chapter == uuid.Nil || chapter.String() != chapterID || runner.Validate() != nil {
		return plan, fmt.Errorf("learner verification requires a fixed chapter and runner identity")
	}
	filesHash, err := LearnerFilesHash(artifact.Files)
	if err != nil {
		return plan, err
	}
	derivedFiles := learnerDerivedFiles(artifact.Files, runner.ToolchainVersion)
	executionFiles := append(append([]LearnerCodeFile(nil), artifact.Files...), derivedFiles...)
	executionFilesHash, err := LearnerFilesHash(executionFiles)
	if err != nil {
		return plan, err
	}
	taskBytes, err := json.Marshal(task)
	if err != nil {
		return plan, err
	}
	plan = LearnerVerificationPlan{Format: LearnerVerificationPlanFormat, Profile: LearnerGoTestProfile, WorkspaceID: artifact.WorkspaceID, ObjectiveID: artifact.ObjectiveID, AttemptID: artifact.AttemptID, SessionID: artifact.SessionID, ChapterID: chapterID, RevisionID: artifact.RevisionID, TaskID: artifact.TaskID, Skill: artifact.Skill, PracticeContentHash: artifact.PracticeContentHash, SnapshotHash: artifact.SnapshotHash, FilesHash: filesHash, DerivedFiles: derivedFiles, ExecutionFilesHash: executionFilesHash, TaskHash: PracticeExcerptHash(string(taskBytes)), Runner: runner, Policy: DefaultLearnerGoTestPolicy()}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return LearnerVerificationPlan{}, err
	}
	plan.InputHash = PracticeExcerptHash(string(encoded))
	return plan, nil
}

func learnerDerivedFiles(files []LearnerCodeFile, toolchainVersion string) []LearnerCodeFile {
	for _, file := range files {
		if file.Path == "go.mod" {
			return nil
		}
	}
	version := strings.TrimPrefix(toolchainVersion, "go")
	parts := strings.Split(version, ".")
	return []LearnerCodeFile{{Path: "go.mod", Content: "module inkwords.local/learner\n\ngo " + strings.Join(parts[:2], ".") + "\n"}}
}

// ValidateForTask lets the execution service recheck owner-resolved task data
// without accepting a browser-supplied projection or trusting a hash alone.
func (plan LearnerVerificationPlan) ValidateForTask(artifact LearnerArtifact, chapterID string, task PracticeTask) error {
	expected, err := newLearnerVerificationPlanForTask(artifact, chapterID, task, plan.Runner)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(plan, expected) {
		return fmt.Errorf("learner verification plan does not match its frozen input")
	}
	return nil
}

// ValidateFor reconstructs the plan from owner-resolved data, rejecting changed
// identity, bytes, rubric or environment. It does not enable an executor.
func (plan LearnerVerificationPlan) ValidateFor(artifact LearnerArtifact, approved LearningProjection) error {
	expected, err := NewLearnerVerificationPlan(artifact, approved, plan.Runner)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(plan, expected) {
		return fmt.Errorf("learner verification plan does not match its frozen input")
	}
	return nil
}
