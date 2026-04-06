# 容器化部署与环境规范

## 1. 核心理念 (Docker-First)
- **环境一致性优先**：在本地开发、测试、以及生产环境部署时，必须首选基于 Docker 与 Docker Compose 的容器化方案，确保跨平台运行环境的一致性。
- **一键拉起**：所有微服务组件（前端、后端、数据库等）都必须能够通过一条 `docker-compose up -d --build` 命令一键拉起并成功互通。

## 2. 容器化设计规范
- **Monorepo 支持**：在项目根目录维护统一的 `docker-compose.yml` 文件，通过上下文路径分别构建前端 (`frontend/Dockerfile`) 和后端 (`backend/Dockerfile`) 镜像。
- **多阶段构建 (Multi-Stage Build)**：
  - **Go 后端**：必须使用多阶段构建（如 `golang:1.21-alpine` 编译，`alpine` 或 `scratch` 运行），以大幅缩减最终镜像体积。
  - **React 前端**：使用多阶段构建（如 `node:18-alpine` 安装依赖并执行 `npm run build` 构建产物，然后使用 `nginx:alpine` 代理并挂载静态文件）。
- **网络与通信**：容器间通过 Docker 内部网络（如 `inkwords-network`）通信，后端连接数据库应使用服务名（如 `postgres`）而非 `localhost`。

## 3. 环境变量与安全
- **隔离配置**：绝不将敏感信息（如 `DEEPSEEK_API_KEY`、数据库密码等）硬编码进 `Dockerfile`。所有配置必须通过 `.env` 文件或 Docker Compose 的 `environment` 字段注入。
- **本地模板**：根目录下必须维护 `.env.example`，作为启动容器的配置模板，指导新开发者如何配置环境。

## 4. 数据持久化 (Volumes)
- **数据库存储**：PostgreSQL 容器必须挂载 Docker Volume（如 `./data/postgres:/var/lib/postgresql/data`），防止容器销毁后丢失用户的博客历史与核心配置数据。

## 5. 开发调试建议
- **热重载支持**：当需要频繁修改代码时，鼓励开发者在本地直接运行前端（`npm run dev`）或后端（`go run cmd/server/main.go`）。
- **混合运行**：允许“部分容器化”，例如仅使用 `docker-compose up -d postgres` 拉起数据库，前后端则在本地原生环境中运行以获取最佳的开发与调试体验。