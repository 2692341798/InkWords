package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	learnerverification "inkwords-backend/services/course-runner/domain/learnerverification"
	textbookverification "inkwords-backend/services/course-runner/domain/textbookverification"
	verification "inkwords-backend/services/course-runner/domain/verification"
	learnerinfra "inkwords-backend/services/course-runner/infra/learnerverification"
	courseroutes "inkwords-backend/services/course-runner/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/postgres"
	"inkwords-backend/shared/platform/teachingartifact"
	"inkwords-backend/shared/platform/visualasset"
)

type taskStore struct{ repo *coretask.GormRepository }

func (s taskStore) ClaimVerificationWorker(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	token, err := s.repo.ClaimVerificationWorker(ctx, id)
	if errors.Is(err, coretask.ErrVerificationTaskTerminal) || errors.Is(err, coretask.ErrVerificationExecutionActive) {
		return uuid.Nil, textbookverification.ErrVerificationTaskTerminal
	}
	return token, err
}
func (s taskStore) ReleaseVerificationWorker(ctx context.Context, id, token uuid.UUID) error {
	return s.repo.ReleaseVerificationWorker(ctx, id, token)
}
func (s taskStore) MarkSucceeded(ctx context.Context, id uuid.UUID, result []byte) error {
	if err := s.repo.UpdateResult(ctx, id, result); err != nil {
		return err
	}
	return s.repo.UpdateStatus(ctx, id, coretask.JobTaskStatusSucceeded, "")
}
func (s taskStore) MarkFailed(ctx context.Context, id uuid.UUID, message string) error {
	return s.repo.UpdateStatus(ctx, id, coretask.JobTaskStatusFailed, message)
}
func (s taskStore) IsCancelled(ctx context.Context, id uuid.UUID) (bool, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return task.Status == coretask.JobTaskStatusCancelled, nil
}
func (s taskStore) TextbookWorkspaceMatches(ctx context.Context, taskID, workspaceID uuid.UUID, taskSubtype string) (bool, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return false, err
	}
	return task.WorkspaceID != nil && *task.WorkspaceID == workspaceID && task.TaskSubtype == taskSubtype, nil
}

// BuildRouter intentionally wires no host executor. The default Runner fails
// closed until a separately deployed sandbox implementation is configured.
func BuildRouter() (*gin.Engine, *textbookverification.Consumer, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil, errors.New("DATABASE_URL environment variable is not set")
	}
	db, err := postgres.InitCore(dsn)
	if err != nil {
		return nil, nil, err
	}
	r := gin.New()
	r.Use(gin.Recovery(), httpx.RequestID(), httpx.RequestLogger("course-runner"))
	httpx.RegisterHealthRoutes(r, httpx.NewHealthAPI("course-runner", map[string]httpx.ReadinessCheck{"db": httpx.NewGormReadinessCheck(db)}))
	sandboxProfilePath := envOrDefault("TEXTBOOK_RUNNER_SANDBOX_PROFILE_PATH", "/app/seccomp/bubblewrap-outer.json")
	learnerCapability, learnerExecutor := configureLearnerRuntime(os.Getenv("LEARNER_ARTIFACT_VERIFICATION_ENABLED") == "true", os.Getenv("TEXTBOOK_RUNNER_IMAGE_DIGEST"), sandboxProfilePath)
	registerLearnerCapabilityRoute(r, learnerCapability)
	learnerResolver, err := learnerinfra.NewClient(envOrDefault("REVIEW_SERVICE_URL", "http://review-service:8080"))
	if err != nil {
		return nil, nil, err
	}
	learnerRunner := learnerverification.Runner{Resolver: learnerResolver, Stager: learnerinfra.Stager{}, Identity: learnerIdentity(learnerCapability)}
	if learnerExecutor != nil {
		learnerRunner.Executor = learnerExecutor
	}
	courseroutes.RegisterLearnerVerificationRoutes(r, learnerRunner)
	tasks := taskStore{repo: coretask.NewGormRepository(db)}
	textbookRunner := textbookverification.Runner{RunnerImageDigest: strings.TrimSpace(os.Getenv("TEXTBOOK_RUNNER_IMAGE_DIGEST")), ToolchainVersion: runtime.Version()}
	if os.Getenv("TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED") == "true" {
		binary, err := exec.LookPath("bwrap")
		if err != nil {
			return nil, nil, fmt.Errorf("textbook verification enabled but bwrap is unavailable: %w", err)
		}
		executor, err := verifiedBubblewrapExecutor(binary, sandboxProfilePath)
		if err != nil {
			return nil, nil, fmt.Errorf("textbook verification enabled but isolation is unavailable: %w", err)
		}
		textbookRunner.Executor = executor
		browserExecutor, err := configuredPlaywrightBrowserExecutor(executor)
		if err != nil {
			log.Printf("textbook browser-page verification disabled after preflight: %v", err)
		} else {
			textbookRunner.BrowserExecutor = browserExecutor
		}
	}
	textbookConsumer := textbookverification.NewConsumer(tasks, textbookArtifactResolver{db: db, store: teachingartifact.NewStore(envOrDefault("TEXTBOOK_TEACHING_ARTIFACTS_DIR", "/app/teaching-artifacts"))}, textbookRunner, textbookEvidenceStore{db: db, visual: visualasset.NewStore(envOrDefault("TEXTBOOK_VISUAL_ASSETS_DIR", "/app/visual-assets")).WithReadGroup(teachingartifact.ReaderGroupID)})
	return r, textbookConsumer, nil
}

func learnerIdentity(capability sharedtextbook.LearnerVerificationCapability) sharedtextbook.LearnerRunnerIdentity {
	if capability.Runner == nil {
		return sharedtextbook.LearnerRunnerIdentity{}
	}
	return *capability.Runner
}

func configuredPlaywrightBrowserExecutor(sandbox verification.BubblewrapExecutor) (textbookverification.BrowserPageExecutor, error) {
	nodeBinary, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node is unavailable: %w", err)
	}
	config := verification.BrowserProbeConfig{
		NodeBinary:       nodeBinary,
		ProbeScript:      "/app/playwright-probe/probe.mjs",
		ModuleDirectory:  "/app/playwright-probe/node_modules",
		BrowserDirectory: "/ms-playwright",
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := sandbox.PreflightBrowser(ctx, config); err != nil {
		return nil, fmt.Errorf("sandboxed Chromium preflight failed: %w", err)
	}
	return playwrightBrowserExecutor{sandbox: sandbox, config: config}, nil
}

func verifiedBubblewrapExecutor(binary, sandboxProfilePath string) (verification.BubblewrapExecutor, error) {
	digest, err := fileSHA256Digest(sandboxProfilePath)
	if err != nil || digest != sharedtextbook.LearnerSandboxProfileDigest {
		return verification.BubblewrapExecutor{}, fmt.Errorf("reviewed sandbox profile is unavailable")
	}
	executor := verification.BubblewrapExecutor{Binary: binary}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if err := executor.Preflight(ctx); err != nil {
		return verification.BubblewrapExecutor{}, err
	}
	if err := executor.PreflightTeachingGoTest(ctx); err != nil {
		return verification.BubblewrapExecutor{}, fmt.Errorf("fixed teaching Go preflight failed: %w", err)
	}
	return executor, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
