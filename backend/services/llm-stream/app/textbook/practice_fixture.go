package textbook

import sharedtextbook "inkwords-backend/shared/kernel/textbook"

// ginPracticeFixture supplies concrete authored exercises only for the fixed
// local Gin fixture. Real provider candidates must generate their own set.
func ginPracticeFixture() sharedtextbook.PracticeSet {
	type exercise struct {
		mode                      sharedtextbook.LearningTaskMode
		prompt, variation, answer string
		criteria                  []string
		hints                     []string
		refs                      []string
	}
	exercises := []exercise{
		{sharedtextbook.LearningTaskExplain,
			"不看正文，用自己的话说明：启动时登记 GET /api/orders 与请求到达时查找该路由，各自经过哪些步骤？画出两条调用链并解释它们为何不能混为一谈。",
			"把 /api/orders 改成 /v2/health，说明哪些登记输入改变、哪些请求查找机制保持不变。",
			"登记链是 GET → handle → addRoute；handle 计算路径并合并处理函数链。请求链是 ServeHTTP → handleHTTPRequest → getValue，按方法选树、按路径查找。登记发生在配置阶段，请求到来不会重新执行路由登记；缩小模型省略生产路由的复杂匹配。",
			[]string{"准确区分登记链与请求链中的函数职责。", "同时说明方法、路径、处理函数链三个输入。", "解释先登记的内容为什么可以被后续请求查到。", "说明类比及缩小路由树不覆盖的生产能力。", "使用自己的话和两条清楚的调用链，不只罗列函数名。"},
			[]string{"先问自己：哪条链发生在服务开始处理请求之前？", "把 GET、handle、addRoute 分为一组，另从 ServeHTTP 开始找请求链。", "对每条链标出方法、路径和处理函数的输入与输出，再说明两条链共享的树。"},
			[]string{"gin-routergroup-get", "gin-routergroup-handle", "gin-engine-add-route", "gin-engine-serve-http", "gin-engine-handle-http-request", "gin-node-get-value"}},
		{sharedtextbook.LearningTaskComplete,
			"在本章 main.go 的 findRoute 中补全 node := r.trees[___] 与 for _, part := range splitPath(___) 两处空白，并补上树不存在时的返回。写明为什么不能只用路径查找。",
			"给同一路径换一个尚未登记的方法，补充一条证明它不会命中的测试。",
			"两处依次使用 method 与 path；方法树不存在或某个路径分段不存在时返回 nil, false。只有找到完整路径且节点有处理函数才算命中。测试应区分同方法同路径命中、方法不同不命中以及路径不同不命中。",
			[]string{"空白分别使用 method、path，缺失树时安全返回。", "覆盖树不存在、分段不存在及没有处理函数的情况。", "提交本次 main.go 的受控执行证据；没有证据则未知。", "测试命中、错误方法、错误路径三种情况并提供运行记录。", "解释按方法分树避免把不同方法的请求混在一起。"},
			[]string{"回想路由登记时用了哪两个查找条件。", "第一处决定选择哪棵树，第二处决定沿树走哪些分段。", "对照 addRoute 的参数顺序，并检查访问 node.children 前 node 是否为 nil。"},
			[]string{"gin-engine-handle-http-request", "gin-node-get-value"}},
		{sharedtextbook.LearningTaskReproduce,
			"合上教学实现，从空目录创建 Go 模块和 main.go、main_test.go。独立实现按方法分树的 routeNode、addRoute、findRoute，并登记 /orders 与 /health 两条路径；不得导入 Gin 代替机制实现。",
			"让 /orders 和 /orders/new 共用前缀仍能分别找到不同处理函数，并增加测试。",
			"方法到根节点的映射保存各方法树，每个 routeNode 保存子分段与可选处理函数。登记时逐段创建节点，查找时逐段访问并在缺失处返回失败。前缀节点可以同时保存处理函数和子节点；从空目录先初始化模块，再运行测试。",
			[]string{"独立写出分方法的树、登记和查找，未调用 Gin 实现。", "处理共用前缀、缺失路径和节点无处理函数的情况。", "提供本次提交工件的隔离执行证据。", "测试两条路径、共用前缀及错误方法，结果需要可核验运行记录。", "说明数据结构与登记、查找机制的对应关系及非生产边界。"},
			[]string{"先画出方法、根节点、路径分段和处理函数的关系。", "将登记和查找分开写；一个负责按需创建，一个只读取已有节点。", "先让一段路径工作，再加入多段路径和前缀节点同时含处理函数的情况。"},
			[]string{"gin-engine-add-route", "gin-engine-handle-http-request", "gin-node-get-value"}},
		{sharedtextbook.LearningTaskTransfer,
			"在自己的缩小路由器中，为 /orders 同时登记 GET 与 POST（提交方法），让它们返回不同标识。说明现有结构是否需要改变，并用测试证明两种方法不会互相覆盖。",
			"把同样的需求迁到 /v2/orders；保留旧路径，验证四种方法与路径组合彼此独立。",
			"如果根节点已经按方法分组，就不需要更换树结构：分别在 GET 和 POST 的树中登记相同路径。若原实现只按路径保存处理函数，则必须先增加方法维度。四个组合应分别查到对应标识，未登记组合应失败。",
			[]string{"两种方法同路径可以返回各自处理函数，不会覆盖。", "同时保留原路径和新前缀的四种组合及未登记情况。", "提供修改后本次工件的隔离执行证据。", "通过测试覆盖方法隔离、路径隔离和未登记组合。", "解释复用原树结构或补充方法维度的选择理由。"},
			[]string{"先检查根节点的索引是不是只有路径。", "画出两棵方法树，让相同路径分别位于各自树内。", "为每个组合使用不同返回标识，再逐项检查查找方法与路径。"},
			[]string{"gin-routergroup-get", "gin-engine-add-route", "gin-engine-handle-http-request"}},
		{sharedtextbook.LearningTaskDiagnose,
			"故障现场：路由组前缀为 /api，登记相对路径 /orders，客户端却访问 GET /orders 并收到 404。处理函数中的断点没有触发。请提出假设、选择观察点、定位根因并验证修复，不要先改处理函数。",
			"客户端已改成 GET /api/orders 仍失败时，列出接下来要核对的登记状态与方法条件。",
			"先比较客户端方法和完整路径与登记结果。handle 计算出的完整路径为 /api/orders；原请求缺少前缀，因此匹配失败且处理函数不会进入。修复请求路径或明确调整登记配置，再观察正确路径命中、错误路径仍不命中；后续检查方法和登记代码是否执行。",
			[]string{"提出路径前缀、方法及是否登记等可检验假设。", "选择登记后的完整路径与请求方法/路径作为观察点。", "用来源位置和实际观察区分事实、预期与推断。", "解释缺少 /api 如何导致匹配失败且断点不触发。", "提供修复后正确路径命中和错误路径不命中的运行证据。"},
			[]string{"断点没触发意味着故障可能发生在处理函数之前。", "把路由组前缀与相对路径拼起来，与客户端路径逐字符比较。", "先验证完整路径，再查方法和启动时登记是否执行；保留错误路径的反例。"},
			[]string{"gin-routergroup-handle", "gin-engine-handle-http-request", "gin-node-get-value"}},
		{sharedtextbook.LearningTaskRetain,
			"至少间隔三天，不看原文重画登记链与请求查找链，并解释为什么同一路径可以属于不同方法。随后用自己的话诊断“登记 /v2/health，却请求 /health”的失败。",
			"把方法和路径各改动一次，预测是否命中，并说明验证预测需要哪些观察。",
			"登记链整理方法、完整路径与处理函数并写入树；请求链按方法选择树后查找路径。不同方法树可以含有相同路径。前缀不匹配会导致查找失败；预测只是推断，仍需日志或受控运行验证。延迟任务的作答时间必须晚于此前练习。",
			[]string{"无提示正确复述登记和请求查找的主要机制。", "同时覆盖方法、完整路径、处理函数及失败情况。", "能解释前缀不匹配为何导致查找失败。", "区分预测和已观察证据，不把当日记忆当延迟保持。", "用自己的话重新表达，并指出仍需复习的部分。"},
			[]string{"先从“服务开始之前”和“请求到达之后”两个时间点回忆。", "分别回想配置入口和请求入口，再补充方法与路径的流向。", "对每条链写下保存或读取的数据，并用缺少前缀的反例检查解释。"},
			[]string{"gin-routergroup-handle", "gin-engine-add-route", "gin-engine-serve-http", "gin-engine-handle-http-request", "gin-node-get-value"}},
	}
	set := sharedtextbook.PracticeSet{Version: sharedtextbook.PracticeSetVersion}
	for _, item := range exercises {
		task := sharedtextbook.PracticeTask{ID: "gin-routing-" + string(item.mode), Mode: item.mode, Prompt: item.prompt, Variation: item.variation, ExpectedAnswer: item.answer, EvidenceIDs: item.refs}
		for index, criterion := range sharedtextbook.PracticeRubricDimensions(item.mode) {
			criterion.Description = item.criteria[index]
			task.Rubric = append(task.Rubric, criterion)
		}
		for index, hint := range item.hints {
			task.Hints = append(task.Hints, sharedtextbook.PracticeHint{Level: index + 1, Text: hint})
		}
		if item.mode == sharedtextbook.LearningTaskRetain {
			task.MinDelayHours = 72
		}
		set.Tasks = append(set.Tasks, task)
	}
	return set
}
