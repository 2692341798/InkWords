package masteryassessment

import (
	"encoding/json"

	"inkwords-backend/services/review-service/domain/mastery"
	"inkwords-backend/shared/kernel/generation"
)

const judgmentInstruction = `
本次采用 judgments 作为唯一评分判断，不返回 criteria、decisions、correct_points、missing_points 或 misconceptions；系统会从同一组 judgments 生成这些显示内容。每个冻结 rubric 条目恰好一个 judgment，criterion_id 使用原 ID，不能新增要求。
先根据实际作答及给定来源判断本项必答要求是否满足，再填写 score/status/answer_quote/evidence_ids/gaps/unknown_reason。satisfied 表示全部必答要求已经体现，对应 3–4 分，gaps=[]、unknown_reason=""；unsatisfied 表示有必答要求缺失或错误，对应 0–2 分，gaps 至少一项且 unknown_reason=""。每个 gap 的 kind 为 omission（遗漏）或 misconception（错误理解），requirement_quote 逐字摘自本项 description，reason 只解释这一个要求的缺口，最多 300 字。不要把参考答案、来源或其他评分项的额外内容当成本项要求。
无法从证据判断时 status=unknown、score=null、gaps=[]，并在 unknown_reason 说明原因。缺少同一作答工件运行记录的运行项必须未知。所引作答、分数、判断和缺口必须一致；不能从来源或参考答案替作答补出未表达的解释。只复述流程未达到冻结因果要求时最多 2 分。保留基于已给来源的下一步弱提示和补救建议。`

func addJudgmentRequest(request *generation.Request, input mastery.AssessmentInput) error {
	var legacy map[string]any
	if json.Unmarshal(feedbackSchema, &legacy) != nil {
		return ErrInvalidFeedback
	}
	var schema map[string]any
	if json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["judgments","next_hint","remediation"],"properties":{"judgments":{"type":"array","minItems":5,"maxItems":20,"items":{"type":"object","additionalProperties":false,"required":["criterion_id","score","status","answer_quote","evidence_ids","gaps","unknown_reason"],"properties":{"criterion_id":{"type":"string"},"score":{"type":["integer","null"],"minimum":0,"maximum":4},"status":{"type":"string","enum":["satisfied","unsatisfied","unknown"]},"answer_quote":{"type":"string"},"evidence_ids":{"type":"array","minItems":1,"items":{"type":"string"}},"unknown_reason":{"type":"string","maxLength":1000},"gaps":{"type":"array","maxItems":5,"items":{"type":"object","additionalProperties":false,"required":["kind","requirement_quote","reason"],"properties":{"kind":{"type":"string","enum":["omission","misconception"]},"requirement_quote":{"type":"string","minLength":1,"maxLength":2000},"reason":{"type":"string","minLength":1,"maxLength":300}}}}}}},"next_hint":{"$ref":"#/$defs/finding"},"remediation":{"type":"array","minItems":1,"maxItems":20,"items":{"$ref":"#/$defs/finding"}}}}`), &schema) != nil {
		return ErrInvalidFeedback
	}
	schema["$defs"] = legacy["$defs"]
	if input.LearnerArtifact != nil {
		item := schema["properties"].(map[string]any)["judgments"].(map[string]any)["items"].(map[string]any)
		item["required"] = append(item["required"].([]any), "answer_path")
		item["properties"].(map[string]any)["answer_path"] = map[string]any{"type": "string", "maxLength": 200}
		item["properties"].(map[string]any)["evidence_ids"].(map[string]any)["minItems"] = 0
		item["required"] = append(item["required"].([]any), "answer_span")
		item["properties"].(map[string]any)["answer_span"] = map[string]any{"anyOf": []any{map[string]any{"type": "null"}, map[string]any{"type": "object", "additionalProperties": false, "required": []string{"format", "start_line", "end_line"}, "properties": map[string]any{"format": map[string]any{"type": "string", "const": mastery.AssessmentCodeSpanFormat}, "start_line": map[string]any{"type": "integer", "minimum": 1}, "end_line": map[string]any{"type": "integer", "minimum": 1}}}}}
		request.SystemInstruction += "\n只有需要运行证据的条目在本次完全没有 runtime 证据、且 score=null/status=unknown 时，evidence_ids 才可为 []，并说明缺少记录；不要引用技术来源来证明运行记录不存在。有分数、静态条目、已有运行记录或提示建议仍须引用给定且相关的 evidence id。"
	}
	var err error
	request.ResponseSchema, err = json.Marshal(schema)
	request.SystemInstruction += judgmentInstruction
	return err
}
