# Tasks

- [x] Task 1: 升级 User 数据模型
  - [x] SubTask 1.1: 在 `backend/internal/model/user.go` 的 `User` 结构体中增加 `PasswordHash string` 字段，并在 JSON 中忽略它（`json:"-"`）。
- [x] Task 2: 增加密码工具与服务端逻辑
  - [x] SubTask 2.1: 在 `backend` 项目中引入 `golang.org/x/crypto/bcrypt` 包。
  - [x] SubTask 2.2: 在 `backend/internal/service/auth.go` 中实现 `Register(email, username, password)`，检查邮箱是否已存在，生成密码的哈希并插入数据库。
  - [x] SubTask 2.3: 在 `backend/internal/service/auth.go` 中实现 `Login(email, password)`，比对密码哈希，并使用 `jwt.GenerateToken` 签发 JWT Token。
- [x] Task 3: 提供注册/登录 API 接口
  - [x] SubTask 3.1: 在 `backend/internal/api/auth.go` 中新增 `Register` 和 `Login` 的处理函数，处理传入的 JSON 数据。
  - [x] SubTask 3.2: 在 `backend/cmd/server/main.go` 中将上述路由绑定至 `POST /api/v1/auth/register` 和 `POST /api/v1/auth/login`。
- [x] Task 4: 设计并实现高品质的登录页面
  - [x] SubTask 4.1: 在 `frontend/src/components/Login.tsx` 中移除或弱化原有的 GitHub 登录按钮，实现一个美观、极简或极具设计感（符合 frontend-design 规范）的登录与注册表单。
  - [x] SubTask 4.2: 实现平滑的在“登录”与“注册”表单之间切换的交互动画。
  - [x] SubTask 4.3: 对接后端的 `/api/v1/auth/login` 和 `/api/v1/auth/register`，成功后将 Token 保存至 `localStorage`，触发重载或状态更新，进入工作台。

# Task Dependencies
- [Task 2] 依赖 [Task 1]
- [Task 3] 依赖 [Task 2]
- [Task 4] 依赖 [Task 3]