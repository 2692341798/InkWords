# Gin 离线依赖包准备与校验

日期：2026-09-13。仅完成准备与产品校验，不是新章运行验收。

## 版本与字节来源

Go 官方模块代理的固定提交查询和版本查询均返回 Gin v1.12.0、来源提交
`73726dc606796a025971fe451f0aa6f1b9b847f6`、`refs/tags/v1.12.0`。两条原始元数据保存在
`fixed-commit-info.json` 和 `release-info.json`。这补齐此前“本机版本与教材提交关系未证明”的缺口。

来源：[固定提交元数据](https://proxy.golang.org/github.com/gin-gonic/gin/@v/73726dc606796a025971fe451f0aa6f1b9b847f6.info)、
[发行版元数据](https://proxy.golang.org/github.com/gin-gonic/gin/@v/v1.12.0.info)。

在独立 output 目录，用操作者编写的仅导入 Gin 的探针提取依赖，未编译/执行探针。
从后端已有版本集合出发，以缓存中的 Go 1.26.0 执行 `go mod tidy` 和 `go mod vendor`，
全程 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local。未下载模块或升级后端依赖。
传递依赖锁定为后端已缓存的实际版本，不声称等于 Gin 原 go.mod 的最低版本集合。

初次命令被宿主默认 Go 1.25.4 拒绝，随后显式选择已有 Go 1.26.0；没有自动下载工具链。
最终依赖 go.mod/清单声明目标运行器 go1.26.8，这一目标尚未进行本次真实编译实测。

## 结果

- 31 个固定模块；30 个向 vendor 提供实际文件，testify 只保留模块元数据。
- vendor 树 2,193 文件、35,403,762 字节；加入 go.mod/go.sum 后 2,195 文件、
  35,413,157 字节。组合接口再附清单，合计 2,196 文件、35,853,139 字节。
- 清单 SHA-256：`79a707dfcddc51376f63339b866bb5d21b0922fa50f2a8b49a63c870bd884f5d`。
- 31 个模块的 ZIP、解压目录和 Go 模块文件均用官方 x/mod dirhash 算法核对 go.sum；
  所有实际 vendor 文件再逐字节对照已核对的缓存。没有把修改过的缓存当作可信原文。
- 30 个有实际文件的模块均有具名许可文件。后续逐项清点更正：旧索引 33 项含
  两份 `.licenserc.yaml` 配置，实际为 31 份许可正文，另有 6 份 PATENTS、1 份
  AUTHORS；见 `docs/qa/dependency-license-inventory-2026-09-13.md`。已保存路径/哈希索引，
  `rights_status=pending_review`。未据此登记许可通过或出版通过。
- 产品只读检查器对真实包返回成功、`activated=false`、`executed=false`。

## 验证与实现范围

新增依赖加载器覆盖清单/文件篡改、额外文件、缺件、符号链接、禁止的 replace、版本
冲突、checksum 冲突、重复、大小预算、未知字段、尾随 JSON 和取消。组合接口测试
证明母稿字节不变、依赖变化导致工件 ID/树哈希变化、来源/工具链不匹配拒绝。
后端 `GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...` 全量与架构通过。

当前仅提供校验器、只读命令和可信调用方组合接口；没有把包接入生产配置，没有改变
普通工件 8 MiB 限制，没有运行或批准 hands_on 章、发起 Provider 调用、修改数据库或
部署新镜像。下一步具体范围见 `docs/decisions/offline-go-dependencies.md`。

全部本地证据位于 `output/real-acceptance/2026-09-13/gin-offline-dependencies/`：
`dependency-manifest.json`、`prepared-bundle.json`、`cache-and-license-audit.json`、
`inspection.json`、`cache-audit.log`、`verify-modules.log`、`red.log`、`green.log`、
`focused-tests.log`、`backend-tests.log`。大体积 vendor 字节只在 output，未提交版本库。
