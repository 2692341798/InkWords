package textbook

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// Real calls require operator-frozen inputs; unit-test source placeholders must
// never silently become the evidence of a paid generation acceptance run.
func loadRealGenerationRequest(dir string, audience sharedtextbook.AudienceLevel, model string) (SampleGenerationRequest, error) {
	var request SampleGenerationRequest
	if strings.TrimSpace(dir) == "" {
		return request, fmt.Errorf("real generation requires a frozen request directory")
	}
	file, err := os.Open(filepath.Join(dir, string(audience)+".json"))
	if err != nil {
		return request, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return request, fmt.Errorf("invalid real request size")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("invalid frozen request JSON")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return request, fmt.Errorf("trailing frozen request data")
	}
	if request.Audience != audience || request.BookContract.Reader.Audience != audience || request.GenerationTarget.ProviderName != "deepseek" || request.GenerationTarget.ModelName != model {
		return request, fmt.Errorf("frozen request audience or model mismatch")
	}
	if err := request.Validate(); err != nil {
		return request, err
	}
	snapshots := append([]sharedtextbook.SourceSnapshot{request.EvidencePack.PrimarySnapshot}, request.EvidencePack.PrimarySnapshots...)
	snapshots = append(snapshots, request.EvidencePack.OfficialSources...)
	for _, snapshot := range snapshots {
		hash, err := hex.DecodeString(strings.TrimPrefix(snapshot.ContentHash, "sha256:"))
		if err != nil || !strings.HasPrefix(snapshot.ContentHash, "sha256:") || len(hash) != 32 {
			return request, fmt.Errorf("real snapshot requires a full SHA-256 digest")
		}
		if snapshot.Kind == sharedtextbook.SourceKindGitRepository {
			commit, err := hex.DecodeString(snapshot.ResolvedVersion)
			if err != nil || len(commit) != 20 {
				return request, fmt.Errorf("real Git snapshot requires a fixed commit SHA")
			}
		}
	}
	for _, ref := range request.EvidencePack.Evidence {
		if digest(request.EvidencePack.Excerpts[ref.ID]) != ref.ContentHash {
			return request, fmt.Errorf("frozen evidence bytes do not match hash: %s", ref.ID)
		}
	}
	return request, nil
}

func TestRealRequestRequiresFrozenInputsAndPreservesIdentity(t *testing.T) {
	_, err := loadRealGenerationRequest("", sharedtextbook.AudienceProgramming, "model")
	require.ErrorContains(t, err, "frozen request directory")
	request := sampleGenerationRequest(t)
	request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: "deepseek", ModelName: "model"}
	request.EvidencePack.PrimarySnapshot.ContentHash = digest("primary fixture")
	for i := range request.EvidencePack.PrimarySnapshots {
		request.EvidencePack.PrimarySnapshots[i].ContentHash = digest("additional fixture")
	}
	for i := range request.EvidencePack.OfficialSources {
		request.EvidencePack.OfficialSources[i].ContentHash = digest("official fixture")
	}
	for i, ref := range request.EvidencePack.Evidence {
		request.EvidencePack.Evidence[i].ContentHash = digest(request.EvidencePack.Excerpts[ref.ID])
	}
	dir := t.TempDir()
	write := func() {
		data, err := json.Marshal(request)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "foundation.json"), data, 0o600))
	}
	write()
	loaded, err := loadRealGenerationRequest(dir, sharedtextbook.AudienceFoundation, "model")
	require.NoError(t, err)
	require.Equal(t, request, loaded)
	request.EvidencePack.PrimarySnapshot.ContentHash = "sha256:placeholder"
	write()
	_, err = loadRealGenerationRequest(dir, sharedtextbook.AudienceFoundation, "model")
	require.ErrorContains(t, err, "full SHA-256")
	request.EvidencePack.PrimarySnapshot.ContentHash = loaded.EvidencePack.PrimarySnapshot.ContentHash
	request.EvidencePack.PrimarySnapshot.ResolvedVersion = "main"
	write()
	_, err = loadRealGenerationRequest(dir, sharedtextbook.AudienceFoundation, "model")
	require.ErrorContains(t, err, "resolved version must be immutable")
	request.EvidencePack.PrimarySnapshot.ResolvedVersion = loaded.EvidencePack.PrimarySnapshot.ResolvedVersion
	write()
	_, err = loadRealGenerationRequest(dir, sharedtextbook.AudienceFoundation, "other-model")
	require.ErrorContains(t, err, "mismatch")
	request.BookContract.Reader.Audience = sharedtextbook.AudienceProgramming
	write()
	_, err = loadRealGenerationRequest(dir, sharedtextbook.AudienceFoundation, "model")
	require.ErrorContains(t, err, "mismatch")
	request.BookContract.Reader.Audience = sharedtextbook.AudienceFoundation
	request.EvidencePack.Excerpts[request.EvidencePack.Evidence[0].ID] += "changed"
	write()
	_, err = loadRealGenerationRequest(dir, sharedtextbook.AudienceFoundation, "model")
	require.ErrorContains(t, err, "hash")
}

func prepareRealGenerationRequests(t *testing.T, dir, model string, audiences []sharedtextbook.AudienceLevel) []SampleGenerationRequest {
	t.Helper()
	requests := make([]SampleGenerationRequest, 0, len(audiences))
	for _, audience := range audiences {
		request, err := loadRealGenerationRequest(dir, audience, model)
		require.NoError(t, err)
		providerRequest, err := providerRequestForSample(model, request)
		require.NoError(t, err)
		budget, err := sharedgeneration.CheckRequestBudget(DefaultSampleGenerationBudget(), providerRequest)
		require.NoError(t, err)
		require.False(t, budget.RequiresCompression)
		t.Logf("frozen audience=%s evidence_count=%d prompt_hash=%s estimated_input=%d allowed_input=%d max_output=%d estimate_method=%s", audience, len(request.EvidencePack.Evidence), digest(providerRequestHashInput(providerRequest)), budget.EstimatedInput, budget.AllowedInput, providerRequest.MaxOutputTokens, sharedgeneration.RequestBudgetEstimateMethod)
		requests = append(requests, request)
	}
	return requests
}

func TestFrozenRealGenerationInputsPreflight(t *testing.T) {
	dir := os.Getenv("INKWORDS_REAL_TEXTBOOK_REQUEST_DIR")
	if dir == "" {
		t.Skip("no frozen real inputs selected; no provider call")
	}
	model := realGenerationSetting(os.Getenv("TEXTBOOK_STANDARD_MODEL"), os.Getenv("DEEPSEEK_MODEL"))
	require.NotEmpty(t, model)
	audiences, err := realGenerationAudiences(os.Getenv("INKWORDS_REAL_TEXTBOOK_AUDIENCES"))
	require.NoError(t, err)
	prepareRealGenerationRequests(t, dir, model, audiences)
}

func writeRealGenerationAttempt(dir string, request SampleGenerationRequest, result SampleGeneration, failed bool) error {
	status := "candidate_requires_review"
	if failed {
		status = "generation_failed"
	}
	if failed && len(result.Quality.Failures) > 0 {
		status = "quality_rejected"
	}
	attempt := struct {
		Status  string                  `json:"status"`
		Origin  string                  `json:"origin"`
		Request SampleGenerationRequest `json:"request"`
		Result  SampleGeneration        `json:"result"`
	}{status, "automated_generation_acceptance", request, result}
	data, err := json.MarshalIndent(attempt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, string(request.Audience)+".attempt.json"), data, 0o600)
}

func TestRealAttemptRetainsRejectedResultBeforeAcceptance(t *testing.T) {
	request := sampleGenerationRequest(t)
	result := SampleGeneration{Quality: QualityReport{Failures: []string{"unverified_runtime_evidence"}}, ProviderCalls: 1}
	dir := t.TempDir()
	require.NoError(t, writeRealGenerationAttempt(dir, request, result, true))
	data, err := os.ReadFile(filepath.Join(dir, "foundation.attempt.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), `"status": "quality_rejected"`)
	require.Contains(t, string(data), "unverified_runtime_evidence")
	require.Contains(t, string(data), `"provider_calls": 1`)
}
