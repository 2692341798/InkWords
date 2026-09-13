package masteryassessment

import (
	"encoding/json"

	"inkwords-backend/shared/kernel/generation"
)

const decisionInstruction = `
本次按 decision_policy 返回 decisions：每个冻结 rubric 条目恰好一个判断。criterion_id 使用原 ID；requirement 逐字复制该条目的完整 description。先核对实际作答满足了本项哪些要求，再选择 status 并评分。
satisfied 表示本项必答要求均已在作答中体现，对应 3–4 分且 gaps 为 []；unsatisfied 表示本项仍有必答要求未体现或错误，对应 0–2 分，并在 gaps 中逐项写 requirement_quote 和 reason。requirement_quote 必须逐字摘自该条目的 description，reason 说明作答缺少或误解了该要求的什么内容。不能从其他条目、参考答案或来源加入本项没有要求的条件。
无法从给定证据判断时使用 unknown，分数为 null、gaps 为 []，在 criteria.reason 说明未知原因；缺少同一工件运行事实的运行项必须如此。来源或参考答案所讲的理由未在学习者作答出现，不能当作作答已满足。仅复述调用顺序的因果项按冻结因果要求判断，不能把流程引文解释成已说明原因。
criteria.reason、answer_quote、分数与 decisions 必须一致；承认本项必答要求缺失时不能同时选择 satisfied 或给 3–4 分。缺口是任务要求的覆盖判断，不是仅仅建议可以拓展的细节。不得猜测或编造满足证据。`

func addDecisionRequest(request *generation.Request) error {
	var schema map[string]any
	if json.Unmarshal(request.ResponseSchema, &schema) != nil {
		return ErrInvalidFeedback
	}
	var decisionSchema map[string]any
	if json.Unmarshal([]byte(`{
"type":"array","minItems":5,"maxItems":20,"items":{
 "type":"object","additionalProperties":false,"required":["criterion_id","requirement","status","gaps"],
 "properties":{
  "criterion_id":{"type":"string"},
  "requirement":{"type":"string","minLength":1,"maxLength":2000},
  "status":{"type":"string","enum":["satisfied","unsatisfied","unknown"]},
  "gaps":{"type":"array","maxItems":10,"items":{"type":"object","additionalProperties":false,"required":["requirement_quote","reason"],"properties":{"requirement_quote":{"type":"string","minLength":1,"maxLength":2000},"reason":{"type":"string","minLength":1,"maxLength":2000}}}}
 }
}}`), &decisionSchema) != nil {
		return ErrInvalidFeedback
	}
	schema["required"] = append(schema["required"].([]any), "decisions")
	schema["properties"].(map[string]any)["decisions"] = decisionSchema
	var err error
	request.ResponseSchema, err = json.Marshal(schema)
	request.SystemInstruction += decisionInstruction
	return err
}
