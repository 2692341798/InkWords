package learnerverification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	domain "inkwords-backend/services/course-runner/domain/learnerverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	learnerartifact "inkwords-backend/shared/platform/learnerartifact"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return nil, fmt.Errorf("invalid local review-service origin")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 2
	return &Client{baseURL: strings.TrimRight(u.String(), "/"), http: &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (client *Client) Resolve(ctx context.Context, reference sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error) {
	if err := reference.Validate(); err != nil {
		return sharedtextbook.LearnerVerificationInput{}, err
	}
	body, err := json.Marshal(reference)
	if err != nil {
		return sharedtextbook.LearnerVerificationInput{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/internal/v1/learner-verification/resolve", bytes.NewReader(body))
	if err != nil {
		return sharedtextbook.LearnerVerificationInput{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return sharedtextbook.LearnerVerificationInput{}, fmt.Errorf("review-service learner input unavailable")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, sharedtextbook.MaxLearnerArtifactBytes+128*1024+1))
	if err != nil || len(data) > sharedtextbook.MaxLearnerArtifactBytes+128*1024 || response.StatusCode != http.StatusOK {
		return sharedtextbook.LearnerVerificationInput{}, fmt.Errorf("review-service rejected learner input capability")
	}
	var envelope struct {
		Code int                                     `json:"code"`
		Data sharedtextbook.LearnerVerificationInput `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code != 200 || envelope.Data.ValidateFor(reference) != nil {
		return sharedtextbook.LearnerVerificationInput{}, fmt.Errorf("review-service learner input contract mismatch")
	}
	return envelope.Data, nil
}

type Stager struct{}

func (Stager) Stage(ctx context.Context, artifact sharedtextbook.LearnerArtifact, plan sharedtextbook.LearnerVerificationPlan) (domain.Workspace, error) {
	workspace, err := learnerartifact.Stage(ctx, artifact, plan)
	if err != nil {
		return domain.Workspace{}, err
	}
	return domain.Workspace{RootDir: workspace.RootDir, TreeHash: workspace.TreeHash, Cleanup: workspace.Cleanup}, nil
}
