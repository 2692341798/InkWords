package textbook

import (
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"inkwords-backend/shared/platform/teachingartifact"
)

// BuildGinSampleVideoRunbookProjection separates static source capture from
// manifest-controlled execution. The Gin sample has no approved IDE debugger
// sandbox, so its default filming guide must not start teaching code on a host.
func BuildGinSampleVideoRunbookProjection(markdown, contentHash string) (sharedtextbook.VideoRunbookProjection, error) {
	if digest(markdown) != contentHash {
		return sharedtextbook.VideoRunbookProjection{}, fmt.Errorf("video runbook manuscript hash mismatch")
	}
	files, _, err := teachingartifact.ManuscriptGoFiles(markdown)
	if err != nil {
		// A projection failure must not discard an otherwise valid manuscript.
		// Retain a pending guide without pretending that files can be filmed.
		return ginRunbookAwaitingSource(contentHash)
	}
	sourceInput := ginRunbookSourceLocations("main.go", files["main.go"], false)
	testInput := ginRunbookSourceLocations("main_test.go", files["main_test.go"], true)
	projection := sharedtextbook.VideoRunbookProjection{
		Format: "inkwords.video-runbook.v1", Stack: "go",
		ObservationGoal: "教学路由结构、方法隔离测试与隔离验证记录",
		Recommendation: sharedtextbook.DemonstrationToolRecommendation{
			Primary: "Visual Studio Code", Alternative: "GoLand（仅静态查看）",
			Reason:                "用编辑器呈现源码布局，用 InkWords 的批准清单执行验证；当前样章没有 IDE 调试沙箱。",
			ManualCaptureRequired: true,
		},
		Steps: []RunbookStep{
			{
				Tool: "Visual Studio Code", ToolVersion: "采集时记录实际完整版本及操作系统",
				StartState:       "已取得同一母稿修订的代码文件与哈希；教学文件仅供静态查看。",
				Action:           "打开代码目录与 main.go，按下列符号和行号对照正文查看数据结构与函数；这里只定位源码，不推断其运行行为。",
				ShortcutOrMenu:   "文件 → 打开文件夹；资源管理器 → main.go",
				Input:            "母稿 content_hash=" + contentHash + "；" + sourceInput,
				ExpectedView:     "文件树与路由数据结构同屏可读。",
				Narration:        "这是教学代码的静态结构；方法分组与路径节点的解释需结合正文，画面不证明运行时调用栈。",
				CapturePoint:     "ide-source-layout：文件树、源文件名、路由结构与行号",
				Recovery:         "若文件或哈希不符，重新取得该修订的代码；不点击运行、调试或安装建议。",
				CompletionSignal: "截图可读且记录修订、代码哈希、工具版本、系统、时间及采集者身份。",
			},
			{
				Tool: "Visual Studio Code", ToolVersion: "与上一步同一实际版本",
				StartState:       "同一修订的 main.go 已核对。",
				Action:           "打开 main_test.go，逐个定位下列实际测试声明，说明每项测试的输入与断言；未在源码中出现的测试不列为已覆盖。",
				ShortcutOrMenu:   "资源管理器 → main_test.go",
				Input:            testInput,
				ExpectedView:     "实际测试函数与断言可读，文件名与行号可辨认。",
				Narration:        "先看测试要区分什么，再看运行收据；测试源码本身不是通过证据。",
				CapturePoint:     "ide-test-layout：基线与方法隔离断言",
				Recovery:         "若代码不含这些断言，返回正文核对并修订候选稿，不拼入另一修订的测试。",
				CompletionSignal: "截图与同一修订的可复制测试文本一一对应。",
			},
			{
				Tool: "InkWords 教材工作台", ToolVersion: "采集时记录实际应用镜像与运行器摘要",
				StartState:       "教学工件与验证清单已批准，工作台选中相同修订。",
				Action:           "打开代码验证记录；没有当前证据时，通过工作台显式启动批准清单的隔离验证，再等待终态。",
				ShortcutOrMenu:   "章节工作台 → 代码验证 → 验证记录；必要时显式启动验证",
				Input:            "核对 revision_id、工件哈希、清单哈希和 VerificationRun。",
				ExpectedView:     "同一工件的终态、实际工具链、退出码与原始输出；保留失败或不可用状态。",
				Narration:        "测试结果来自这条隔离运行记录。编辑器截图只展示源码，不把二者合称 IDE 调试证明。",
				CapturePoint:     "sandbox-verification：运行标识、工件身份与结构化原始输出",
				Recovery:         "若证据过期或哈希不符，重新预检；沙箱不可用时停止并记录原因，不回退到宿主机。",
				CompletionSignal: "引用实际运行标识与结果；只有当前成功收据支持已验证结论。",
			},
		},
		CaptureChecklist: []string{
			"记录工具完整版本、操作系统、画面尺寸、采集时间、修订与文件哈希。",
			"每个素材绑定 CapturePoint，提供说明、替代文本与待核实的权利状态；保留可复制源码。",
			"明确 human 或 delegated_ai 采集身份；未采集、静态画面、运行观察分别记录。",
			"没有 IDE 断点或调用栈记录时保持未验证，不用静态截图替代运行时观察。",
		},
		ManualCapturePending: true, VerificationStatus: sharedtextbook.ArtifactStatusUnverified,
	}
	return projection, projection.Validate()
}

func ginRunbookAwaitingSource(contentHash string) (sharedtextbook.VideoRunbookProjection, error) {
	projection := sharedtextbook.VideoRunbookProjection{
		Format: "inkwords.video-runbook.v1", Stack: "go", ObservationGoal: "教学文件待核对，尚未生成源码拍摄步骤",
		Recommendation: sharedtextbook.DemonstrationToolRecommendation{
			Primary: "InkWords 教材工作台", Alternative: "编辑器仅核对母稿",
			Reason: "母稿尚不能无歧义地提取教学文件；先修订候选稿。", ManualCaptureRequired: true,
		},
		Steps: []RunbookStep{{
			Tool: "InkWords 教材工作台", ToolVersion: "核对时记录实际应用版本",
			StartState:     "当前母稿已保留，教学文件拍摄待处理。",
			Action:         "核对母稿中 main.go 与 main_test.go 的代码来源标签、完整语法和非生产限制，另建修订候选稿。",
			ShortcutOrMenu: "章节工作台 → 候选稿对比",
			Input:          "母稿 content_hash=" + contentHash,
			ExpectedView:   "明确显示待核对的候选稿与既有批准稿。",
			Narration:      "当前尚未取得可定位的同源教学文件，不能声称已经拍摄或验证。",
			CapturePoint:   "source-review-pending", Recovery: "保留原稿；不要拼接其他修订的代码或执行宿主机命令。",
			CompletionSignal: "新候选稿可提取同源文件后，重新派生拍摄步骤；本步骤不代表采集完成。",
		}},
		CaptureChecklist:     []string{"教学文件待核对；没有源码截图或运行观察证据。"},
		ManualCapturePending: true, VerificationStatus: sharedtextbook.ArtifactStatusUnverified,
	}
	return projection, projection.Validate()
}

// The shared extractor has already validated these bytes. AST positions refer
// to the exported file, not to the surrounding Markdown or a different revision.
func ginRunbookSourceLocations(filename string, source []byte, testsOnly bool) string {
	positions := token.NewFileSet()
	file, _ := parser.ParseFile(positions, filename, source, 0)
	var locations []string
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *goast.FuncDecl:
			name := declaration.Name.Name
			if testsOnly && (declaration.Recv != nil || !strings.HasPrefix(name, "Test") || name == "TestMain") {
				continue
			}
			if declaration.Recv != nil {
				receiver := declaration.Recv.List[0].Type
				if pointer, ok := receiver.(*goast.StarExpr); ok {
					receiver = pointer.X
				}
				if identifier, ok := receiver.(*goast.Ident); ok {
					name = identifier.Name + "." + name
				}
			}
			locations = append(locations, fmt.Sprintf("%s（第 %d 行）", name, positions.Position(declaration.Pos()).Line))
		case *goast.GenDecl:
			if testsOnly {
				continue
			}
			for _, spec := range declaration.Specs {
				if definition, ok := spec.(*goast.TypeSpec); ok {
					locations = append(locations, fmt.Sprintf("%s（第 %d 行）", definition.Name.Name, positions.Position(definition.Pos()).Line))
				}
			}
		}
	}
	return filename + " " + digest(string(source)) + "；定位 " + strings.Join(locations, "、") + "。行号以该文件原字节为准；不要格式化或混用其他修订。"
}
