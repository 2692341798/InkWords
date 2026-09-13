# r11 真实 PDF 与 ZIP 运行验收

日期：2026-09-10。状态：两个真实导出路径已打通；PDF 版面尚未通过审校。

## 实际修复

export-service 的 Chromium 直接运行环境与 course-runner 的嵌套 Bubblewrap 浏览器环境不同。固定页 spike 先定位默认 seccomp 的 namespace 拒绝，再定位新 namespace 内 `chroot` 拒绝；派生 Moby profile 只增加 Chromium 建立沙箱所需四项调用。非 root、只读根、cap-drop ALL、no-new-privileges、无网络的一次性容器中生成 PDF，renderer 进程的独立 namespace 和第二层 seccomp 过滤器也已采集。未使用 `--no-sandbox`、privileged、cap-add 或 Docker socket。

部署 `docker-compose.textbook-pdf.yml` 后，实际 export-service healthy，UID/GID 10001、cap-drop ALL、no-new-privileges、只读根，1 GiB 内存、1.5 CPU、192 PID；既有工件和 Obsidian 挂载保留。部署容器使用的 seccomp 结构与仓库文件逐项相同，见 `deployed-runtime.json`。部署后的服务仍有其数据库与 Obsidian 网络需求，不能把一次性无网络 probe 的条件冒称整个 export-service 无网络。

ZIP 随后真实返回 409，日志定位为教学工件目录 permission denied：冻结工件的既有权限为 root:20001、目录 0750，而 exporter 缺少 reader group。已在基本 Compose 为 exporter 增加既有组 20001；工件的权限和只读挂载不变，没有 chmod、chown 或复制替代工件。

## 权威输出

输出根：`output/real-acceptance/2026-09-10/pdf-sandbox-spike/`。
均绑定 BookBuild `8aa50b6b-45b3-477d-aece-4bbe1b164089`，manifest
`sha256:4164d754fc06350bf2a4bbb08e5d7b9281e8d7ba55a41d13af7075b9869e960b`。

| 产物 | 真实结果 | SHA256 |
| --- | --- | --- |
| Gin-r11-服务端校样.pdf | HTTP 200；1,025,619 字节；Chromium 生成的 19 页 A4 PDF | b73710accbfd8f18a17fcdf43aa2721749e0c19d59b72f17d8ce8e305e4f797c |
| Gin-r11-真实审校包.zip | HTTP 200；542,632 字节；17 个文件 | f14a329b57e7c7d48b0655cd5d14ac0732608d5022475f2960a07e63d08f493c |

ZIP CRC、manifest.sha256 及 manifest 内每个文件 SHA256 全部核对通过；包内 Markdown 与此前 r11 独立导出逐字相同。包中包括 Canonical Book AST、三种文本/排版投影、PDF 运行日志、原冻结教学代码三文件、权利/审阅/预检记录和 BookBuild manifest。publication_candidate 仍 false，真人审阅 0，委托记录包含原 layout v1；这些未知/未通过状态没有被改成通过。详细事实和 HTTP 头在 `export-facts.json`、`pdf-headers.txt`、`zip-final-headers.txt`。

## 测试与版面判定

新增运行配置测试先真实失败，再通过：限定策略差异、禁止高权限/关闭沙箱配置，固定 exporter 只读工件组。相关 Peer/Runtime/Textbook 架构检查通过。实际 Linux 测试二进制在同一镜像、非 root、只读、无网络、同一 profile 的一次性容器运行 `TestChromiumBookPDFRendererRendersCanonicalASTWhenExplicitlyConfigured`，通过，未跳过；见 `real-chromium-contract.log`。只读运行容器的 tmpfs 不允许执行测试二进制，因此没有修改 tmpfs 执行权限，而将可信测试二进制作为独立只读挂载用于一次性测试。真实 API 导出在生产服务路径执行。

已渲染全部 19 页为 PNG，目前只查看第 1、8、19 页便确认硬性版面缺口：没有导航目录和页码，一级标题后大片留白，长代码与来源条目跨页拆开。故不声明 19 页全部视觉验收，不向用户交付为最终出版 PDF，不把 ZIP 下载成功视为出版质量达标。下一步在相同冻结 AST 上改进 PDF 导航、分页和页码，完成全部页面复核后追加 layout 复审。旧校样与审阅保留。

本轮无 Provider 调用、无依赖安装/下载、无母稿改写、无学习作答写入、无 Git 提交。课程 browser_page 沙箱问题不在本切片解决范围。完整计划仍未完成。

真实浏览器已追加 layout v2：`e6b87ad5-4edf-4f0f-b171-ac2f8c29c859`，HTTP 201，`needs_revision`、2/4。随后 GET 确认两条历史记录都保留、manifest 未变、preflight false。v2 将阻碍从“PDF 不可用”更新为实际 PDF 版面缺口；不复写 v1。当前 ZIP 是 v2 保存前的快照，内含 v1，下一次导出将带最新追加记录。最终 `git diff --check` 通过。

运行配置依据、限制及回退方式见 `docs/decisions/textbook-pdf-runtime.md`。以后重新创建 export-service 时必须保留该 opt-in 覆盖层，否则会恢复原未启用 PDF 的行为。
