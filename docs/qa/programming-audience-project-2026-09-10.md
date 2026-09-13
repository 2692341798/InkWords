# 有编程基础版：正式项目、真实生成与隔离验收

日期：2026-09-10。用户已授权真实电脑操作、模型调用及委托 AI 决策。
本轮审阅来源为 `delegated_ai`，不代表真人试学或出版社审校。

## 正式项目与冻结输入

通过真实浏览器创建 `Gin 路由机制：有编程基础版`，project
`46ccb5fe-e707-4dac-8c82-ae9e11daff30`，audience `programming`。
章节 `c981b187-3328-4e7a-82be-d50a248f0162`，类型 `source_walkthrough`。
读者会函数、循环、字典、基础单元测试和终端，但不默认会 Go、HTTP 或 Gin 内部结构。

三份 Gin 文件固定提交 `73726dc606796a025971fe451f0aa6f1b9b847f6`，上传前逐字节
哈希与原项目已导入文件相符；没有执行导入仓库。另行导入 Gin HTTP 方法、Go Gin
教程和 Go 安装官方页面，6 项真实解析任务全部成功。安装抓取按已批准的路径前缀
边界保留 `/doc/install/source`、`/doc/install/gccgo`，选用的是 Mac 安装片段。
新项目拥有自己的不可变来源快照，不改变原项目版本。

BookContract `fbe865bd-c7c7-4c83-bd7a-3a8c9edb74fc`、StyleSheet
`da494e57-89a8-4cfc-a378-810d68fc530b`、蓝图
`0b4d19ca-c21c-4f4c-88c0-2b69f154e9bd` 均经页面明确批准。
14 项冻结证据覆盖登记/查找、前缀分裂、官方方法与路径绑定、安装和模块初始化。
预检输入估算 16,487/20,000 Token，预留输出 12,000。

## 模型结果与有界修订

真实任务 `11e601ed-7286-45dd-96d2-f80fe474006f`，DeepSeek `deepseek-v4-flash`，
1 次调用、37,896 ms，输入 13,054 / 输出 9,091 Token；缓存和费用未知，retry 0。
因首次 POST 未释义被 v9 质量门禁拒绝，原稿完整保存在私有收据
`0646e1d707a62fc93aa9f480241747b8f6e5f01d4ba64303f757a0187d5efa19`。

委托 AI 阅读原稿后同步修订正文、可运行代码、六维练习与答案：

- 限定路径字典后写覆盖的条件，纠正“先路径再方法必然丢信息”的说法。
- 纠正 GET 内部调用关系与共享树读取/请求上下文修改的边界。
- 用真实函数值及调用返回值替代处理函数名字符串。
- 使用字符串分段保留原始字符，补空分段等教学边界。
- 增加四种结构比较及三步 Gin 压缩前缀形状推演；明确推演不是 Gin 运行快照。
- 将安装和 Go 语法前提放在代码前，说明完整示例如何转成补全练习副本，避免默认删除模块文件。

离线修订门禁通过；初次权限和内容校验失败日志保留。校验对“以你实际输出为准”
产生的字样误报，通过改为明确引用运行收据的表述消除，没有削弱门禁。
真实浏览器上传修订 JSON，任务 `aa02735c-ddfe-4730-9733-d86f83176ea2` 成功，
`automated_local_correction`，0 次模型调用，retry 0；原模型用量只作为来源保留。

候选 r1 `fcc982ca-4664-4acb-88c5-7067211d1bf1`，批准 r2
`64ea0e77-bb97-4ac4-9af9-98ba1a909ede`，两者正文均为
`a6c74c66c114bdc25d57a8a99ccb6102d6b5711d9fdafe8bc3869b5b85eac5b2`。
委托审批 `8ac35bc0-4c7f-4ed5-b391-4887dd5cfcbd`，八维分数 3/4/3/4/4/3/3/3，
保留长段、重复和练习梯度仍可改善的说明。

## 真实教学代码验证

工件 `7cb95d3c-3127-5afa-8b53-d50a7602a51a`；代码树
`sha256:8d6ae2f7c19d2a68f74355ef7daf0239fdddeab265a0649bd76bd9c1c6eebce9`；清单
`sha256:e7d7711babe04b0e9b1331ce040afe2e435353ab093aa841ff9b439f422b9a76`。
页面启动任务 `5e15929a-85a5-4ad2-93e1-d94ae0e01d19`，第 1 次独立验证，retry 0，
Go 1.26.8 / Bubblewrap / 无网络 / 只读系统目录，8.870 秒成功。
原始输出为 `ok example.com/inkwords/teaching 0.001s`，未截断。
收据 `8106413d-e0ff-41a5-84ee-a3d5337d5b49`，有效至 2026-10-10。

执行代码中的三项测试覆盖错误路径字典基线、实际函数调用、方法隔离、未命中、共享 api
节点、父子共存以及重复斜杠、尾斜杠和中文分段。标准 go_test 输出只含包级通过，
没有声称采集了 verbose 单项日志、实际 HTTP 响应、性能测量或 Gin 内存快照。
批准 r2 通过同正文父修订关联 r1 工件和有效证据，页面已实际显示并截图检查。

## 导出故障、修复与重新验收

章节 Markdown 下载 30,614 字节，与批准正文哈希一致。首次旧“含谱系 ZIP”入口返回
HTTP 200 / 22 字节空包。服务日志证明章节 AST 缺冻结引用，且 handler 在失败后
已经输出 ZIP；既有测试只有不带引用的简单章节，未覆盖这个故障。

新增回归先复现引用缺失、工件晚期失败和未配置时错误 Content-Type 三种情况。
旧章节入口现在对引用稿返回 409 `TEXTBOOK_CHAPTER_REQUIRES_BOOK_BUILD`，指向现有
冻结构建流程；普通打包先在内存完成再写下载头，失败返回 JSON，绝不下载半成品。
沿用已有整书导出的缓冲模式，无新依赖或数据库变更；大包内存占用仍是既有边界。
没有让 export-service 查询可变来源表，也没有绕开 Canonical Book AST 校验。

export-service 全部包及服务架构检查通过，含普通无引用章节成功导出回归；
`git diff --check` 通过。没有重跑无关前端或把这些结果称作全仓测试。
仅部署 `inkwords/export-service:chapter-zip-errors-v1-20260910`，镜像
`sha256:4a3979071cce689af2eb0cf4f93e4d753a68143ca55cbd78f6f37c33b993fffc`。
保留当前 core/runner/frontend、真实 vault、字体、模型与隔离配置。
真实网关复核为 409 / JSON / 无 Content-Disposition，不再返回空包。

真实浏览器冻结构建 `691f94e4-5564-4779-9c2f-2a18f5d39265`，manifest
`sha256:514e5abed8f709ca011fdbe4d109fd4cd3795da20a088d1e40639ccd6488142f`。
新整书审校 ZIP 成功返回 592,955 字节，19 文件；17 个内容哈希及 manifest 自身哈希
均核对通过，AST 正文与批准稿逐字节一致，两份 Go 文件与母稿代码块逐字节一致。
14 项引用被冻结，Markdown 投影已转换内部 evidence 标记；包中有四格式、来源、
质量报告、有效运行快照和代码。冻结/导出时均接受上述同一收据。

本轮仅检查 DOCX/PDF 已真实生成并在清单内，不宣称逐页版式校样通过。当前新构建无
整书审阅和权利条目，不是出版候选；章节委托批准不自动变成整书或真人审阅。
两张桌面截图已查看；未重新测试窄视口、安装流程、视频 Runbook 或截图资产登记。

## 计划与检查点

这是正式 programming 项目代表章，区别于早期独立评测稿；foundation 已有正式样章，
`stack_familiar` 正式代表章仍未完成。三 audience 整项、六维个人学习闭环、整书试学
和出版验收继续未勾选。没有新增个人作答、延迟保持或真人记录。
原 foundation 项目的 workspace 与 chapter workspace 在本輪前后 JSON 完全一致。

原始证据：`output/real-acceptance/2026-09-10/programming-project/`。
桌面图片：`output/playwright/programming-project-20260910/`。
后续部署应从本目录 `runtime-build/deploy.py` 追加层，避免退回旧 export 镜像。
