# Tasks

- [x] Task 1: 新增前端登录页面
  - [x] SubTask 1.1: 创建 `frontend/src/components/Login.tsx`，包含 GitHub 授权登录按钮。
  - [x] SubTask 1.2: 在 `App.tsx` 中拦截未登录状态，渲染 `Login` 页面。
  - [x] SubTask 1.3: 处理 OAuth 回调，保存 Token 并重新加载数据。
- [x] Task 2: 修复 Editor.tsx 无法编辑的问题
  - [x] SubTask 2.1: 修复 `Editor.tsx` 中 `useEffect` 状态覆盖逻辑，只在 `selectedBlog.id` 改变时重置 `title` 和 `content`。
- [x] Task 3: 增强 Generator.tsx 上传功能与状态显示
  - [x] SubTask 3.1: 在 `Generator.tsx` 新增文件上传（PDF、Word、Markdown）交互区域。
  - [x] SubTask 3.2: 增加调用后端文件解析接口的逻辑，支持上传触发流式生成单篇博客。
  - [x] SubTask 3.3: 改进分析代码仓库时的 Loading 动画，增加更清晰的进度或步骤提示，避免让用户以为卡死。
- [x] Task 4: 修复 Sidebar.tsx 中生成卡片无法点击的问题
  - [x] SubTask 4.1: 修改 `Sidebar.tsx` 中 `streamStore.outline` 渲染逻辑，为生成的章节卡片添加 `onClick` 事件。
  - [x] SubTask 4.2: 点击生成中的卡片时，调用 `selectBlog` 将其在右侧 `Editor` 打开，实现边生成边预览。

# Task Dependencies
- [Task 2] 和 [Task 4] 可以并行。
- [Task 3] 独立于其他任务，可以在 [Task 1] 完成后或并行开发。
