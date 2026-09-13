# 工作台外层滚动修复：2026-09-13

本次为本地真实窗口的可用性修复，不扩展 PRD 的移动端或设备专项验收范围。

## 问题与改动

Gin 工作台的绝对定位无障碍标签、隐藏文件输入和 fieldset legend 没有归属于
工作区定位容器。长页面中这些元素使外层文档高度达到 13,779 px；外层滚动
167.5 px 后，应用顶部被移出窗口，底部出现空白。工作区本身已有内部滚动。

仅在 `frontend/src/index.css` 的 `.app-workspace` 增加 `relative`，使绝对定位
元素随工作区滚动并被其滚动边界包含。没有移除无障碍标签或禁用工作区滚动。

## 实际验证

- 改动前当前窗口：文档 clientHeight 763 / scrollHeight 13,779，外层 scrollTop
  167.5；main 高约 427 px、top 168.21875。外层滚动条存在时 clientWidth 738。
- 开发页面修复后：文档高度 763、外层 scrollTop 0，main position relative、
  top 335.71875；滚动到“用户委托 AI 整书审阅”时 main scrollTop 25,671.5，
  nav top 118，导航仍在窗口内。去掉外层滚动条后 clientWidth 753。
- 通过真实浏览器按钮切换到知识复习页，完成加载并看到题卡入口及历史记录；
  外层高度仍 763、scrollTop 0。没有开始练习或产生掌握证据。
- 本地正式网关刷新、重新打开 Gin 工作台并定位同一深处标题：viewport
  753×763，文档高度 763、外层 scrollTop 0、main scrollTop 25,671.5、
  main top 335.71875、nav top 118，position relative。
- `npm test -- --maxWorkers=2`：80 文件、319 项全部通过。
- `npm run lint` 与 `npm run build`：退出 0。构建仍有现存大 chunk 提示；
  Mermaid 测试环境有 getBBox 降级日志，相关测试通过。

浏览器验收通过 CUA 真实页面语义操作和只读 DOM 几何检查完成。开发页截图
已观察；正式页截图接口本次返回异常小图，不把它作为可交付截图证据。
没有使用另一路截图脚本采集浏览器，也没有据此声称全页视觉或打印验收通过。

## 部署与回退

只从已有本地镜像构建并替换 frontend，构建禁用网络与拉取；服务健康，部署前后
环境、挂载、权限和隔离参数一致。没有重建后端、修改母稿或审批账本。

- 新镜像 `inkwords/frontend:workspace-scroll-v1-20260913`，ID
  `sha256:e5498eff57164d49c02f7846769264b0077fc39bb3f4bb0d3442f72fe9a702cc`。
- 网关 index 与本次构建逐字节相同，SHA-256
  `731b91884a0d163ec8d7f0b4b68342b333665221db022f44eb94326d891a2915`。
- 旧镜像 `inkwords/frontend:asset-font-review-v1-20260913` 保留；回退使用原
  Compose 文件链，最后追加证据目录的 `compose.rollback.yml`，仅更新 frontend。
- 证据目录：`output/real-acceptance/2026-09-13/workspace-scroll/`，含
  `images.json`、`deployment-plan.json`、`deployment.json`、`validation.json`、
  构建/部署日志及 Compose 覆盖文件。不得直接用仅含 image 的覆盖文件启动整套服务。

本次不增加依赖、不调用 Provider、不运行教学代码、不提交或推送。
动手代表章、真实六维学习与延迟保持、独立试学及正文/截图权利缺口仍未完成。
