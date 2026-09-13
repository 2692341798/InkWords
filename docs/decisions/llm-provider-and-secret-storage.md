# 模型提供商与密钥存放

## 当前决定

教材生成的业务层只依赖 `shared/kernel/generation.Port`。实际提供商由
`llm-stream` 进程在启动环境中选择；默认 `fake` 提供商保持离线、确定性，
不会因为打开工作台或运行普通测试而产生外部调用。

目前真实提供商的密钥只从进程环境读取：`DEEPSEEK_API_KEY` 或
`OPENAI_API_KEY`。密钥不写入数据库、候选稿、导出物或浏览器状态，也不会由
HTTP 接口返回。本地设置页的安全存储录入仍是后续工作，不能把环境变量机制误称
为已完成的浏览器端配置功能。

## 显式连接探测

本机用户可在教材项目工作台点击“测试已配置连接”。这会通过固定本地工作区
调用 `POST /api/v1/stream/provider-connection-test`，构造一个固定、限 16 token 的
JSON 探测；它不含教材、导入资料、用户文本或 legacy user 标识。按钮之外不会
自动发送该请求。

成功结果只显示提供商、模型、请求 ID 和 usage 是否已返回。未配置提供商或调用
失败时，HTTP 和前端均使用稳定中文提示；不会回传模型输出、请求体、原始提供商
响应体或密钥。DeepSeek 的 `APIError` 也不再把响应体格式化进错误字符串，以免
错误日志、任务持久化或 SSE 路径意外带出不可信诊断。

DeepSeek 的普通、轻量和流式请求均不发送 `user_id`。博客/教材归属只在本地
workspace、任务和数据库合同中校验，不把历史 user UUID 或由其派生的字符串发送给
Provider；这不会移除请求 ID、模型、Token、缓存、延迟和重试等必要遥测。

`generation.ProviderError` 是非流式 adapter 的安全失败合同：仅保留
provider、类别与 HTTP status。OpenAI 和 DeepSeek 都把限流、拒绝、服务端、
传输与无效结构化响应映射到这个合同；调用取消和超时仍保留标准 context 原因，
以便任务层正确停止而不是把它误判为 provider 故障。

## 边界与后续

该探测验证的是已配置连接是否可用，不验证教材内容质量，也不应替代真实提供商
的显式 opt-in 小预算验收。完整的 provider error 合同、限流、超时、取消、浏览器
端安全存储与端到端 SSE/导出红队检查仍须在 PR-11 的剩余项中完成。
