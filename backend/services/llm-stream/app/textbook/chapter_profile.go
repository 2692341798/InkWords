package textbook

import sharedtextbook "inkwords-backend/shared/kernel/textbook"

func headingsForProfile(profile sharedtextbook.ChapterProfile, headings []string) []string {
	result := append([]string(nil), headings...)
	if profile == sharedtextbook.ChapterProfileHandsOn {
		for i, heading := range result {
			switch heading {
			case "## 为什么要先登记路由":
				result[i] = "## 为什么先验证请求与响应"
			case "## 跟着源码走三步":
				result[i] = "## 把请求连到固定源码"
			}
		}
	}
	return result
}

func teachingRulesForProfile(profile sharedtextbook.ChapterProfile) []string {
	if profile != sharedtextbook.ChapterProfileHandsOn {
		return append([]string(nil), conceptTeachingRules...)
	}
	return []string{
		"本章是 Gin 集成动手章。使用已选定依赖包中的真实 github.com/gin-gonic/gin，不重复上一概念章的缩小路由树；解释它们的区别，并引用固定生产源码连接注册与请求处理路径。",
		"main.go 声明 func buildRouter() *gin.Engine，用 gin.New() 创建路由，设置 HandleMethodNotAllowed = true，注册 GET /orders 返回 200 与 orders-list，POST /orders 返回 201 与 order-created。响应仅为演示文本，订单不持久化；迁移任务独立增加 GET /health。",
		"main.go 声明 func main()，使用 http.ListenAndServe(\"127.0.0.1:38080\", buildRouter()) 启动本地服务并处理错误；只能绑定该回环地址。main_test.go 用 httptest.NewRecorder、http.NewRequest 和 router.ServeHTTP 检查请求；至少覆盖两种成功响应、未知路径 /missing 返回 404、PUT /orders 返回 405，逐项断言状态码及正文。",
		"main.go 与 main_test.go 必须是可解析的 package main 源码，所有 import 均实际使用，禁止点导入。测试函数只放 main_test.go，必须实际声明 func TestXxx(t *testing.T)，注释或字符串中的名称不算声明。",
		"先展示未登记 /orders 的预期失败，再添加 GET、POST 并测试。未知路径与错误方法的处理不同，解释 HandleMethodNotAllowed 的选择。必须明确：httptest 是进程内检查，不证明端口监听或浏览器成功；服务启动与浏览器取证需要另一次隔离验证，当前均保持未验证。",
		"从空目录的准备说明先解释 go mod init，再说明使用经审阅的固定 vendor 包及锁文件；不得使用 go get -u 或宣称安装已完成。给出 go test ./...、go run .、http://127.0.0.1:38080/orders 的逐步动作与预期；命令使用行内代码，不从教材命令触发执行。缺依赖时停止并回到依赖选择预检，不打开沙箱网络或借用宿主缓存。",
		"main.go 前段以 **代码来源：教学实现（main.go）。** 开头，说明这是教学集成示例，非生产用途，省略认证、订单存储、输入校验及服务超时/优雅关闭；不要说 Gin 自身缺少中间件、参数或并发能力。测试前段以 **代码来源：教学实现测试（main_test.go）。** 开头。",
		"学习目标、示例、练习、答案和排错须引用 buildRouter、实际请求与断言，不沿用 routeNode/addRoute/findRoute 的概念章练习。对运行只能给预期结果；实际通过需对应工件的 VerificationRun。",
	}
}

var conceptTeachingRules = []string{
	"main.go 必须从零实现按方法和路径登记、查找的缩小路由树：逐字包含 type routeNode struct、addRoute、findRoute；addRoute 和 findRoute 的参数都必须显式包含 method 与 path，且不得 import github.com/gin-gonic/gin。main.go 与 main_test.go 必须是可解析的 package main 源码，每个 import 都必须在对应文件中实际使用，禁止点导入；main_test.go 必须逐字包含 func Test；从空目录开始的动手步骤必须先执行 go mod init，再执行 go test ./...。",
	"main.go 必须实际声明顶层 func main()，构造缩小路由树、登记一条路径，再按方法和路径查找并打印结果；仅有结构体和辅助函数不是完整可运行文件。main_test.go 必须实际声明至少一个 func TestXxx(t *testing.T)，测试方法相同且路径命中、方法不同、路径不同三种情形；注释或字符串中的函数名不算声明。不要把测试函数放进 main.go。",
	"教学路由树必须先按 method 选择根节点，再按路径分段逐层访问 children：addRoute 遇到不存在的分段创建子节点，只在完整路径末端保存处理函数；findRoute 沿相同分段逐层查找，缺方法、缺节点或末端未登记处理函数都返回未找到。禁止将 method 与完整 path 拼接为单个键，也不得用完整 path 作为根节点 children 的键来冒充树。登记 /api/orders 与 /api/health 必须共享 api 节点；api 前缀节点可同时拥有自己的处理函数和两个子节点。正文用这两条路径逐步说明节点如何建立和复用，而不只列函数名。",
	"main_test.go 除命中和缺失外，还必须断言 /api/orders 与 /api/health 共享 api 节点、/api 未登记时不命中及登记后不破坏两个子路径，并覆盖同一路径在两个方法下分别登记且返回不同处理结果。边界说明必须明确：本例是按完整路径分段的普通前缀树，不等同于 Gin 生产实现的压缩路径树；仅支持规范化静态绝对路径，根路径、尾斜杠及重复斜杠的处理应与实际代码一致，不宣称复现 Gin 的重定向行为。practice_set 的题目、答案及评分依据必须引用实际生成的结构与函数，不得假设代码中不存在的变量、节点或分支。",
	"紧邻 main.go 围栏的来源段落必须以 **代码来源：教学实现（main.go）。** 开头，并在同一段逐字包含“非生产用途”和“省略”，列出参数/通配符、冲突检测、中间件、重定向、并发或内存优化中至少三项未覆盖的生产能力。main_test.go 围栏前一段必须以 **代码来源：教学实现测试（main_test.go）。** 开头。",
}
