package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"inkwords-backend/services/review-service/domain/mastery"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const maxProjectionBytes = 3 * 1024 * 1024

// Client reads only the configured local core API, with no redirects or retry.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient pins the server-controlled origin; no browser URL is accepted.
func NewClient(baseURL string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("invalid core API origin")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 4
	return &Client{baseURL: strings.TrimRight(u.String(), "/"), http: &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// LoadApprovedPractice checks the core response's workspace and revision before
// allowing it to become a frozen review-service learning basis.
func (client *Client) LoadApprovedPractice(ctx context.Context, workspaceID, chapterID, revisionID uuid.UUID) (mastery.PracticeBasis, error) {
	if workspaceID == uuid.Nil || chapterID == uuid.Nil || revisionID == uuid.Nil {
		return mastery.PracticeBasis{}, fmt.Errorf("invalid practice identity")
	}
	target := client.baseURL + "/api/v1/textbook-projects/chapters/" + chapterID.String() + "/projections?revision_id=" + revisionID.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return mastery.PracticeBasis{}, fmt.Errorf("prepare practice request")
	}
	response, err := client.http.Do(req)
	if err != nil {
		return mastery.PracticeBasis{}, fmt.Errorf("approved practice service unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return mastery.PracticeBasis{}, fmt.Errorf("approved practice unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxProjectionBytes+1))
	if err != nil || len(data) > maxProjectionBytes {
		return mastery.PracticeBasis{}, fmt.Errorf("invalid practice response size")
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			WorkspaceID uuid.UUID `json:"workspace_id"`
			Blog        struct {
				Title string `json:"title"`
			} `json:"blog"`
			Learning sharedtextbook.LearningProjection `json:"learning"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code != 0 || envelope.Data.WorkspaceID != workspaceID || envelope.Data.Learning.ChapterID != chapterID.String() || envelope.Data.Learning.RevisionID != revisionID.String() || envelope.Data.Learning.Format != "inkwords.learning-projection.v2" || envelope.Data.Learning.Validate() != nil {
		return mastery.PracticeBasis{}, fmt.Errorf("approved practice contract mismatch")
	}
	return mastery.PracticeBasis{WorkspaceID: workspaceID, Title: envelope.Data.Blog.Title, Learning: envelope.Data.Learning}, nil
}
