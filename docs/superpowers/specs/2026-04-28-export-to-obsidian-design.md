# 导出博客到 Obsidian (第二大脑) 架构设计

## 1. 背景与目标
为了将自动生成的项目解析博客无缝集成到个人的 Obsidian 第二大脑中，InkWords 需要提供一个“导出到 Obsidian”的快捷功能。通过本地文件系统的直接挂载，后端可以直接生成带有标准化 YAML Frontmatter 的 Markdown 笔记，从而完美契合现有的知识库管理规范。

## 2. 整体架构与方案

### 2.1 环境与容器配置
- **`.env` 环境变量**：新增 `OBSIDIAN_VAULT_PATH`，允许用户在主机上指定本地 Obsidian 仓库的绝对路径。
- **`docker-compose.yml` 修改**：在 `backend` 服务中新增数据卷挂载，例如 `- ${OBSIDIAN_VAULT_PATH:-/tmp/obsidian}:/app/obsidian`。后端代码将固定向容器内的 `/app/obsidian` 目录写入文件。

### 2.2 后端 API 与服务 (Go)
- **API 路由**：新增 `POST /api/v1/blogs/:id/export/obsidian` 接口。
- **业务逻辑 (Service 层)**：
  - 根据 `id` 从数据库中查询完整的博客信息（标题、内容、创建时间等）。
  - **组装 Frontmatter**：根据《第二大脑约束 (Obsidian LLM Wiki)》规范，动态生成 YAML 头。
    ```yaml
    ---
    type: concept
    title: "{{Blog Title}}"
    created: {{YYYY-MM-DD}}
    updated: {{YYYY-MM-DD}}
    tags:
      - "#domain/tech"
    status: seed
    ---
    ```
  - **文件写入**：检查 `/app/obsidian` 目录是否可写。将组装好的内容（Frontmatter + Markdown正文）写入文件 `/app/obsidian/{{Blog Title}}.md`。如果同名文件已存在，则追加后缀或覆盖（策略定为：覆盖，确保内容最新）。
  - **安全与权限**：接口复用现有的 JWT 鉴权中间件，只有登录用户才能触发导出。

### 2.3 前端 UI (React)
- **触发入口**：在博客详情页/编辑器页面（`Editor.tsx` 或类似查看页面）的操作区新增一个「📤 导出到 Obsidian」按钮。
- **交互反馈**：点击后显示 loading 状态，调用导出 API。成功后使用 Shadcn UI 的 Toast 或 Alert 组件提示用户“导出成功，已保存至 Obsidian”。失败则提示对应错误（如未配置挂载目录）。

## 3. 验收标准 (DoD)
1. **容器通信**：通过 docker-compose 重启后，挂载目录生效，后端能够正确写入主机的文件系统。
2. **格式正确**：生成的 `.md` 文件在 Obsidian 中打开时，YAML Frontmatter 能够被正确解析，正文格式完好。
3. **前端体验**：按钮位置合理，具备明确的成功/失败交互反馈。
4. **安全与隔离**：未配置 `OBSIDIAN_VAULT_PATH` 时（默认挂载到 tmp），接口能正常返回但不影响宿主机，或给出友好提示。

## 4. 后续扩展性
- 未来可支持选择导出类型（如 entity, concept, source）。目前默认以 `concept` (或通用类型) 导出。
