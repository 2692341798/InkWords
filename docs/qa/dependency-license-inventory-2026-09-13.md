# Gin 固定离线依赖包：许可材料清点

日期：2026-09-13。范围是本地文件完整性、缓存来源与声明定位，不是法律审查或出版授权结论。

## 核验对象与结果

- Gin v1.12.0，提交 `73726dc606796a025971fe451f0aa6f1b9b847f6`，Go 1.26.8。
- 固定依赖清单 SHA-256：`79a707dfcddc51376f63339b866bb5d21b0922fa50f2a8b49a63c870bd884f5d`。
- 原始 `bundle-tree` 的 2,195 个声明文件、35,413,157 字节均逐项核对长度与 SHA-256；部署准备副本逐字节相同。加上依赖清单自身，组合输入为 2,196 个文件、35,853,139 字节。这两个口径不同，不是缺少一个文件。
- 31 个本地模块缓存 ZIP 按文件名排序、逐内容散列重算 h1，与固定清单及此前缓存审计一致。只说明与已记录来源一致，本轮没有联网重新认证上游。
- 实际带入文件的模块为 30 个；`github.com/stretchr/testify` 只在模块图中，没有 vendor 文件，因此其根 LICENSE 未被带入不计作已分发组件的许可遗漏。
- 对每个实际带入模块，缓存 ZIP 根目录匹配 LICENSE / LICENCE / COPYING / NOTICE / COPYRIGHT / UNLICENSE / PATENTS / AUTHORS / LEGAL 的文件，均已带入且字节相同。该规则不能证明源码内、嵌套目录或外部链接的声明都已完备。

## 更正旧计数

此前“33 份许可文件”的名称匹配包括两个 `.licenserc.yaml`。它们是许可头检查配置，不是许可证正文：

| 材料 | 数量 | 说明 |
| --- | ---: | --- |
| LICENSE 类正文 | 31 | 文本标识为 Apache 2.0 的 8 份、MIT 式 16 份、BSD 式 7 份；这是文本分类，不能外推整个模块的许可 |
| PATENTS | 6 | 五个 `golang.org/x` 模块与 `google.golang.org/protobuf` |
| AUTHORS.md | 1 | Gin 作者材料 |
| .licenserc.yaml | 2 | sonic、base64x 的配置，单列不计正文 |

共 38 份具名材料，另有 2 份配置。没有去重、删除或改写任何材料；base64x 的 LICENSE 与 LICENSE-APACHE 均保留。

## 已定位的逐文件声明例外

以下路径相对 `output/real-acceptance/2026-09-13/gin-offline-dependencies/bundle-tree/vendor/`。行号来自固定文件；完整字节与散列保存在清单中。

| 路径 | 本地证据 | 仍需核对 |
| --- | --- | --- |
| `github.com/gin-gonic/gin/tree.go:1`、`path.go:1` | Julien Schmidt 版权与 BSD-style 声明，指向 httprouter LICENSE；path.go 另写 Go Authors | Gin 根 LICENSE 是 MIT，不能据此把这两个文件也概括为 MIT；需映射其具体声明及固定来源 |
| `github.com/goccy/go-yaml/stdlib_quote.go:1` | 注明复制并裁剪 Go 固定提交 `e3769299cd3484e018e0e2a6e1b95c2b18ce4f41` 的 strconv/quote.go；第 8–10 行含 Go Authors/BSD-style | 根 MIT 与此声明分别记录；本轮未取回该上游提交 |
| `github.com/klauspost/cpuid/v2/os_linux_arm64.go:1` | MIT 声明后又保留 Go Authors/BSD-style 和 golang/sys LICENSE 链接 | 不能把包内另一模块的 BSD 文件自动认定为这个文件的完整许可映射 |
| `github.com/quic-go/qpack/varint.go:3` | 声明复制 Go 标准库 HPACK 实现 | 头部未给精确上游版本，需补来源定位 |
| `github.com/bytedance/sonic/internal/rt/fastconv.go:8` | `Copied from Golang` | 根 Apache 2.0 不能代替被复制片段的来源/声明核对 |
| `github.com/twitchyliquid64/golang-asm/obj/sym.go:1` | Inferno 派生路径、多方版权及内嵌 MIT 式许可正文 | 根 BSD 与文件内声明并存；保留完整头部，不能统一删除注释或只导出根许可证 |

这些是人工定向阅读后的例子，不是完整例外清单。辅助扫描仅检查源码前 45 行，产生 653 个含许可/复制关键词的文件信号；其中大量是正常 Go BSD 头部，不能把 653 当成缺陷数，也不能据未命中认定无声明。

## 状态与可复现证据

`rights_status=pending_review` 保持不变。既有 r17 小型教学代码的限定权利记录不扩展到整个 vendor 依赖包。没有修改数据库权利记录、批准稿、冻结构建、原依赖包、部署副本或旧审计 JSON；没有联网、下载、编译、执行教学代码或调用模型。

证据目录：`output/real-acceptance/2026-09-13/dependency-license-inventory/`。

- `audit.py`：只读取固定包、部署准备副本与本机缓存，写本目录派生清单。
- `inventory.json`：每份材料路径/散列、模块 ZIP h1、根材料覆盖与源码头信号。
- 复现：`python3 output/real-acceptance/2026-09-13/dependency-license-inventory/audit.py`。

本轮仅证据与文档变更，运行该核验及 `git diff --check`；未重复产品测试。下一步仍是逐文件来源/声明映射及真实动手章验收；原计划三项综合验收保持未完成。
