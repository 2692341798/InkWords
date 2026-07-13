package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	coretask "inkwords-backend/services/core-api/domain/task"
	verification "inkwords-backend/services/course-runner/domain/verification"
	"inkwords-backend/shared/kernel/httpx"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
	"inkwords-backend/shared/platform/postgres"
)

type taskStore struct{ repo *coretask.GormRepository }

func (s taskStore) MarkRunning(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, coretask.JobTaskStatusRunning, "")
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

type artifactResolver struct{ root string }

func (r artifactResolver) Resolve(_ context.Context, payload verification.VerificationPayload) (verification.RunRequest, error) {
	if strings.TrimSpace(r.root) == "" || strings.TrimSpace(payload.ArtifactToken) == "" || filepath.Base(payload.ArtifactToken) != payload.ArtifactToken {
		return verification.RunRequest{}, errors.New("invalid artifact token")
	}
	root, err := filepath.Abs(r.root)
	if err != nil {
		return verification.RunRequest{}, err
	}
	artifactPath := filepath.Join(root, payload.ArtifactToken)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return verification.RunRequest{}, fmt.Errorf("resolve artifact root: %w", err)
	}
	resolvedArtifact, err := filepath.EvalSymlinks(artifactPath)
	if err != nil {
		return verification.RunRequest{}, fmt.Errorf("resolve artifact: %w", err)
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedArtifact)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return verification.RunRequest{}, errors.New("artifact path escapes artifact root")
	}
	info, err := os.Stat(resolvedArtifact)
	if err != nil || !info.IsDir() {
		return verification.RunRequest{}, errors.New("course artifact directory is unavailable")
	}
	manifestBytes, err := os.ReadFile(filepath.Join(resolvedArtifact, "manifest.json"))
	if err != nil {
		return verification.RunRequest{}, fmt.Errorf("read course manifest: %w", err)
	}
	var wrapper struct {
		CourseID         string                   `json:"course_id"`
		BlueprintVersion int                      `json:"blueprint_version"`
		Manifest         sharedkernel.LabManifest `json:"manifest"`
	}
	if err := json.Unmarshal(manifestBytes, &wrapper); err != nil {
		return verification.RunRequest{}, fmt.Errorf("decode course manifest: %w", err)
	}
	if wrapper.CourseID != "" && wrapper.CourseID != payload.CourseID.String() {
		return verification.RunRequest{}, errors.New("artifact course ID does not match task")
	}
	if wrapper.BlueprintVersion != 0 && wrapper.BlueprintVersion != payload.BlueprintVersion {
		return verification.RunRequest{}, errors.New("artifact blueprint version does not match task")
	}
	if err := wrapper.Manifest.Validate(); err != nil {
		return verification.RunRequest{}, fmt.Errorf("invalid course manifest: %w", err)
	}
	return verification.RunRequest{Manifest: wrapper.Manifest, RootDir: resolvedArtifact}, nil
}

// BuildRouter intentionally wires no host executor. The default Runner fails
// closed until a separately deployed sandbox implementation is configured.
func BuildRouter() (*gin.Engine, *verification.Consumer, error) {
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
	tasks := taskStore{repo: coretask.NewGormRepository(db)}
	resolver := artifactResolver{root: envOrDefault("COURSE_ARTIFACTS_DIR", "/app/course-artifacts")}
	consumer := verification.NewConsumer(tasks, resolver, verification.Runner{})
	return r, consumer, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
