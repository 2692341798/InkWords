# InkWords 前端（frontend/）

## 推荐：Docker 一键启动

在仓库根目录执行：

```bash
cp backend/.env.example backend/.env
docker compose --env-file backend/.env up -d --build
```

启动后请通过 `http://localhost` 访问应用。

## 本地开发

```bash
cd frontend
npm install
npm run dev
```

浏览器访问：`http://localhost:5173`

## 测试与构建

```bash
cd frontend
npm test
npm run build
```
