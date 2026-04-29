# 导出到 Obsidian（通过 Local REST API）架构设计

## 1. 背景

InkWords 需要将生成的博客/系列内容写入 Obsidian 的个人知识库，并遵循 Karpathy LLM Wiki Pattern 的目录结构与索引同步规则（`sources/`、`concepts/`、`entities/`、`index.md`、`hot.md`、`log.md` 等）。

此前方案通过 Docker volume 直接写入宿主机 vault 目录完成导出。本方案升级为：**后端容器内同步调用 `obsidian-local-rest-api`**，以获得更强的 Obsidian “应用层能力”（命令面板、Dataview、active file、精准 PATCH 等），但这些能力仍只在后端内部使用，不对前端暴露。

## 2. 目标与范围

### 2.1 目标

- 保留现有「导出到 Obsidian」入口（按钮/后端 API），导出链路改为后端调用 Obsidian Local REST API。
- 继续生成符合知识库规范的 Markdown（YAML Frontmatter + 正文），并同步更新 `index.md`、`hot.md`、`log.md` 及各目录 `_index.md`。
- 所有导出动作在一次 HTTP 请求中同步完成，完成后返回成功/失败。

### 2.2 不做

- 不将 Obsidian API 直接暴露给前端或外部服务调用。
- 不实现异步任务队列与进度流（SSE），保持最小改动。
- 不实现事件监听或状态订阅（REST 请求-响应即可）。

## 3. 总体架构

### 3.1 组件

- Obsidian Desktop + Community Plugin：`obsidian-local-rest-api`
  - 监听：`https://127.0.0.1:27124`
  - 认证：API Key
  - TLS：自签名证书
- Docker Compose sidecar：`obsidian-bridge`
  - 监听：`27125`
  - 转发：`host.docker.internal:27124`
  - 透传：TCP 透传，不解 TLS
- InkWords backend（Go）
  - 内部模块 `ObsidianStore` 抽象
  - `RestAPIStore` 通过 `https://obsidian-bridge:27125` 调用 REST API

### 3.2 数据流（同步）

1. 前端点击「导出到 Obsidian」按钮（或其他同等入口）。
2. 请求 InkWords 后端导出 API（受 JWT 鉴权保护）。
3. 后端根据博客/系列数据，构建 Karpathy Wiki 的目标文件内容（frontmatter + markdown）。
4. 后端通过 `RestAPIStore` 调用 Obsidian REST API 写入/更新：
   - `vault/wiki/sources/...`
   - `vault/wiki/concepts/...`
   - `vault/wiki/entities/...`
   - `vault/wiki/index.md`
   - `vault/wiki/hot.md`
   - `vault/wiki/log.md`
   - 以及 `*_index.md` 与 `domains/*` 相关索引（如果启用）
5. 后端返回成功或失败（中文可读错误信息）。

## 4. Docker Compose 设计

### 4.1 新增服务：obsidian-bridge

- 目的：解决 “容器无法访问宿主机 127.0.0.1” 的网络隔离问题。
- 约束：
  - `obsidian-bridge` 不对宿主机端口映射（只在 docker network 内可访问）。
  - 只用于将容器内的请求透传到宿主机 Obsidian 插件端口。

### 4.2 访问地址

- 后端访问 base URL：`https://obsidian-bridge:27125`
- bridge 侧转发目标：`host.docker.internal:27124`

## 5. 配置与安全

### 5.1 环境变量（后端）

- `OBSIDIAN_REST_API_BASE_URL`：默认 `https://obsidian-bridge:27125`
- `OBSIDIAN_REST_API_KEY`：Obsidian Local REST API 插件生成的 API Key
- `OBSIDIAN_REST_API_CERT_PATH`：容器内证书路径（只读挂载）

### 5.2 TLS 策略（证书信任/Pin）

- 从插件下载证书：
  - `https://127.0.0.1:27124/obsidian-local-rest-api-certificate.crt`
- 将证书以只读 volume 挂载进后端容器。
- Go 客户端使用该证书构建 `RootCAs`，并显式设置 `ServerName = "127.0.0.1"`，避免证书主机名不匹配。
- 默认禁用跳过 TLS 校验；仅允许本机开发使用显式开关 `OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=true` 进行降级验证。

### 5.3 认证策略

- 每次请求必须携带 `Authorization: Bearer <OBSIDIAN_REST_API_KEY>`
- 禁止在日志中打印 API Key、证书内容或完整请求头。

## 6. 后端设计

### 6.1 ObsidianStore 抽象

定义 `ObsidianStore` 接口，封装导出所需的最小能力集合：

- `Read(path string) ([]byte, error)`
- `Put(path string, contentType string, body []byte) error`
- `Post(path string, contentType string, body []byte) error`
- `Patch(path string, headers map[string]string, contentType string, body []byte) error`
- `List(dirPath string) ([]string, error)`

说明：
- `path` 为 vault 内相对路径（与 Obsidian REST API 的 `/vault/{path}` 语义一致）。
- `List` 用于生成 `_index.md` 的链接列表，保持与既有 wiki scaffold 行为一致。

### 6.2 RestAPIStore 实现要点

- 统一 base URL：`OBSIDIAN_REST_API_BASE_URL`
- 默认使用 `/vault/{path}` 系列端点完成文件 CRUD：
  - GET：读文件
  - PUT：写/覆盖文件
  - POST：追加（在支持的 target 语义下）
  - PATCH：按 header 指定 `Operation/Target-Type/Target` 实现精准修改
- 遇到非 2xx：
  - 对内保留原始状态码与响应体片段作为根因（wrap）
  - 对外返回稳定中文错误信息（不泄露敏感 header）

### 6.3 导出服务逻辑（保持 Karpathy Pattern）

导出逻辑与现有 wiki 结构保持一致，核心动作包括：

- 确保目录与索引存在（scaffold）
- 写入：
  - 单篇：`concepts/<title>.md`（`type: concept`）
  - 系列：父篇写入 `sources/<series-title>.md`（`type: source`），子篇写入 `concepts/*`，并按抽取结果补齐 `entities/*` 与可能的额外 `concepts/*`
- 更新：
  - `index.md`：追加 series/source 链接或刷新索引页（策略需与现有实现保持一致）
  - `log.md`：追加 ingest/export 记录（插入位置规则保持一致）
  - `hot.md`：覆盖刷新热点上下文
  - `sources/_index.md`、`concepts/_index.md`、`entities/_index.md`：重建列表
  - `domains/*` 索引（如果启用）

## 7. API 设计（不新增对外能力）

- 沿用现有导出 API（示例）：
  - `POST /api/v1/blogs/:id/export/obsidian`
  - `POST /api/v1/blogs/:id/export/obsidian/series`（如已存在）

约束：
- 对外返回只表达“成功/失败 + 可读信息”，不返回 Obsidian API 的任何细节。

## 8. 验收标准（DoD）

- `docker compose down && docker compose up -d --build` 后：
  - 点击导出按钮，导出成功。
  - Obsidian 内可看到对应 `sources/`、`concepts/`、`entities/` 文件写入或更新。
  - `index.md`、`hot.md`、`log.md` 与各目录 `_index.md` 更新符合既有规则。
- 后端对 Obsidian 的访问使用证书信任/Pin 生效（非 `InsecureSkipVerify`）。
- 若开启 `OBSIDIAN_REST_API_INSECURE_SKIP_VERIFY=true`，必须确保仅访问 `obsidian-bridge` 内网服务，禁止在非本机/生产环境启用。
- Obsidian API Key 不会出现在后端日志、错误响应或前端可见信息中。
