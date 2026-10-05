package learnerverification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	app "inkwords-backend/services/review-service/app/masteryverification"
	"inkwords-backend/services/review-service/domain/mastery"
	v1 "inkwords-backend/services/review-service/transport/http/v1"
	"inkwords-backend/shared/kernel/httpx"
	shared "inkwords-backend/shared/kernel/textbook"
	platformpostgres "inkwords-backend/shared/platform/postgres"
)

// The source is explicitly synthetic; the round trip validates the production
// transport/store/runner boundary, not core-api approval or a real learner.
type roundtripInputs struct{ dir string }

func (source roundtripInputs) PrepareLearnerVerificationInput(_ context.Context, owner, objective, attempt uuid.UUID, runner shared.LearnerRunnerIdentity) (shared.LearnerVerificationInput, error) {
	for _, name := range []string{"correct-method-selection", "wrong-method-selection"} {
		data, err := os.ReadFile(filepath.Join(source.dir, name+"-input.json"))
		if err != nil {
			continue
		}
		var input shared.LearnerVerificationInput
		if json.Unmarshal(data, &input) == nil && input.Plan.WorkspaceID == owner.String() && input.Plan.ObjectiveID == objective.String() && input.Plan.AttemptID == attempt.String() && input.Plan.Runner == runner {
			return input, nil
		}
	}
	return shared.LearnerVerificationInput{}, app.ErrNotFound
}

type capturedRunner struct {
	*Client
	mu   sync.Mutex
	refs []shared.LearnerVerificationReference
}

func (runner *capturedRunner) Execute(ctx context.Context, ref shared.LearnerVerificationReference) (shared.LearnerVerificationReport, error) {
	runner.mu.Lock()
	runner.refs = append(runner.refs, ref)
	runner.mu.Unlock()
	return runner.Client.Execute(ctx, ref)
}

func TestRealLearnerHTTPRoundtrip(t *testing.T) {
	if os.Getenv("INKWORDS_REAL_LEARNER_HTTP") != "approved" {
		t.Skip("explicit real HTTP/sandbox opt-in required")
	}
	dir := os.Getenv("INKWORDS_LEARNER_HTTP_DIR")
	require.True(t, filepath.IsAbs(dir))
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	container, err := postgrescontainer.Run(ctx, "postgres:14-alpine", postgrescontainer.WithDatabase("learner_http_evaluation"), postgrescontainer.WithUsername("evaluation"), postgrescontainer.WithPassword("test-only-password"), postgrescontainer.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := platformpostgres.InitReview(dsn)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	client, err := NewClient(os.Getenv("INKWORDS_EVALUATION_RUNNER_ORIGIN"))
	require.NoError(t, err)
	runner := &capturedRunner{Client: client}
	store := NewStore(db)
	service := app.NewService(runner, roundtripInputs{dir: dir}, store)
	var owner uuid.UUID
	// Owner comes from the same frozen operator fixture factory, not an HTTP field.
	ownerFile := filepath.Join(dir, "owner.json")
	ownerBytes, err := os.ReadFile(ownerFile)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(ownerBytes, &owner))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1.RegisterLearnerVerificationRoutes(router, httpx.LocalWorkspaceContext(func(context.Context) (uuid.UUID, error) { return owner, nil }), service)
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	writeRoundtripJSON(t, filepath.Join(dir, "review-ready.json"), map[string]any{"port": port, "origin": "operator_evaluation", "database": "isolated_testcontainer"})
	require.Eventually(t, func() bool { _, err := os.Stat(filepath.Join(dir, "fixtures-ready")); return err == nil }, 90*time.Second, 200*time.Millisecond)
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	call := func(method, path string, input, target any) int {
		var body io.Reader
		if input != nil {
			data, err := json.Marshal(input)
			require.NoError(t, err)
			body = bytes.NewReader(data)
		}
		request, err := http.NewRequestWithContext(ctx, method, base+path, body)
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&envelope))
		if response.StatusCode == 200 && target != nil {
			require.NoError(t, json.Unmarshal(envelope.Data, target))
		}
		return response.StatusCode
	}
	for _, name := range []string{"correct-method-selection", "wrong-method-selection"} {
		var input shared.LearnerVerificationInput
		data, err := os.ReadFile(filepath.Join(dir, name+"-input.json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &input))
		objective, attempt := uuid.MustParse(input.Plan.ObjectiveID), uuid.MustParse(input.Plan.AttemptID)
		require.Equal(t, owner.String(), input.Plan.WorkspaceID)
		require.NoError(t, db.Exec(`INSERT INTO mastery_objectives(id,workspace_id,chapter_id,title,behavior,required_skills,rubric,key_points,evidence_refs) VALUES(?,?,?,'HTTP 评测','操作方夹具','["reproduce"]','["方法选择"]','["路由"]','["gin-method-tree-identity"]') ON CONFLICT(id) DO NOTHING`, objective, owner, input.Plan.ChapterID).Error)
		require.NoError(t, db.Exec(`INSERT INTO mastery_attempts(id,objective_id,skill,answer,correct,independent,hint_count,took_millis,confidence,error_kinds,attempted_at) VALUES(?,?,'reproduce','操作方 HTTP 评测，非个人学习',false,false,0,1000,1,'[]',CURRENT_TIMESTAMP)`, attempt, objective).Error)
		prefix := fmt.Sprintf("/api/v1/mastery/objectives/%s/attempts/%s", objective, attempt)
		var preview app.Preview
		require.Equal(t, 200, call("GET", prefix+"/verification-preview", nil, &preview))
		require.Equal(t, input.Plan.InputHash, preview.InputHash)
		var count int64
		require.NoError(t, db.Table("mastery_learner_verification_runs").Where("attempt_id=?", attempt).Count(&count).Error)
		require.Zero(t, count)
		request := app.StartInput{RequestID: uuid.New(), ExpectedInputHash: preview.InputHash}
		bad := request
		bad.ExpectedInputHash = digest('0')
		require.Equal(t, 409, call("POST", prefix+"/verifications", bad, nil))
		var started, duplicate app.Job
		require.Equal(t, 200, call("POST", prefix+"/verifications", request, &started))
		require.Equal(t, 200, call("POST", prefix+"/verifications", request, &duplicate))
		require.Equal(t, started.ID, duplicate.ID)
		var final app.Job
		require.Eventually(t, func() bool {
			if call("GET", prefix+"/verification", nil, &final) != 200 {
				return false
			}
			return final.Status != "queued" && final.Status != "running"
		}, 45*time.Second, 200*time.Millisecond)
		expected := "passed"
		if name == "wrong-method-selection" {
			expected = "failed"
		}
		require.Equal(t, expected, final.Status, final.ErrorCode)
		require.NotNil(t, final.Report)
		require.True(t, final.Report.ExecutionStarted)
		if expected == "passed" {
			require.Equal(t, 0, *final.Report.ExitCode)
		} else {
			require.Equal(t, 1, *final.Report.ExitCode)
			require.Contains(t, final.Report.Output, "method POST")
			require.Contains(t, final.Report.Output, "method DELETE")
		}
		runner.mu.Lock()
		ref := runner.refs[len(runner.refs)-1]
		runner.mu.Unlock()
		// Replaying an already consumed capability must not reopen the saved files.
		require.Equal(t, 404, call("POST", "/internal/v1/learner-verification/resolve", ref, nil))
		readInput, report, err := service.LatestAssessmentEvidence(ctx, owner, objective, attempt)
		require.NoError(t, err)
		require.Equal(t, input, *readInput)
		require.Equal(t, *final.Report, *report)
		var assessment mastery.AssessmentInput
		data, err = os.ReadFile(filepath.Join(dir, name+"-assessment.json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &assessment))
		attached, err := mastery.AttachLearnerVerification(assessment, *readInput, *report)
		require.NoError(t, err)
		require.Equal(t, "learner-runtime:"+started.ID.String(), attached.Evidence[len(attached.Evidence)-1].ID)
		// This reconstructs service/store objects in the same process; it is not
		// evidence of an operating-system process crash or restart.
		recreated := app.NewService(runner, roundtripInputs{dir: dir}, NewStore(db))
		require.NoError(t, recreated.Recover(ctx))
		persisted, err := recreated.Read(ctx, owner, objective, started.ID)
		require.NoError(t, err)
		require.Equal(t, final.Report, persisted.Report)
		require.NoError(t, db.Table("mastery_learner_verification_runs").Where("attempt_id=?", attempt).Count(&count).Error)
		require.EqualValues(t, 1, count)
		writeRoundtripJSON(t, filepath.Join(dir, name+"-http-result.json"), map[string]any{"origin": "operator_http_sandbox_evaluation", "job": final, "assessment_input": attached, "duplicate_request_same_job": true, "consumed_claim_rejected": true, "service_recreation_preserves_report": true, "persisted_runs": count, "provider_calls": 0})
		t.Logf("case=%s run_id=%s status=%s exit=%d persisted=%d", name, started.ID, final.Status, *final.Report.ExitCode, count)
	}
	runner.mu.Lock()
	calls := len(runner.refs)
	runner.mu.Unlock()
	require.Equal(t, 2, calls)
}

func writeRoundtripJSON(t *testing.T, path string, value any) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(file).Encode(value))
	require.NoError(t, file.Close())
}
