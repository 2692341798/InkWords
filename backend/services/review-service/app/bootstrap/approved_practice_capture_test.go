package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	app "inkwords-backend/services/review-service/app/masteryassessment"
	"inkwords-backend/services/review-service/domain/mastery"
	persistence "inkwords-backend/services/review-service/infra/assessment"
	"inkwords-backend/shared/kernel/generation"
	platformllm "inkwords-backend/shared/platform/llm"
)

// Restore only an explicitly selected operator-acceptance dump into the new
// disposable database. This is never a production database recovery path.
func restoreAcceptanceDatabase(t *testing.T, ctx context.Context, containerID string) {
	t.Helper()
	path := os.Getenv("INKWORDS_ACCEPTANCE_RESTORE_SQL")
	if path == "" {
		return
	}
	require.True(t, filepath.IsAbs(path))
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	command := exec.CommandContext(ctx, "docker", "exec", "-i", containerID, "psql", "-v", "ON_ERROR_STOP=1", "-U", "evaluation", "-d", "approved_practice_acceptance")
	command.Stdin = file
	require.NoError(t, command.Run(), "restore explicitly selected isolated acceptance dump")
}

func acceptanceAssessmentService(db *gorm.DB, source *mastery.Service, verification app.VerificationSource, dir string) (*app.Service, error) {
	if os.Getenv("INKWORDS_ACCEPTANCE_CAPTURE") != "approved" {
		return BuildAssessmentService(db, source, verification)
	}
	model := firstNonEmpty(os.Getenv("MASTERY_DEEPSEEK_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	policy, timeout, err := resolveAssessmentPolicy(os.Getenv("MASTERY_ASSESSMENT_PROFILE"), model)
	if err != nil || policy == nil {
		return nil, fmt.Errorf("capture requires configured local evaluation policy")
	}
	port := platformllm.NewDeepSeekGenerationAdapterWithTimeout(platformllm.NewDeepSeekClient(os.Getenv("MASTERY_DEEPSEEK_API_KEY")), timeout)
	engine := app.NewGeneratorWithTaskPolicy(&acceptanceCapturedPort{inner: port, dir: dir}, "deepseek", model, *policy)
	service := app.NewService(app.NewVerifiedInputSource(source, verification), engine, persistence.NewStore(db))
	return service, service.Recover(context.Background())
}

type acceptanceCapturedPort struct {
	inner generation.Port
	dir   string
	mu    sync.Mutex
	calls int
}

func (port *acceptanceCapturedPort) Generate(ctx context.Context, request generation.Request) (generation.Result, error) {
	port.mu.Lock()
	defer port.mu.Unlock()
	if port.calls >= 2 {
		return generation.Result{}, fmt.Errorf("acceptance provider budget exhausted")
	}
	port.calls++
	write := func(name string, value any) error {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(port.dir, fmt.Sprintf("provider-%02d-%s.json", port.calls, name)), data, 0600)
	}
	if err := write("request", request); err != nil {
		return generation.Result{}, err
	}
	result, err := port.inner.Generate(ctx, request)
	if err == nil {
		// Normal output only, for these explicitly marked nonpersonal inputs.
		// Provider error bodies and credentials are never persisted.
		err = write("result", result)
	}
	return result, err
}
