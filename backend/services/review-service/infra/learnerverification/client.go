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

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type Client struct {
	baseURL string
	http    *http.Client
	runHTTP *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return nil, fmt.Errorf("invalid local course-runner origin")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 2
	redirect := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{baseURL: strings.TrimRight(u.String(), "/"), http: &http.Client{Timeout: 6 * time.Second, Transport: transport, CheckRedirect: redirect}, runHTTP: &http.Client{Timeout: 40 * time.Second, Transport: transport, CheckRedirect: redirect}}, nil
}

func (client *Client) Capability(ctx context.Context) (sharedtextbook.LearnerVerificationCapability, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/internal/v1/learner-verification/capability", nil)
	if err != nil {
		return sharedtextbook.LearnerVerificationCapability{}, fmt.Errorf("prepare course-runner capability request")
	}
	response, err := client.http.Do(request)
	if err != nil {
		return sharedtextbook.LearnerVerificationCapability{}, fmt.Errorf("学习者代码隔离运行器不可访问")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 16*1024+1))
	if err != nil || len(data) > 16*1024 || response.StatusCode != http.StatusOK {
		return sharedtextbook.LearnerVerificationCapability{}, fmt.Errorf("学习者代码隔离运行器返回无效预检")
	}
	var envelope struct {
		Code int                                          `json:"code"`
		Data sharedtextbook.LearnerVerificationCapability `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code != 200 || envelope.Data.Validate() != nil {
		return sharedtextbook.LearnerVerificationCapability{}, fmt.Errorf("学习者代码隔离运行器合同不匹配")
	}
	return envelope.Data, nil
}

func (client *Client) Execute(ctx context.Context, reference sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationReport, error) {
	if err := reference.Validate(); err != nil {
		return sharedtextbook.LearnerVerificationReport{}, err
	}
	body, err := json.Marshal(reference)
	if err != nil {
		return sharedtextbook.LearnerVerificationReport{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/internal/v1/learner-verification/runs", bytes.NewReader(body))
	if err != nil {
		return sharedtextbook.LearnerVerificationReport{}, fmt.Errorf("prepare learner verification run")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.runHTTP.Do(request)
	if err != nil {
		return sharedtextbook.LearnerVerificationReport{}, fmt.Errorf("学习者代码隔离运行器不可访问")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, sharedtextbook.MaxLearnerVerificationOutputBytes+64*1024+1))
	if err != nil || len(data) > sharedtextbook.MaxLearnerVerificationOutputBytes+64*1024 || response.StatusCode != http.StatusOK {
		return sharedtextbook.LearnerVerificationReport{}, fmt.Errorf("学习者代码隔离运行器未返回有效结果")
	}
	var envelope struct {
		Code int                                      `json:"code"`
		Data sharedtextbook.LearnerVerificationReport `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code != 200 {
		return sharedtextbook.LearnerVerificationReport{}, fmt.Errorf("学习者代码隔离运行器结果合同不匹配")
	}
	return envelope.Data, nil
}
