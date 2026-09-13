# 从已准备的离线依赖生成教学工件

适用：本地单用户工作台的 `hands_on` 章。依赖来自操作方事先准备的固定模块包；
网页不下载依赖、不接收文件路径或 shell 命令，也不自动运行或批准候选。

## 操作方准备

1. 按现有 Go 离线包流程准备 `bundle-tree` 和 `dependency-manifest.json`，保留固定
   module/version/commit、精确工具链、每文件哈希和许可文件。先用
   `textbook-dependency-inspect` 预检，不能用当前下载结果替代未核对的固定版本。
2. 使用 `TeachingDependencySelection` 记录本地工作区、项目、章节、主资料快照、
   依赖清单哈希和可选 `browser_page` 观测。选择 JSON 的原始字节也计算 SHA-256。
3. 在 core-api 配置只读目录，设置 `TEXTBOOK_DEPENDENCY_CATALOG_FILE` 指向目录内
   `catalog.json`。未配置时返回空选项，不自动寻找缓存或从互联网补齐依赖。

目录示例（路径为容器内操作方配置，不是 HTTP 请求字段）：

```json
{
  "contract": "inkwords.dependency-catalog.v1",
  "entries": [{
    "id": "gin-orders-pinned",
    "title": "Gin 订单接口固定依赖",
    "selection_path": "/app/prepared-dependencies/selection.json",
    "selection_hash": "sha256:<已核对的选择文件哈希>",
    "root": "/app/prepared-dependencies/bundle-tree",
    "manifest_path": "/app/prepared-dependencies/dependency-manifest.json"
  }]
}
```

操作目录应只包含该目录需要的选择、依赖树、清单和许可，不挂载整个主目录或项目根。
目录文件最多 64 KiB、16 项，选择文件最多 16 KiB；未知字段、尾随 JSON、重复 ID、
不完整路径或选择哈希不符会拒绝。目录是可信的操作方配置，不能从导入教材生成。
依赖树的符号链接、路径、逐文件哈希与大小继续由已有离线包校验器限制。

## 网页工作流

打开动手章，在“为候选准备离线依赖”选择已准备包和已保存的生成候选：

1. 查看固定源码、依赖清单与工具链。空章可以查看包，但不能预览或登记工件。
2. 点击“预览依赖工件”。后端重读包和选择文件，检查当前来源归属以及候选原生成
   任务冻结的证据，不从其它章节或项目补选来源。
3. 核对文件数量、总字节、Go 测试/浏览器观测步骤和确认指纹，勾选确认再登记。
   指纹覆盖候选身份、内容、来源选择、依赖包、合同、全部文件、入口、限制与步骤。
4. 登记时重新计算指纹；来源、候选或依赖变化即拒绝，必须重新预览。成功后刷新
   章节的教学工件区，状态仍为未验证。随后才能通过独立按钮申请隔离验证。

页面不会自动选择依赖、调用生成模型、追加真人审校或改变权利状态。生成输入不因
目录读取而改变；依赖在候选投影时由用户显式确认，并冻结到工件执行清单。刷新页面
后需重新预览确认。登记响应不确定时先刷新章节核对；相同请求使用同一确定性工件 ID，
已有登记不被覆盖。没有宿主机执行或去掉依赖继续投影的回退路径。

## HTTP 合同

同源前缀：`/api/v1/textbook-projects/chapters/:chapterID/dependency-projection`。
工作区由服务端本地中间件确定；请求不能指定工作区、依赖路径或执行命令。

| 方法与后缀 | 输入 | 结果 |
| --- | --- | --- |
| GET `/options` | 无 | 本章可用选项及来源，不暴露操作方路径 |
| POST `/preview` | `option_id`、`task_id`、`revision_id`、`content_hash` | `inkwords.dependency-projection.v1` 预览和 `confirmation_hash` |
| POST `/apply` | 同一输入，追加 `confirmed_preview_hash` | 幂等登记后的未验证 CodeArtifactRow |

JSON 请求最多 4 KiB，严格拒绝未知字段与尾随内容。接口每实例只允许一次依赖操作，
并设置 30 秒上下文上限；忙时返回 429。未确认或指纹变化为 409，未知选项/候选为
404，文件或来源核对失败为 503，路径及底层错误不会返回浏览器。
此 API 不新建数据库表或索引；沿用既有来源成员查询、候选读取与事务登记校验。

## 恢复与验收边界

禁用目录配置即可停止提供选项；已有冻结工件继续按其清单校验，不被删除或重写。
更新目录或依赖需要重新预览，不能改旧工件中的执行清单。回退服务时使用原 Compose
文件链和旧镜像，撤去新增的只读目录挂载；不要仅用 image 覆盖文件启动整套服务。

组件/合同测试不替代真实 Gin 候选登记与执行。当前真实章若仍为 r0，应明确记录
“选项核对通过，真实工件登记和 go_test/browser_page 尚未执行”。固定来源哈希不证明
依赖许可已获出版审核，也不证明读者完成了学习。
