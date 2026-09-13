// Package textbook builds deterministic, evidence-bound textbook candidates.
// It does not call a model; provider integration is added only after this
// pedagogical contract has a reproducible baseline.
package textbook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

const ginRequestLifecycleChapterID = "gin-request-routing-concept"

// EvidencePack is the stable core-to-worker evidence boundary.
type EvidencePack = sharedtextbook.GenerationEvidencePack

// SampleChapter is a candidate manuscript together with the learning structures it claims to contain.
type SampleChapter struct {
	Profile             sharedtextbook.ChapterProfile     `json:"profile,omitempty"`
	ID                  string                            `json:"id"`
	Audience            sharedtextbook.AudienceLevel      `json:"audience"`
	Title               string                            `json:"title"`
	Markdown            string                            `json:"markdown"`
	Scenario            sharedtextbook.ScenarioFrame      `json:"scenario"`
	Understanding       sharedtextbook.UnderstandingChain `json:"understanding"`
	LearningArc         sharedtextbook.LearningArc        `json:"learning_arc"`
	PracticeSet         sharedtextbook.PracticeSet        `json:"practice_set"`
	EvidenceIDs         []string                          `json:"evidence_ids"`
	Claims              []sharedtextbook.Claim            `json:"claims"`
	ContentHash         string                            `json:"content_hash"`
	GenerationMode      string                            `json:"generation_mode"`
	RuntimeVerification string                            `json:"runtime_verification"`
}

// BuildGinRequestLifecycleSample preserves the original foundation fixture API.
func BuildGinRequestLifecycleSample(pack EvidencePack) (SampleChapter, error) {
	return BuildGinRequestLifecycleSampleForAudience(pack, sharedtextbook.AudienceFoundation)
}

// BuildGinRequestLifecycleSampleForAudience creates an audience-specific,
// evidence-bound baseline. The shared mechanism and source citations remain
// fixed while reader assumptions and practice scaffolding change explicitly.
func BuildGinRequestLifecycleSampleForAudience(pack EvidencePack, audience sharedtextbook.AudienceLevel) (SampleChapter, error) {
	if err := audience.Validate(); err != nil {
		return SampleChapter{}, err
	}
	chapter, err := buildGinRequestLifecycleFoundationSample(pack)
	if err != nil {
		return SampleChapter{}, err
	}
	return adaptGinRequestLifecycleSample(chapter, audience), nil
}

// buildGinRequestLifecycleFoundationSample stays precise about the fact that
// the cited excerpts describe registration and lookup, not a live trace.
func buildGinRequestLifecycleFoundationSample(pack EvidencePack) (SampleChapter, error) {
	if err := pack.Validate(); err != nil {
		return SampleChapter{}, err
	}
	requiredEvidence := []string{"gin-routergroup-get", "gin-routergroup-handle", "gin-engine-add-route", "gin-engine-serve-http", "gin-engine-handle-http-request", "gin-node-get-value"}
	if err := requireEvidence(pack, requiredEvidence); err != nil {
		return SampleChapter{}, err
	}

	scenario := sharedtextbook.ScenarioFrame{
		Situation:   "你给新接口写了 GET(\"/orders\", handler)，却不知道框架把这行配置放到了哪里。",
		Trigger:     "接口变多后，靠记忆查找处理函数很慢，也难解释中间件为何会一起生效。",
		Consequence: "排错时容易只盯 handler，漏掉路径、路由组和处理链的注册步骤。",
		Analogy:     "把开店前的菜单登记想成给餐厅前台一本可查的点菜单：顾客到来前，菜名和负责厨房已经先登记好了。",
		Mapping: []sharedtextbook.AnalogyMapping{
			{AnalogyElement: "菜单上的菜名", TechnicalElement: "HTTP 方法和绝对路径"},
			{AnalogyElement: "负责厨房", TechnicalElement: "合并后的处理函数链"},
			{AnalogyElement: "前台点菜单", TechnicalElement: "Engine 持有的按 HTTP 方法组织的路由树"},
		},
		Mechanism:       "Gin 的 GET 会转给 handle 完成注册；请求到达 ServeHTTP 后，handleHTTPRequest 按方法选择树并让 getValue 沿路径查找处理函数。",
		Boundary:        "餐厅类比只帮助理解“预先登记并按条件查找”。它不代表 Gin 真的有前台、厨房或按订单逐个调用的线程模型。",
		DemonstrationID: "gin-route-registration-unverified",
	}
	understanding := sharedtextbook.UnderstandingChain{
		Need:         "让读者能从一行 GET 配置追到路由登记的位置，而不是把框架当黑箱。",
		Problem:      "手写 if/else 随接口增长会难查、难组合，也无法统一管理中间件。",
		Alternatives: "小型程序可直接用 net/http 的 HandleFunc；需要路由组和处理链组合时使用 Gin 的 RouterGroup；需要完全自定义匹配策略时可自己维护映射表，但要自行处理方法、参数和冲突。",
		Usage:        "先创建路由组，再用 GET/POST 登记路径和处理函数；需要共享步骤时用 Use 添加中间件。",
		Mechanism:    "GET 调用 handle；handle 计算绝对路径、合并 handlers，再调用 Engine.addRoute。请求到达 ServeHTTP 后，handleHTTPRequest 按 HTTP 方法选择树，并调用 getValue 按路径查找 handlers。",
		Observation:  "在调试器中应先看到启动阶段注册，再看到请求阶段由 ServeHTTP 进入 handleHTTPRequest；每个请求查树，但不会重新建树。",
		Tradeoffs:    "预先登记换来请求时更直接的查找；代价是重复路径或错误顺序通常在启动配置阶段暴露，需要读懂注册链定位。",
		AssessmentID: []string{"explain-route-registration", "complete-route", "transfer-route", "diagnose-route", "retain-route"},
	}
	arc := learningArc()
	claims := []sharedtextbook.Claim{
		{ID: "gin-get-delegates-to-handle", Text: "RouterGroup.GET 将 GET 方法、相对路径和处理函数交给 handle。", Kind: "source_fact", Critical: true, EvidenceIDs: []string{"gin-routergroup-get"}, Status: sharedtextbook.ClaimStatusVerified},
		{ID: "gin-handle-combines-chain", Text: "RouterGroup.handle 会合并处理函数链，再继续路由登记。", Kind: "source_fact", Critical: true, EvidenceIDs: []string{"gin-routergroup-handle"}, Status: sharedtextbook.ClaimStatusVerified},
		{ID: "gin-engine-adds-route", Text: "Engine.addRoute 将路径和处理函数链交给对应方法的路由树。", Kind: "source_fact", Critical: true, EvidenceIDs: []string{"gin-engine-add-route"}, Status: sharedtextbook.ClaimStatusVerified},
		{ID: "gin-serve-http-dispatches", Text: "Engine.ServeHTTP 将准备好的 Context 交给 handleHTTPRequest。", Kind: "source_fact", Critical: true, EvidenceIDs: []string{"gin-engine-serve-http"}, Status: sharedtextbook.ClaimStatusVerified},
		{ID: "gin-request-selects-route", Text: "handleHTTPRequest 按请求方法选择路由树，再调用 getValue 按路径查找处理函数链。", Kind: "source_fact", Critical: true, EvidenceIDs: []string{"gin-engine-handle-http-request", "gin-node-get-value"}, Status: sharedtextbook.ClaimStatusVerified},
	}
	markdown := `# 从一行 GET 到 Gin 的路由登记：先把请求的去向登记清楚

## 先遇到一个真实问题

你刚接手一个订单接口。代码里只有 GET("/orders", listOrders)，同事却问：“这个请求为什么会到 listOrders？路由组和中间件又是什么时候加进去的？”如果只回答“Gin 会处理”，你下次遇到 404 仍然不知道从哪里查。

把启动阶段想成餐厅开门前登记菜单：菜名对应 HTTP（超文本传输协议）方法和路径，负责厨房对应处理函数链，前台点菜单对应 Engine 中按方法组织的路由树。这个类比只说明“先登记、后查找”；Gin 不是餐厅，也不是每个请求都重新登记菜单。

## 先看全貌：这不是一次请求正在执行

本章把两个阶段接起来。注册阶段，RouterGroup.GET 把工作交给 handle；handle 计算绝对路径、合并处理函数，然后交给 Engine.addRoute。[evidence:gin-routergroup-get] [evidence:gin-routergroup-handle] [evidence:gin-engine-add-route] 请求阶段，ServeHTTP 把 Context 交给 handleHTTPRequest；后者按方法选择路由树，并调用 getValue 按路径找到处理函数链。[evidence:gin-engine-serve-http] [evidence:gin-engine-handle-http-request] [evidence:gin-node-get-value]

本节只引入四个核心元素：相对路径、处理函数链、RouterGroup 和 Engine。先把它们理解成“登记时要交齐的材料”，再记住准确名称；不要一次背更多内部类型。

## 为什么要先登记路由

不使用框架时，小程序可以用 net/http 的 HandleFunc。接口少时这样很直接。接口和共享步骤变多后，你还要自己管理前缀、方法、参数和“先执行哪些公共步骤”。Gin 的 RouterGroup 让这些配置集中在一起：GET 只是一个方便入口，真正登记由 handle 完成。[evidence:gin-routergroup-get] [evidence:gin-routergroup-handle]

可选方案至少有三种：

1. net/http.HandleFunc：适合极小服务，路径与方法组合需要你自己约束。
2. Gin RouterGroup：适合需要路径前缀、路由组和处理函数链的服务。
3. 自己维护映射表：适合研究或特殊协议，但参数解析、冲突检测和调试工具都要自己实现。

## 跟着源码走三步

### 第 1 步：GET 只是转发

GET 传入方法、相对路径和处理函数，然后调用 handle。[evidence:gin-routergroup-get]

### 第 2 步：handle 整理登记材料

handle 先计算绝对路径，再合并处理函数链，最后调用 engine.addRoute。因此，排查“路由组前缀有没有生效”时，先看路径计算；排查“中间件为什么一起出现”时，先看处理链合并。[evidence:gin-routergroup-handle]

### 第 3 步：Engine 把路由放进按方法组织的树

addRoute 会按 HTTP 方法取得对应根节点；第一次使用该方法时创建根节点，再把路径加入路由树。[evidence:gin-engine-add-route]

请求到达时不会再走 GET 或 addRoute。ServeHTTP 把请求交给 handleHTTPRequest；handleHTTPRequest 读取方法和路径，选中对应方法树，调用 getValue 查找，并在找到 handlers 后交给 Context.Next 执行。[evidence:gin-engine-serve-http] [evidence:gin-engine-handle-http-request] [evidence:gin-node-get-value]

## 按自然学习曲线练习

### 先验激活：先说出你已经见过的地址

想一想：浏览器访问 /api/orders 时，你能指出“方法、路径、处理函数”里的哪两项吗？先写下猜测；答不完整也没关系，后面的源码路径会补上它。

### 问题体验：别把 404 先怪到 handler 身上

假设你访问 GET /api/orders 得到 404。此刻不要改 handler：先检查方法、路由组前缀和相对路径，再确认启动时有没有执行 api.GET。这样能把“没有登记”和“登记后执行失败”分开。

### 全貌预告：三步各解决一个疑问

GET 回答“登记从哪里开始”，handle 回答“前缀和公共处理步骤怎样合在一起”，addRoute 回答“整理好的材料交到哪里”。带着这三问再读下面代码，就不会把每一行都当成孤立知识。

### 完整示范：先跟着一行配置走完

请按顺序读 GET → handle → addRoute 三个小节，并把每一步用“输入是什么、交给谁、解决什么”复述一遍。示范使用的是固定 Gin 快照中的三个片段，不是凭空画出的调用图。

### 引导练习：在提示下补全登记

先补全下方 api.GET 的第二个参数；提示：它必须接收 *gin.Context，且负责写出响应。完成后再把 /api 改成 /v2，并预测最终路径会怎样变化。

### 脚手架渐退：去掉提示后自己定位

现在合上本章的“三步”标题，只保留 GET、/orders 和 404 这三个线索。请自行决定先查看路径、处理函数链还是 handler；写下理由，再回来看你的顺序是否能缩小范围。

### 独立迁移：换一个没有见过的接口

从零新建 /health 路由，不复制 /orders 的 handler 内容。然后把它放进一个带前缀的路由组，并用自己的话解释 GET、handle、addRoute 在这次配置中各做了什么。

### 复述并关联：讲给未来的自己听

用一分钟录音或写一段话：为什么需要在启动阶段登记？为什么类比不能解释线程模型？如果同事把 404 归咎于 handler，你会如何带他按登记链排查？

### 延迟提取：三天后再测一次

三天后不看源码，画出 GET → handle → addRoute 的箭头；再补出 /api + /orders 的最终路径。若忘了，先只回看你写下的理由，不要立刻重读整章。

### 根据表现调整：把卡点变成下一次练习

若你能解释但不能补全代码，下一次只做“从零写 /health”；若能写代码却仍把 404 归咎于 handler，下一次只做三组不同前缀的排错题。这样复习针对你的表现，而不是机械重读。

## 动手：写一份最小登记代码

1. 打开 VS Code 1.90 或更新版本，选择“文件 → 打开文件夹”，新建一个空文件夹 gin-route-demo。
2. 在终端进入该文件夹，执行 go mod init example.com/gin-route-demo；预期看到新建的 go.mod。若提示 go 找不到，请安装 Go 并重新打开终端。
3. 新建 main.go，粘贴下面代码。它从零实现“按方法选树、按路径向下查找”的缩小模型，不导入 Gin。

**验证状态：未验证。** 本仓库尚未在隔离 runner 中运行该示例，因此这里没有把想象中的终端输出或截图当成运行结果。

**代码来源：教学实现（main.go）。** 这是非生产用途的缩小模型；它省略参数路由、通配符、冲突检测、中间件链、重定向、并发优化和内存复用。

` + "```go\n" + `package main

import (
	"fmt"
	"strings"
)

type handler func() string

type routeNode struct {
	children map[string]*routeNode
	handler  handler
}

type router struct {
	trees map[string]*routeNode
}

func newRouter() *router { return &router{trees: map[string]*routeNode{}} }

func (r *router) addRoute(method, path string, h handler) {
	root := r.trees[method]
	if root == nil {
		root = &routeNode{children: map[string]*routeNode{}}
		r.trees[method] = root
	}
	for _, part := range splitPath(path) {
		if root.children[part] == nil {
			root.children[part] = &routeNode{children: map[string]*routeNode{}}
		}
		root = root.children[part]
	}
	root.handler = h
}

func (r *router) findRoute(method, path string) (handler, bool) {
	node := r.trees[method]
	for _, part := range splitPath(path) {
		if node == nil {
			return nil, false
		}
		node = node.children[part]
	}
	if node == nil || node.handler == nil {
		return nil, false
	}
	return node.handler, true
}

func splitPath(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool { return r == '/' })
}

func main() {
	r := newRouter()
	r.addRoute("GET", "/api/orders", func() string { return "orders" })
	h, ok := r.findRoute("GET", "/api/orders")
	fmt.Println(ok, h())
}
` + "```\n" + `
4. 新建 main_test.go，加入同方法同路径命中、方法不同不命中、路径不同不命中三组测试。

**代码来源：教学实现测试（main_test.go）。** 它只验证上述缩小模型，不代表 Gin 生产实现已经通过验证。

` + "```go\n" + `package main

import "testing"

func TestRouterMatchesMethodAndPath(t *testing.T) {
	r := newRouter()
	r.addRoute("GET", "/api/orders", func() string { return "orders" })
	if h, ok := r.findRoute("GET", "/api/orders"); !ok || h() != "orders" {
		t.Fatal("expected registered handler")
	}
	if _, ok := r.findRoute("POST", "/api/orders"); ok {
		t.Fatal("different method must not match")
	}
	if _, ok := r.findRoute("GET", "/api/missing"); ok {
		t.Fatal("different path must not match")
	}
}
` + "```\n" + `
5. 执行 go test ./...；预期测试通过。再执行 go run .；预期看到 true orders。若测试出现空指针错误，请检查 findRoute 是否在访问 children 前判断 node 为 nil。

## 用自己的话检查理解

1. 不看本文，用一句话说明为什么路由要在启动时登记。
2. 补全 api.GET("/orders", ___) 的第二个参数，并解释它在登记阶段代表什么。
3. 把 /api 改成 /v2，说明你会先检查哪一步来解释最终路径的变化。
4. 某接口返回 404。先分别检查 HTTP 方法、路由组前缀和相对路径，再检查程序是否真的执行了路由配置；不要先猜 handler 内部出错。
5. 三天后，不看源码复述：GET、handle 和 addRoute 各自做什么，它们之间如何衔接？
6. 从零写出一个 /health 路由，并说明若它返回 404，你会先检查哪一个登记步骤。

## 你现在应该能解释什么

你应该能用自己的话说清：Gin 的 GET 不是魔法开关；它会把方法、路径和处理函数交给 handle，handle 再把整理后的信息交给 Engine 注册。你也应该能区分“登记阶段”与“请求执行阶段”，并知道类比在这里停止。`

	practice := ginPracticeFixture()
	chapter := SampleChapter{ID: ginRequestLifecycleChapterID, Audience: sharedtextbook.AudienceFoundation, Title: "从一行 GET 到 Gin 的路由登记：先把请求的去向登记清楚", Markdown: markdown + "\n\n" + sharedtextbook.RenderPracticeSet(practice), Scenario: scenario, Understanding: understanding, LearningArc: arc, PracticeSet: practice, EvidenceIDs: requiredEvidence, Claims: claims, GenerationMode: "deterministic_fixture", RuntimeVerification: "unverified"}
	chapter.ContentHash = digest(chapter.Markdown)
	return chapter, nil
}

func adaptGinRequestLifecycleSample(chapter SampleChapter, audience sharedtextbook.AudienceLevel) SampleChapter {
	type adaptation struct {
		prelude              string
		title                string
		situation            string
		need                 string
		priorObjective       string
		priorRecovery        string
		practiceMode         sharedtextbook.LearningTaskMode
		practicePromptSuffix string
		practiceAnswerSuffix string
	}
	profiles := map[sharedtextbook.AudienceLevel]adaptation{
		sharedtextbook.AudienceFoundation: {
			prelude:              "**读者起点：** 不要求你先会 Go 或 Gin。遇到函数、map 和测试时，本章会先说明它们在这个例子里做什么。",
			title:                chapter.Title,
			situation:            chapter.Scenario.Situation,
			need:                 chapter.Understanding.Need,
			priorObjective:       "从浏览器地址中找出路径，并先用白话区分配置和处理动作。",
			priorRecovery:        "回到地址栏例子，只圈出方法和路径；暂时不用记函数名。",
			practiceMode:         sharedtextbook.LearningTaskExplain,
			practicePromptSuffix: "先画出两条调用链，再为每个函数补一句白话职责。",
			practiceAnswerSuffix: "对零基础读者，函数名之后还要补充它接收什么、交出什么。",
		},
		sharedtextbook.AudienceProgramming: {
			prelude:              "**读者起点：** 假设你已经会读 Go 函数、map 和单元测试；不假设你使用过 Gin。本章把熟悉的代码结构映射到 Gin 的登记链和请求查找链。",
			title:                "用 Go 结构读懂 Gin 路由登记与查找",
			situation:            "你会写 Go 接口和测试，却第一次接手 Gin 服务；GET(\"/orders\", handler) 背后的路由组、处理链和查找树仍是黑箱。",
			need:                 "让读者把已有的 Go 函数、map 和测试经验迁移到 Gin 路由登记，而不假设已经用过 Gin。",
			priorObjective:       "回忆 map 查找和表驱动测试，再预测方法与路径为何需要两个维度。",
			priorRecovery:        "先写一个 map[method]map[path]handler 草图，再回到路由树。",
			practiceMode:         sharedtextbook.LearningTaskReproduce,
			practicePromptSuffix: "使用表驱动测试覆盖方法、路径、共用前缀和未登记组合。",
			practiceAnswerSuffix: "表驱动测试应让每个方法与路径组合都有独立期望，避免只验证成功路径。",
		},
		sharedtextbook.AudienceStackFamiliar: {
			prelude:              "**读者起点：** 假设你已经能创建 Gin 路由并使用调试器；不假设你读过这个固定版本的路由树实现。本章把重点放在调用边界、源码位置和缩小模型省略的复杂性。",
			title:                "从 RouterGroup 到路由树：核对 Gin 的登记与查找边界",
			situation:            "你已经能写 Gin 路由，但一次 404 需要你越过公开 API，准确判断故障发生在分组路径、处理链合并、方法树选择还是路径查找。",
			need:                 "让熟悉 Gin 用法的读者把调试观察映射到固定版本的生产源码路径，并明确教学模型省略了什么。",
			priorObjective:       "画出你记忆中的 Gin 请求路径，并标出哪些箭头只是推测。",
			priorRecovery:        "先只标 RouterGroup.GET 与 ServeHTTP 两端，再用六条证据补齐中间节点。",
			practiceMode:         sharedtextbook.LearningTaskExplain,
			practicePromptSuffix: "把每一步对应到固定快照中的生产源码符号，并注明教学 routeNode 未覆盖的生产能力。",
			practiceAnswerSuffix: "生产源码符号需要逐一引用证据；教学树只能解释方法分树与逐段查找，不能代替 Gin 的完整实现。",
		},
	}
	profile := profiles[audience]
	oldRenderedPractice := sharedtextbook.RenderPracticeSet(chapter.PracticeSet)
	for index := range chapter.PracticeSet.Tasks {
		if chapter.PracticeSet.Tasks[index].Mode == profile.practiceMode {
			chapter.PracticeSet.Tasks[index].Prompt += " " + profile.practicePromptSuffix
			chapter.PracticeSet.Tasks[index].ExpectedAnswer += " " + profile.practiceAnswerSuffix
			break
		}
	}
	chapter.Audience = audience
	chapter.Title = profile.title
	chapter.Scenario.Situation = profile.situation
	chapter.Understanding.Need = profile.need
	if len(chapter.LearningArc.Stages) > 0 {
		chapter.LearningArc.Stages[0].Objective = profile.priorObjective
		chapter.LearningArc.Stages[0].RecoveryRoute = profile.priorRecovery
	}
	manuscript := strings.TrimSuffix(chapter.Markdown, "\n\n"+oldRenderedPractice)
	if newline := strings.IndexByte(manuscript, '\n'); newline >= 0 {
		manuscript = manuscript[:newline] + "\n\n" + profile.prelude + manuscript[newline:]
	}
	firstLineEnd := strings.IndexByte(manuscript, '\n')
	if firstLineEnd >= 0 {
		manuscript = "# " + profile.title + manuscript[firstLineEnd:]
	}
	chapter.Markdown = manuscript + "\n\n" + sharedtextbook.RenderPracticeSet(chapter.PracticeSet)
	chapter.ContentHash = digest(chapter.Markdown)
	return chapter
}

func requireEvidence(pack EvidencePack, required []string) error {
	known := make(map[string]bool, len(pack.Evidence))
	for _, evidence := range pack.Evidence {
		known[evidence.ID] = true
	}
	for _, id := range required {
		if !known[id] {
			return fmt.Errorf("Gin sample requires evidence %q", id)
		}
	}
	return nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func learningArc() sharedtextbook.LearningArc {
	stages := sharedtextbook.DefaultLearningArc().Stages
	details := map[sharedtextbook.LearningStage]struct {
		objective string
		evidence  []string
		recovery  string
	}{
		sharedtextbook.LearningStageActivatePriorKnowledge:     {"说出一个 URL 中的路径，并区分它和处理函数。", []string{"reader-notes:prior-knowledge"}, "回到地址栏例子，只圈出方法和路径。"},
		sharedtextbook.LearningStageExperienceProblem:          {"面对 404 时先列出登记链检查项，而不是猜 handler。", []string{"reader-notes:404-checklist"}, "回到“问题体验”，按方法、前缀、相对路径重排检查顺序。"},
		sharedtextbook.LearningStagePreviewWhole:               {"画出 GET、handle、addRoute 的三步全貌。", []string{"reader-drawing:registration-flow"}, "回到“三步各解决一个疑问”，只画三根箭头。"},
		sharedtextbook.LearningStageWorkedExample:              {"跟随固定快照解释注册和请求查找两个阶段。", []string{"gin-routergroup-get", "gin-routergroup-handle", "gin-engine-add-route", "gin-engine-serve-http", "gin-engine-handle-http-request", "gin-node-get-value"}, "先读 GET、handle、addRoute，再读 ServeHTTP、handleHTTPRequest、getValue。"},
		sharedtextbook.LearningStageGuidedPractice:             {"在提示下补全 api.GET 的处理函数并预测路径。", []string{"reader-code:guided-get"}, "保留函数签名提示，只补全响应逻辑。"},
		sharedtextbook.LearningStageFadeScaffolding:            {"只凭 GET、/orders、404 三个线索选择排查起点。", []string{"reader-notes:faded-diagnosis"}, "重新显示方法、前缀、相对路径三个检查项。"},
		sharedtextbook.LearningStageIndependentTransfer:        {"从零登记 /health 路由并解释它的登记链。", []string{"reader-code:health-route"}, "先写最小路由，再加入路由组前缀。"},
		sharedtextbook.LearningStageExplainAndConnect:          {"用自己的话向同事解释为什么启动阶段需要登记。", []string{"reader-explanation:registration"}, "使用“输入、交给谁、解决什么”三句模板复述。"},
		sharedtextbook.LearningStageSpacedInterleavedRetrieval: {"三天后不看源码画出登记箭头并补全最终路径。", []string{"reader-retrieval:day-3"}, "只回看自己的卡片，再尝试重画。"},
		sharedtextbook.LearningStageAdaptFromEvidence:          {"根据解释、编码和排错表现选择下一项最小练习。", []string{"reader-plan:next-practice"}, "将错误归类为解释、编码或排错，再选对应练习。"},
	}
	for index := range stages {
		detail := details[stages[index].Stage]
		stages[index].Objective = detail.objective
		stages[index].SuccessEvidence = detail.evidence
		stages[index].RecoveryRoute = detail.recovery
	}
	return sharedtextbook.LearningArc{Stages: stages}
}
