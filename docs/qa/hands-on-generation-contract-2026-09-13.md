# Gin 动手章生成与服务观测合同

日期：2026-09-13。状态：代码、静态合同回归及真实选择预检通过；尚未部署、生成或运行真实动手章。

## 问题与修改

原生成器没有按 BlueprintChapter.profile 区分代码任务。所有章节的提示词及质量门禁
都要求 routeNode/addRoute/findRoute，且禁止导入 Gin；hands_on 蓝图因此无法生成
真实 Gin 集成章。本轮按冻结章节类型选择合同，保留 concept 的从零实现要求。

- concept 继续解释缩小的分段路由树，保持原结构检查和来源规则。
- hands_on 使用真实 Gin 的 buildRouter、GET/POST /orders，演示文本响应；表驱动
  请求测试覆盖成功、未知路径和错误方法，要求检查状态码及正文。新增静态 AST 检查
  导入、路由注册调用、回环监听、请求执行及响应比较结构，注释中的调用不算实际调用。
- 动手章使用“为什么先验证请求与响应”“把请求连到固定源码”两项专属标题；教学练习
  合同要求围绕 buildRouter 和实际请求，不沿用概念章的树实现练习。共同学习、引用、
  恢复路径、练习与未验证标识门禁保留。
- Profile 来自冻结蓝图，覆盖模型自报值；本地修正与离线重检也使用冻结值。合同未允许
  的 profile 在模型调用前拒绝；固定概念夹具不能冒充 hands_on，其它章节类型暂不支持。
- 提示合同升级至 `inkwords.textbook.sample.v17`，质量合同升级至 `inkwords.sample-quality.v10`；
  缓存与任务输入会按新版本区分。没有改写旧批准、审阅、收据或冻结构建。

## 文件与观测边界

母稿仍仅允许两个顶层、带来源标签的 Go 围栏，对应 main.go/main_test.go，字节原样
进入工件。安装与操作命令使用行内代码说明，不增加 shell 执行入口。提示要求先解释
go mod init、已审核的固定 vendor 包，再执行 go test ./...；缺依赖时返回选择预检。

动手章主程序合同使用 `http.ListenAndServe("127.0.0.1:38080", buildRouter())`。
这是匹配已有隔离浏览器探针的入口要求，不表示服务已经启动。httptest 仅为进程内
检查，不能当作端口或浏览器证据。固定依赖包内 Gin gin.go 第 33/34、738–759 行支持
默认 404/405 响应说明，405 需要显式启用 HandleMethodNotAllowed。

显式依赖选择新增可选 browser_observation，仅接收现有 browser_page 模板、本地路径
和有界预期文本。选择含该项时，候选工件投影在 go_test 后加入对应 browser_page。
执行清单拒绝缺失、重复或不一致的浏览器观测；观测随选择哈希和验证哈希冻结。选择
返回值深拷贝，调用者不能在预览后改动内部预期。修复工具预览现在列出完整 commands。
修改已登记的相同代码树不会覆盖旧清单，需按新候选/工件流程处理。

## 实际验证

证据目录：`output/real-acceptance/2026-09-13/hands-on-generation-contract/`。

- `red.log`：实现前 hands_on 提示仍含“不得 import Gin”，回归失败。
- `profile-tests.log`：概念/动手提示分流、模型伪造 Profile 被覆盖、未允许类型在调用前
  拒绝、缺 Gin 导入/注册/请求/状态或正文断言/正确监听等静态失败用例通过。无函数体
  的 main 声明被安全拒绝，未导致检查器 panic。测试里的 Go 教学示例仅解析，未执行。
- `observation-tests.log`：选择复制隔离、危险 URL 拒绝、浏览器命令投影及缺失检查通过。
- `backend-final.log`：`GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...`
  全量通过，含架构与历史清单兼容检查。gofmt、`git diff --check` 通过。
- 真实空章只读预检返回 validated=true、activated=false、executed=false、registered=false。
  新选择包含 `/orders` 与 `orders-list`，选择哈希为
  `sha256:779a97fed807ae7346936f16898abe04e5a87f4ecc5d90ad0a398a119e6fd03f`。
  依赖清单仍是 `sha256:79a707dfcddc51376f63339b866bb5d21b0922fa50f2a8b49a63c870bd884f5d`。
- `real-selection-rejections.json` 的另一实际项目/变更快照两请求被拒绝；
  `real-selection-state-check.json` 证明项目、章节完整记录与修订/工件/任务数前后相等。
  只读检查器使用已有镜像中的临时进程，没有部署应用服务或执行教学代码。

## 尚未验收

静态比较结构不证明测试有效、覆盖充分或运行通过；样章必须再经过隔离执行与内容审阅。
本轮没有模型调用、实际候选、工件登记、浏览器运行收据或沙箱截图。新版本需要 core-api
与 llm-stream 协同部署后才进入真实生成预检；页面依赖选择和自动投影仍待集成，当前可用
本地明确选择/投影工具。下一步部署并核对新预检、审核蓝图 r6，再取得真实样章和对应的
go_test/browser_page 证据。实际生成仍遵循一次有界授权，不自动重试失败的模型调用。
原个人学习、冷读者、许可、整书审校与三项综合验收继续保留。
