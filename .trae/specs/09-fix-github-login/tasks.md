# Tasks
- [x] Task 1: 修改后端 `OAuthCallback` 重定向逻辑
  - [x] SubTask 1.1: 导入 `os`, `fmt`, `net/url` 包
  - [x] SubTask 1.2: 在 `OAuthCallback` 中获取 `FRONTEND_URL`（默认 `http://localhost:5173`）
  - [x] SubTask 1.3: 当回调发生错误时，使用 `c.Redirect` 重定向到前端并附加 `error` 参数
  - [x] SubTask 1.4: 当回调成功时，使用 `c.Redirect` 重定向到前端并附加 `token` 参数
- [x] Task 2: 修改前端 `Login.tsx` 组件以捕获错误信息
  - [x] SubTask 2.1: 从 `react` 中导入 `useEffect`
  - [x] SubTask 2.2: 添加 `useEffect` 钩子解析 URL 中的 `error` 参数，调用 `setError` 进行提示，并使用 `window.history.replaceState` 清除 URL 参数

# Task Dependencies
无
