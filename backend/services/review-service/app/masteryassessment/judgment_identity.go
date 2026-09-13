package masteryassessment

import (
	"encoding/json"

	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
)

func groundJudgmentRequest(request *generation.Request, input mastery.AssessmentInput) error {
	var schema map[string]any
	if err := json.Unmarshal(request.ResponseSchema, &schema); err != nil {
		return ErrInvalidFeedback
	}
	judgments := schema["properties"].(map[string]any)["judgments"].(map[string]any)
	properties := judgments["items"].(map[string]any)["properties"].(map[string]any)
	criterionIDs := make([]string, 0, len(input.Rubric))
	for _, criterion := range input.Rubric {
		criterionIDs = append(criterionIDs, criterion.ID)
	}
	evidenceIDs := make([]string, 0, len(input.Evidence))
	for _, evidence := range input.Evidence {
		evidenceIDs = append(evidenceIDs, evidence.ID)
	}
	judgments["minItems"], judgments["maxItems"] = len(criterionIDs), len(criterionIDs)
	properties["criterion_id"].(map[string]any)["enum"] = criterionIDs
	properties["evidence_ids"].(map[string]any)["items"] = map[string]any{"type": "string", "enum": evidenceIDs}
	finding := schema["$defs"].(map[string]any)["finding"].(map[string]any)["properties"].(map[string]any)
	finding["evidence_ids"].(map[string]any)["items"] = map[string]any{"type": "string", "enum": evidenceIDs}
	// Envelope IDs also identify answers. The citation vocabulary must contain
	// authoritative input evidence only, including matching execution records.
	request.SystemInstruction += "\n本次 evidence_ids 只能选 response_schema 中列出的精确编号，保留全部字符，不添加 evidence- 前缀，不改成 chunk_id、snapshot_id、文件名或作答编号。文字 answer_quote 选取足以支持本项判断的一段连续原文，不改标点、不拼接不同位置、不插入省略号、不摘抄参考答案；引文与说明分开。"
	encoded, err := json.Marshal(schema)
	request.ResponseSchema = encoded
	return err
}
