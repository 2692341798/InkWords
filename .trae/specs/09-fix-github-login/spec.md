# Fix GitHub Login Spec

## Why
当前用户通过 GitHub 授权登录后，后端 `OAuthCallback` 接口直接向浏览器返回 JSON 数据，而没有进行重定向。这导致用户在授权完成后卡在显示 JSON 的空白页面，无法正确带回 Token 并返回到前端系统，从而导致无法使用 GitHub 登录方式。

## What Changes
- 修改后端 `internal/api/auth.go` 中的 `OAuthCallback` 接口，从读取环境变量 `FRONTEND_URL` 获取前端地址（如果未设置则回退到 `http://localhost:5173`），并将 Token 或错误信息作为 URL 查询参数，重定向回前端页面。
- 修改前端 `frontend/src/components/Login.tsx`，在组件挂载时通过 `useEffect` 监听 URL 参数中的 `error` 信息，若存在则提示给用户，并清空 URL 参数。

## Impact
- Affected specs: 登录认证流程
- Affected code:
  - `backend/internal/api/auth.go`
  - `frontend/src/components/Login.tsx`

## MODIFIED Requirements
### Requirement: 修复 GitHub 登录流程
**后端重定向**：
- 当用户完成 GitHub 授权，后端接收到回调时，若失败需重定向回前端并带上 `?error=` 参数。
- 若成功完成，需生成 JWT Token，重定向回前端并带上 `?token=` 参数。
**前端处理**：
- 前端 `Login` 组件需要能捕获到 `error` 参数，解析后显示给用户，并在展示后清除该 URL 参数。
