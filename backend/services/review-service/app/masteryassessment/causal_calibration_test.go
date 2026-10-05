package masteryassessment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// These are operator-authored synthetic evaluation answers, never learner
// attempts or evidence of a real reader's mastery.
const proceduralAcceptanceAnswer = "注册阶段，RouterGroup.GET 把 GET 方法、完整路径和处理函数交给 handle，最终由 Engine.addRoute 写入 GET 对应的路由树。请求到达后，Engine.handleHTTPRequest 先按 HTTP 方法选择树，再用路径执行 getValue；找到后运行 handlers。路径没有匹配时返回 404；只有启用 HandleMethodNotAllowed 且其他方法树存在同一路径时才返回 405。"

const causalAcceptanceAnswer = proceduralAcceptanceAnswer + "登记把路径与 handlers 的对应关系保存进方法树，请求查找依赖已经保存的对应关系；没有登记就没有该路径的 handlers 可取，查询不会临时创造处理函数。同一个路径可以分别登记 GET 和 POST 的处理函数，因此方法也是查找身份的一部分；先选择方法树把不同操作隔开，仅有 GET 的路径不会让 POST 自动命中那个 GET 处理函数。之后的路径匹配再区分同一方法下的不同资源。"

const incompleteAcceptanceAnswer = "Gin 收到请求后会查找路由，然后执行处理函数。"

const misconceptionAcceptanceAnswer = "Gin 只看 URL 路径，HTTP 方法不会影响路由选择；没有登记的路径会在第一次请求时自动创建并返回 200。"

func TestCausalCalibrationInstructionsSeparateCorrectSequenceFromExplanation(t *testing.T) {
	input := realAcceptanceInput(0, proceduralAcceptanceAnswer)
	request, err := requestForAssessment("test-model", input)
	require.NoError(t, err)
	require.Contains(t, request.SystemInstruction, "只列出先后步骤")
	require.Contains(t, request.SystemInstruction, "作答引文自身须表达")
	require.Contains(t, request.SystemInstruction, "不要从来源替作答补出原因")
	require.Contains(t, request.SystemInstruction, "正确的流程仍可")
	require.NotContains(t, request.SystemInstruction, input.Answer)
	flowPreview, err := NewGenerator(&capturedPort{}, "deepseek", "test-model").Preview(input)
	require.NoError(t, err)
	input.Answer = causalAcceptanceAnswer
	causalPreview, err := NewGenerator(&capturedPort{}, "deepseek", "test-model").Preview(input)
	require.NoError(t, err)
	require.NotEqual(t, flowPreview.RequestHash, causalPreview.RequestHash)
	require.True(t, strings.HasPrefix(causalAcceptanceAnswer, proceduralAcceptanceAnswer), "paired answers must retain the same correct sequence")
}
