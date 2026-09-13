# 官方网页的非活动标签页必须保留平台语境

日期：2026-09-10。真实 Go 安装资料导入暴露：v2 正文抽取把 HTML `hidden` 一律排除，导致 `go.dev/doc/install` 只留下默认 Linux 面板，Mac/Windows 安装步骤从证据库消失。与此同时，tab 按钮被正确视为交互控件排除，默认 Linux 内容也失去平台标题。

## 实现

采用已有 `golang.org/x/net/html` DOM，不运行网页脚本、不点击远程控件。v3 只展开满足以下条件的面板：`role=tabpanel`、唯一 ID、`aria-labelledby` 指向唯一且位于可见 tablist 中的 tab，且该 tab 的 `aria-controls` 反向指回面板。只移除该面板根节点的 `hidden`/`aria-hidden`；普通隐藏元素、导航、脚本及不明确的关联仍排除。

从关联 tab 的文字生成平台标题，层级承接 tablist 前的正文标题；后续证据片段保留 `heading_path`，使 Mac 安装说明不会与 Linux 命令混为同一个无条件结论。页面内容仍是不可信来源数据，不能执行其中的命令或指令。

## 不可变与版本

- 新任务使用 `inkwords.official-html.v3`，现有任务 envelope 仍为 task_version 2。
- 保留 `inkwords.official-html.v2` 的原正文规则及 v1 tokenizer。任务携带的解析版本决定重放规则，不按当前默认版本重解释旧任务。
- input hash 与 parsed content hash 均包含解析版本；新导入形成独立 snapshot，不改写原始来源或旧片段。
- 先更新消费者、再更新生产者。旧消费者遇到未知 v3 版本会拒绝，不能静默用 v2 产生“成功但漏内容”的结果。

## 验证与范围

聚焦测试覆盖非活动面板、平台标题、代码块、普通隐藏内容/脚本继续排除、孤立/冲突/错误关联拒绝以及 v2 重放完全一致。共享合同验证旧版兼容、版本绑定与未知版本拒绝；解析 worker 验证 v2/v3 选择。相关服务与架构测试通过。

直接保存的官方 HTML 只用作解析器对照夹具；产品证据必须经正式爬取任务和 core 持久化确认。真实导入、快照及后续教材修订证据见 `output/real-acceptance/2026-09-10/foundation-setup/`。

本实现覆盖标准 ARIA tab 关系。纯脚本异步加载、无关联语义或非标准控件不会被猜测为已导入；仍需显式补入可核对的官方资料。此变化不执行网页脚本，也不扩大允许抓取的域名或路径边界。
