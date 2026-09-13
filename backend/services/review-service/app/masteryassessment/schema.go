package masteryassessment

import "encoding/json"

var feedbackSchema = json.RawMessage(`{
  "type":"object","additionalProperties":false,
  "required":["criteria","correct_points","missing_points","misconceptions","next_hint","remediation"],
  "$defs":{
    "finding":{"type":"object","additionalProperties":false,"required":["text","evidence_ids"],"properties":{"text":{"type":"string","minLength":1,"maxLength":2000},"evidence_ids":{"type":"array","minItems":1,"items":{"type":"string"}}}}
  },
  "properties":{
    "criteria":{"type":"array","minItems":5,"maxItems":20,"items":{"type":"object","additionalProperties":false,"required":["id","score","reason","answer_quote","evidence_ids"],"properties":{"id":{"type":"string"},"score":{"type":["integer","null"],"minimum":0,"maximum":4},"reason":{"type":"string","minLength":1,"maxLength":2000},"answer_quote":{"type":"string"},"evidence_ids":{"type":"array","minItems":1,"items":{"type":"string"}}}}},
    "correct_points":{"type":"array","maxItems":20,"items":{"$ref":"#/$defs/finding"}},
    "missing_points":{"type":"array","maxItems":20,"items":{"$ref":"#/$defs/finding"}},
    "misconceptions":{"type":"array","maxItems":20,"items":{"$ref":"#/$defs/finding"}},
    "next_hint":{"$ref":"#/$defs/finding"},
    "remediation":{"type":"array","minItems":1,"maxItems":20,"items":{"$ref":"#/$defs/finding"}}
  }
}`)
