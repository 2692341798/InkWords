# r17 权利登记与试学证据范围

2026-09-13，用户委托 AI。当前构建 `9c4dfc67-532a-46f6-906a-80a2a7876e67`，
冻结 manifest `sha256:0dc7bedd7b784b3d399258da513dd00de9de91cd6117c2a96dfb3ef2551ed730`。
本轮没有修改母稿、产品代码、部署或依赖，没有 Provider 调用、教学执行或个人成绩写入。

## 实际登记及页面验收

通过本地受支持 API 登记当前五个必需权利主体，均为 `pending`：r17 正文、r16 代码工件、
三张原字节复用截图。记录包含生成/编辑来源、固定来源版本、文件哈希、旧截图身份与当前绑定，
并明确尚未确认的引用/改写范围、署名及界面/商标/字体复用范围。没有由采集授权推导出版许可。
后续补证使用现有版本化 amendment 流程，不覆盖原件。

三张本地原图与审校 ZIP 的对应 PNG 字节逐一相同；固定 Gin LICENSE 本地原文哈希重新核对为
`03458b6d5828e1be1127ca2adf122572eb574fc47b56190c3b38203b8b2a98d0`。
当前包已有代码 `THIRD-PARTY-NOTICES.txt` 和整书 `publication-notices.txt`，不能沿用早期
“包中没有声明”的判断，但已有声明也不能替代实际适用范围核对。本轮未联网作法律来源更新，
不对未核对的适用范围给出许可判断。

真实工作台刷新恢复五项“已登记，待补证”。随后通过真实表单保存：

- rights v1 `a1688cc8-b4db-4dcf-b1cb-87d6a5c2b805`：需要修改，1/4。
- reader_trial v1 `c06d9d2e-bbf6-4117-b4e5-b28c45cf1bf7`：尚未评估，未评分。

读者试学记录只说明当前构建未取得独立冷读者证据；不将 AI 自学性复审、WPS 校样或原示范
go_test 收据当作新迁移题通过、个人掌握或延迟保持。现有六条审阅逐字段保留，共八条 AI、零真人。

## 字体阻断的真实原因

当前 `book.pdf.font-evidence.json` 已包含受限 `source_set`：四个固定字体文件、正文字符
由实际观测候选覆盖、页脚字符由受限来源覆盖、打印前后字体清单稳定。
`body_observation_status=partial` 与 `print_furniture_status=unobserved` 是原设计保留的
DOM 观测边界，不代表受限来源功能丢失。详见此前 `closed-pdf-font-profile-2026-09-10.md`。

仍有两个独立缺口：当前构建尚无四个文件的独立 font 权利项；正文含三张截图，
`PublicationReadyForBook` 中 `hasOnlyTextFontSurfaces` 明确拒绝图片等未独立核对的字体载体。
Fontconfig 的封闭集合只约束渲染文字，不能证明截图内已经绘制的字体来源。
因此不能复制旧纯文字构建的通过结论，也不能仅删除图片检查以放行。

可继续开发的最小切片是资产字体范围证据：绑定冻结 asset ID、字节哈希、载体类别、审阅来源、
证据引用及有效权利版本；区分截图内文字与当前渲染器字体。需覆盖缺记录、错误哈希/构建、
pending、过期版本与未支持载体的拒绝测试，明确 PNG 内文字也需单独权利审阅。
这是后续方向，当前尚未实现或宣称通过。

## 导出与剩余事项

新审校 ZIP SHA-256 `81a335f00e5f85ba6b9d78f3137923da65d2728f4b11ab5af8114309dc97c4d5`。
24 项内容哈希及 manifest 自身哈希通过；权利原件/有效项/账本和八条审阅与 API 完全一致。
冻结清单、Canonical Book AST、Markdown、代码、图片、Runbook、声明及零真人记录逐字节不变。
PDF/DOCX 本轮未重新逐页校对；之前 r17 全页证据保留，不伪称本轮重复验收。

API 16 项、ZIP 18 项阻断，`publication_candidate=false`。计数增加来自新增明确的 pending
项目及具体未通过审阅，不能按条数判断质量退步或伪造完成。ZIP 还包含字体和构建状态门禁。
原计划三项综合验收仍未勾选；hands_on 批准稿、其真实 Gin 验证/观察、个人六维学习及冷读仍缺。

证据目录：`output/real-acceptance/2026-09-13/r17-rights-review/`。
登记脚本写入成功后的首个验证因误用响应键 `delegated_ai_reviews` 报 KeyError；改为实际
`delegated_reviews`，仅重新 GET 验证，未重复提交五条记录。一次 UI 脚本语法解析失败未执行动作；
修正后提交仅新增两条预期记录。完整事实见 `verified-facts.json`。
