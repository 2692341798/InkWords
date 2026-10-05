package textbook

import "encoding/json"

var practiceSetResponseSchema = json.RawMessage(`{
"type":"object","additionalProperties":false,"required":["version","tasks"],"properties":{
"version":{"type":"string","enum":["inkwords.practice-set.v1"]},
"tasks":{"type":"array","minItems":6,"maxItems":6,"items":{"type":"object","additionalProperties":false,"required":["id","mode","prompt","variation","expected_answer","rubric","hints","evidence_ids","min_delay_hours"],"properties":{
"id":{"type":"string","minLength":1,"maxLength":100},"mode":{"type":"string","enum":["explain","complete","reproduce","transfer","diagnose","retain"]},
"prompt":{"type":"string","minLength":1,"maxLength":2000},"variation":{"type":"string","minLength":1,"maxLength":2000},"expected_answer":{"type":"string","minLength":1,"maxLength":4000},"min_delay_hours":{"type":"integer","minimum":0,"maximum":8760},
"rubric":{"type":"array","minItems":5,"maxItems":5,"items":{"type":"object","additionalProperties":false,"required":["id","description"],"properties":{"id":{"type":"string"},"description":{"type":"string","minLength":1,"maxLength":2000}}}},
"hints":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"object","additionalProperties":false,"required":["level","text"],"properties":{"level":{"type":"integer","minimum":1,"maximum":3},"text":{"type":"string","minLength":1,"maxLength":1000}}}},
"evidence_ids":{"type":"array","minItems":1,"items":{"type":"string"}}
}}}}}`)

func sampleResponseSchemaWithPractice() (json.RawMessage, error) {
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(sampleChapterResponseSchema, &schema); err != nil {
		return nil, err
	}
	var properties map[string]json.RawMessage
	if err := json.Unmarshal(schema["properties"], &properties); err != nil {
		return nil, err
	}
	properties["practice_set"] = practiceSetResponseSchema
	var required []string
	if err := json.Unmarshal(schema["required"], &required); err != nil {
		return nil, err
	}
	required = append(required, "practice_set")
	var err error
	if schema["properties"], err = json.Marshal(properties); err != nil {
		return nil, err
	}
	if schema["required"], err = json.Marshal(required); err != nil {
		return nil, err
	}
	return json.Marshal(schema)
}
