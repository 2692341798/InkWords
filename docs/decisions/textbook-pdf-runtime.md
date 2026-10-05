# 本地 Chromium 教材 PDF 运行配置

日期：2026-09-10。状态：固定页隔离 spike 与真实 PDF/ZIP 导出通过，PDF 版面仍需修改，见 `docs/qa/textbook-pdf-zip-runtime-2026-09-10.md`。

export-service 直接启动 Chromium，不在 course-runner 的 Bubblewrap 内层；不能把后者的 `/proc` 嵌套问题直接当作导出 PDF 的根因。当前 Alpine Chromium 124.0.6367.78 在 Docker 默认 seccomp 下不能建立 namespace；仅放行 namespace 建立后，真实错误进一步定位为 `sys_chroot("/proc/self/fdinfo/")` 被拒绝。

从仓库已有 Moby 默认策略派生独立 Chromium profile，仅增加 `clone/setns/unshare/chroot`，保留其它默认规则。`chroot` 用于 Chromium 在新 user namespace 中建立自身沙箱，外层仍 cap-drop ALL；不增加 SYS_ADMIN 或 SYS_CHROOT，不允许 mount/pivot_root，也不改变课程运行器策略。官方 [Playwright Docker 文档](https://playwright.dev/docs/docker)说明非 root Chromium 的 namespace 配置；[Chromium Linux sandbox](https://chromium.googlesource.com/chromium/src.git/+/refs/heads/main/sandbox/linux)说明其分层隔离；最终配置以本机实测和固定 profile 为准。

一次性固定页容器无网络、无业务/知识库挂载、非 root、只读根、no-new-privileges、cap-drop ALL，限制 CPU/内存/PID/tmpfs。正常沙箱参数下生成 660 字节空白 PDF。进一步采集进程：renderer 的 NSpid 有三层、独立 user/PID/network namespace、CapEff=0、NoNewPrivs=1、Seccomp=2、Seccomp_filters=2；浏览器父进程只有外层一层过滤器。Chromium 自建的沙箱辅助进程在内部 user namespace 有临时权限，不能将此误称外层 privileged。证据在 `output/real-acceptance/2026-09-10/pdf-sandbox-spike/`。

显式启用 `docker-compose.textbook-pdf.yml`，保留默认 Compose 的 opt-in 行为。此覆盖层限制 export-service 根文件系统与进程资源，直接指定真实 Chromium 二进制，避免发行版启动包装器从环境追加参数。服务既有数据库、工件与 Obsidian I/O 保留；母稿生成的 HTML 自带禁网 CSP，原始 HTML/脚本不从 Markdown 放行。这是可信导出程序的沙箱运行配置，不是任意代码执行授权，也未使 course-runner 的 browser_page 可用。

本机无新依赖或镜像下载，沿用现有已部署镜像。新的 Chromium/image/runtime 必须重做固定页进程证据及实际整书校样，不能仅依据该 profile 的存在宣称安全或排版通过。当前 Chromium 版本事实不等于对其全部漏洞的安全认证。

恢复之前的运行条件时，移除该覆盖层并重新创建 export-service；保留持久卷和所有 BookBuild/审阅记录。默认未配置 TEXTBOOK_CHROMIUM_BIN 时仍 fail-closed。无需数据库迁移。

## 受限字体 profile

PDF 覆盖层同时启用 `TEXTBOOK_PDF_FONT_PROFILE=inkwords.pdf-font-source-set.v1`。exporter 镜像提供
`/usr/share/fonts/inkwords-reviewed/` 的四个固定链接；运行时核对精确字节、独立配置和前后清单。
禁止继承 `FONTCONFIG_SYSROOT`，不把空值或 `/` 当作可靠的清理方式：当前 Chromium 对照中前者
无法读取配置，后者造成中文回退方框。不要通过增加沙箱权限修复字体问题。
缺字体、版本哈希改变、规则包含外部来源或正文候选不覆盖字符时失败，需重新核对部署/字体。

来源约束与同构建有效字体权利记录共同满足时，仅 PDF 字体门禁可通过。含图片/SVG等可能携带
字体的载体仍阻断此门禁，自动字符检查不代替校样。详见 `docs/qa/closed-pdf-font-profile-2026-09-10.md`。
恢复旧 exporter 时须同时撤回新增字体 profile 环境值，保留既有 Chromium 沙箱覆盖参数；
旧报告无闭合来源证据，字体出版门禁继续失败。无数据库迁移。
