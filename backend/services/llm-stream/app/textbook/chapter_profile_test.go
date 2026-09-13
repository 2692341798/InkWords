package textbook

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"strings"
	"testing"
)

func TestHandsOnPromptUsesIntegrationContractInsteadOfTreeExercise(t *testing.T) {
	request := sampleGenerationRequest(t)
	request.BlueprintChapter.Profile = sharedtextbook.ChapterProfileHandsOn
	request.BookContract.ChapterProfiles = append(request.BookContract.ChapterProfiles, sharedtextbook.ChapterProfileHandsOn)
	prompt, err := providerRequestForSample("configured-model", request)
	require.NoError(t, err)
	require.NotContains(t, prompt.TaskInstruction, "不得 import github.com/gin-gonic/gin")
	require.NotContains(t, prompt.TaskInstruction, "逐字包含 type routeNode struct")
	require.Contains(t, prompt.TaskInstruction, "buildRouter")
	require.Contains(t, prompt.TaskInstruction, "127.0.0.1:38080")
	require.Contains(t, prompt.TaskInstruction, "进程内")
	require.Contains(t, prompt.TaskInstruction, "go test ./...")
	require.Contains(t, prompt.TaskInstruction, "go mod init")
	require.Contains(t, prompt.TaskInstruction, "## 把请求连到固定源码")
}

const integrationMainFixture = `package main
import (
 "net/http"
 "github.com/gin-gonic/gin"
)
func buildRouter() *gin.Engine {
 router := gin.New()
 router.HandleMethodNotAllowed = true
 router.GET("/orders", func(c *gin.Context) { c.String(200, "orders-list") })
 router.POST("/orders", func(c *gin.Context) { c.String(201, "order-created") })
 return router
}
func main() {
 if err := http.ListenAndServe("127.0.0.1:38080", buildRouter()); err != nil { panic(err) }
}
`
const integrationTestFixture = `package main
import (
 "net/http"
 "net/http/httptest"
 "testing"
)
func TestRoutes(t *testing.T) {
 router := buildRouter()
 for _, tc := range []struct { method, path string; status int; body string }{
  {"GET", "/orders", 200, "orders-list"},
  {"POST", "/orders", 201, "order-created"},
  {"GET", "/missing", 404, "404 page not found"},
  {"PUT", "/orders", 405, "405 method not allowed"},
 } {
  recorder := httptest.NewRecorder()
  request, err := http.NewRequest(tc.method, tc.path, nil)
  if err != nil { t.Fatal(err) }
  router.ServeHTTP(recorder, request)
  if recorder.Code != tc.status || recorder.Body.String() != tc.body { t.Fatalf("unexpected response") }
 }
}
`

func integrationManuscriptForTest() string {
	return "**代码来源：教学实现（main.go）。** 教学集成示例，非生产用途，省略认证与存储。\n\n```go\n" + integrationMainFixture + "```\n\n**代码来源：教学实现测试（main_test.go）。**\n\n```go\n" + integrationTestFixture + "```\n\n先执行 `go mod init`，安装选定的离线包，再执行 `go test ./...`。这是进程内检查，端口与浏览器均未验证。另执行 `go run .` 并打开 `http://127.0.0.1:38080/orders`。\n"
}

func TestGinIntegrationSourceContractDoesNotExecuteTeachingFiles(t *testing.T) {
	markdown := integrationManuscriptForTest()
	require.Empty(t, validateGinIntegrationImplementation(markdown))
	require.Contains(t, validateGinTeachingImplementation(markdown), "teaching_implementation_uses_production_library")
	for name, pair := range map[string][2]string{
		"missing import":            {"github.com/gin-gonic/gin", "example.com/fakegin"},
		"fake route comment":        {"router.POST(\"/orders\", func(c *gin.Context) { c.String(201, \"order-created\") })", "// router.POST(\"/orders\", handler)"},
		"wrong bind":                {"127.0.0.1:38080", "0.0.0.0:38080"},
		"prototype main":            {"func main() {\n if err := http.ListenAndServe(\"127.0.0.1:38080\", buildRouter()); err != nil { panic(err) }\n}", "func main()"},
		"missing status assertion":  {"recorder.Code != tc.status", "tc.status != tc.status"},
		"missing body assertion":    {"recorder.Body.String() != tc.body", "tc.body != tc.body"},
		"missing request execution": {"router.ServeHTTP(recorder, request)", "// router.ServeHTTP(recorder, request)"},
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, validateGinIntegrationImplementation(strings.ReplaceAll(markdown, pair[0], pair[1])))
		})
	}
}

func TestGeneratedProfileComesFromFrozenBlueprint(t *testing.T) {
	request := sampleGenerationRequest(t)
	chapter, err := BuildGinRequestLifecycleSample(request.EvidencePack)
	require.NoError(t, err)
	chapter.Profile = sharedtextbook.ChapterProfileHandsOn
	data, err := json.Marshal(chapter)
	require.NoError(t, err)
	port := &capturedGenerationPort{result: sharedgeneration.Result{Provider: "fake-provider", Model: "configured-model", Output: string(data)}}
	generation, err := NewPortSampleGenerator(port, "configured-model").Generate(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, sharedtextbook.ChapterProfileConcept, generation.Chapter.Profile)
	request.BlueprintChapter.Profile = sharedtextbook.ChapterProfileHandsOn
	_, err = NewPortSampleGenerator(port, "configured-model").Generate(context.Background(), request)
	require.ErrorContains(t, err, "book contract")
	require.Equal(t, 1, port.calls, "profile preflight must reject before provider call")
	request.BookContract.ChapterProfiles = append(request.BookContract.ChapterProfiles, sharedtextbook.ChapterProfileHandsOn)
	generation, err = NewPortSampleGenerator(port, "configured-model").Generate(context.Background(), request)
	require.Error(t, err)
	require.Equal(t, sharedtextbook.ChapterProfileHandsOn, generation.Chapter.Profile)
	require.Contains(t, generation.Quality.Failures, "gin_integration_imports_missing", "tree-only prose cannot pass as a hands-on integration chapter")
}
