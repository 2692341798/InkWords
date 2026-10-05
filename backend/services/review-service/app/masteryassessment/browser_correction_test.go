package masteryassessment

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"inkwords-backend/services/review-service/domain/mastery"
)

type browserCorrectionFixture struct {
	Receipt      realAssessmentArtifact
	Original     []byte
	Corrections  []mastery.AssessmentCorrection
	Effective    mastery.AssessmentFeedback
	ExpectedHash string
}

func (f *browserCorrectionFixture) job(name string) map[string]any {
	return map[string]any{"id": name, "objective_id": f.Receipt.Input.ObjectiveID, "attempt_id": f.Receipt.Input.AttemptID, "status": "succeeded", "input": f.Receipt.Input, "result": f.Receipt.Result, "corrections": f.Corrections, "effective_feedback": f.Effective, "effective_hash": mastery.AssessmentFeedbackHash(f.Effective)}
}

// TestBrowserCorrectionReplay serves a loopback-only domain harness for the
// real frontend controls. It has no provider, production DB or learning port.
func TestBrowserCorrectionReplay(t *testing.T) {
	source := os.Getenv("INKWORDS_ASSESSMENT_BROWSER_SOURCE")
	if source == "" {
		t.Skip("requires explicit frozen browser receipts")
	}
	proposals, output := os.Getenv("INKWORDS_ASSESSMENT_BROWSER_PROPOSALS"), os.Getenv("INKWORDS_ASSESSMENT_BROWSER_OUTPUT")
	require.NotEmpty(t, proposals)
	require.NotEmpty(t, output)
	require.NoError(t, os.MkdirAll(output, 0o700))
	fixtures := map[string]*browserCorrectionFixture{}
	names := []string{"complete-causal-answer", "incomplete-answer", "misconception-answer"}
	selection := os.Getenv("INKWORDS_ASSESSMENT_BROWSER_CASES")
	require.Contains(t, []string{"", "code"}, selection)
	if selection == "code" {
		names = []string{"correct-method-selection", "wrong-method-selection"}
	}
	for _, name := range names {
		f := &browserCorrectionFixture{Corrections: []mastery.AssessmentCorrection{}}
		var err error
		f.Original, err = os.ReadFile(filepath.Join(source, name+".json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(f.Original, &f.Receipt))
		require.NoError(t, f.Receipt.Result.ValidateFeedback(*f.Receipt.Input))
		f.Effective = *f.Receipt.Result.Feedback
		data, err := os.ReadFile(filepath.Join(proposals, name+".json"))
		require.NoError(t, err)
		var proposal struct {
			OriginalSHA string `json:"original_sha256"`
			Hash        string `json:"proposed_effective_hash"`
		}
		require.NoError(t, json.Unmarshal(data, &proposal))
		require.Equal(t, digest(string(f.Original)), proposal.OriginalSHA)
		f.ExpectedHash = proposal.Hash
		fixtures[name] = f
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var mu sync.Mutex
	done := make(chan struct{}, 1)
	posts := 0
	write := func(w http.ResponseWriter, code int, data any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "data": data})
	}
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5192")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/fixtures" && r.Method == http.MethodGet {
			jobs := []any{}
			for _, name := range names {
				jobs = append(jobs, fixtures[name].job(name))
			}
			write(w, 200, jobs)
			return
		}
		if r.URL.Path == "/stop" && r.Method == http.MethodPost {
			write(w, 200, nil)
			select {
			case done <- struct{}{}:
			default:
			}
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/corrections/")
		f := fixtures[name]
		if f == nil || r.Method != http.MethodPost {
			write(w, 404, nil)
			return
		}
		var input CorrectionInput
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			write(w, 400, nil)
			return
		}
		correction := mastery.AssessmentCorrection{ID: input.ID.String(), ReviewerID: "automated-ui-fixture-not-human", CorrectedAt: time.Now().UTC(), Reason: input.Reason, InputHash: f.Receipt.Result.InputHash, PreviousHash: input.PreviousHash, Changes: input.Changes, Findings: input.Findings}
		sequence := append(append([]mastery.AssessmentCorrection{}, f.Corrections...), correction)
		effective, err := mastery.ReplayAssessmentCorrections(*f.Receipt.Input, *f.Receipt.Result.Feedback, sequence)
		if err != nil {
			write(w, 409, map[string]string{"error": "correction contract rejected"})
			return
		}
		f.Corrections, f.Effective = sequence, effective
		posts++
		record, err := json.MarshalIndent(map[string]any{"origin": "automated_ui_acceptance", "human_reviews": 0, "learning_record_writes": 0, "request": input, "job": f.job(name)}, "", "  ")
		if err != nil {
			write(w, 500, nil)
			return
		}
		file, err := os.OpenFile(filepath.Join(output, name+"-"+input.ID.String()+".json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			write(w, 500, nil)
			return
		}
		_, err = file.Write(record)
		_ = file.Close()
		if err != nil {
			write(w, 500, nil)
			return
		}
		write(w, 200, f.job(name))
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()
	metadata, err := json.Marshal(map[string]any{"url": "http://" + listener.Addr().String(), "origin": "automated_ui_acceptance", "provider_calls": 0, "human_reviews": 0, "learning_record_writes": 0})
	require.NoError(t, err)
	file, err := os.OpenFile(filepath.Join(output, "server.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = file.Write(metadata)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	select {
	case <-done:
	case <-time.After(15 * time.Minute):
		t.Fatal("browser acceptance timed out")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, name := range names {
		f := fixtures[name]
		require.NotEmpty(t, f.Corrections)
		require.Equal(t, f.ExpectedHash, mastery.AssessmentFeedbackHash(f.Effective), name)
		after, err := os.ReadFile(filepath.Join(source, name+".json"))
		require.NoError(t, err)
		require.Equal(t, f.Original, after)
		require.NoError(t, f.Receipt.Result.ValidateFeedback(*f.Receipt.Input))
	}
	t.Logf("validated_correction_posts=%d provider_calls=0 human_reviews=0 learning_record_writes=0", posts)
}
