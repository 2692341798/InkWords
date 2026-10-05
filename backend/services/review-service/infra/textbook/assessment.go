package textbook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// LoadApprovedAssessmentEvidence retrieves the exact approved task's source window.
func (client *Client) LoadApprovedAssessmentEvidence(ctx context.Context, workspaceID, chapterID, revisionID uuid.UUID, taskID string) (sharedtextbook.PracticeEvidenceProjection, error) {
	var projection sharedtextbook.PracticeEvidenceProjection
	if workspaceID == uuid.Nil || chapterID == uuid.Nil || revisionID == uuid.Nil || taskID == "" {
		return projection, fmt.Errorf("invalid assessment source identity")
	}
	query := url.Values{"revision_id": {revisionID.String()}, "task_id": {taskID}}
	target := client.baseURL + "/api/v1/textbook-projects/chapters/" + chapterID.String() + "/practice-evidence?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return projection, fmt.Errorf("prepare assessment source request")
	}
	response, err := client.http.Do(request)
	if err != nil {
		return projection, fmt.Errorf("assessment source service unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return projection, fmt.Errorf("approved assessment sources unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxProjectionBytes+1))
	if err != nil || len(data) > maxProjectionBytes {
		return projection, fmt.Errorf("invalid assessment source response size")
	}
	var envelope struct {
		Code int                                       `json:"code"`
		Data sharedtextbook.PracticeEvidenceProjection `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code != 0 {
		return projection, fmt.Errorf("invalid assessment source response")
	}
	projection = envelope.Data
	if projection.Validate() != nil || projection.WorkspaceID != workspaceID.String() || projection.ChapterID != chapterID.String() || projection.RevisionID != revisionID.String() || projection.TaskID != taskID {
		return sharedtextbook.PracticeEvidenceProjection{}, fmt.Errorf("assessment source contract mismatch")
	}
	return projection, nil
}
