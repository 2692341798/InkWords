# InkWords 前端（frontend/）

## 推荐：Docker 一键启动

在仓库根目录执行：

```bash
cp backend/.env.example backend/.env
docker compose --env-file backend/.env up -d --build
```

启动后请通过 `http://localhost` 访问应用。

## 本地开发

先在仓库根目录启动微服务与统一网关：

```bash
FRONTEND_PORT=8081 \
FRONTEND_URL=http://localhost:5173 \
DOCKER_GITHUB_REDIRECT_URL=http://localhost:5173/api/v1/auth/callback/github \
docker compose --env-file backend/.env up -d --build
```

然后运行 Vite：

```bash
cd frontend
npm install
INKWORDS_GATEWAY_ORIGIN=http://localhost:8081 npm run dev
```

浏览器访问：`http://localhost:5173`

Vite 只代理 `/api` 与 `/uploads` 到 `INKWORDS_GATEWAY_ORIGIN`；具体微服务分流由 Nginx 负责。浏览器不会直接访问任何后端容器或独立端口。

## 测试与构建

```bash
cd frontend
npm test
npm run build
```
