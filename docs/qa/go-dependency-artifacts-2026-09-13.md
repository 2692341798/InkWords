# Go 离线依赖组合工件验收

日期：2026-09-13。状态：代码及存储/导出夹具通过；未部署、未登记教材、未执行教学代码。

## 本轮变化

- 普通工件保留 8 MiB；显式依赖入口独立限制正文 8 MiB、依赖 64 MiB、清单 2 MiB。
  普通入口拒绝 vendor/依赖锁文件，组合入口拒绝额外正文、错误哈希和超限内容。
- 执行清单新增可选 dependency_manifest_hash，参与验证输入哈希。空字段省略，旧版
  v1 清单序列化哈希兼容测试通过。
- 写入、带哈希解析、私有快照及 Bubblewrap 复制统一采用有界校验。执行前校验清单
  和实际工具链；执行器接收私有快照，避免校验后原目录修改影响输入，结束清理快照。
- 冻结/章节代码导出从对应执行清单取得依赖哈希，保留 vendor 与依赖锁文件。
  未改变沙箱权限、禁网、资源限制或数据库 schema。

## 实际验证

证据目录：`output/real-acceptance/2026-09-13/go-dependency-artifacts/`。

- `red.log`：新增组合入口实现前测试失败；`green.log`、`focused-green.log`：实现后通过。
  首次 focused 回归出现一处错误文字兼容断言失败，修正后通过，原始日志保留。
- `backend-tests.log`：`GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...`
  全量通过，含架构检查。gofmt 与 `git diff --check` 通过。
- 覆盖错误 pin、依赖变更、缺失清单、额外正文、正文超限、工具链不匹配、旧清单哈希
  兼容、私有快照隔离/清理及拒绝覆盖非空复制目标。runner 测试使用 fake executor。
- `verify_bundle.go` 与 `real-bundle-storage-export.json`：使用先前已校验的真实 Gin
  离线依赖树，加两份操作者占位文件，验证内容寻址写入、读取、私有快照、复制及
  FrozenBookPackageFiles。共 2,198 文件、35,853,261 字节；导出文件数一致且锁文件
  原字节保留。普通写入/读取入口均拒绝该包。

依赖清单：`sha256:79a707dfcddc51376f63339b866bb5d21b0922fa50f2a8b49a63c870bd884f5d`。
夹具工件及复制哈希均为
`sha256:d983a0ebc88c040b58903f1c51f7a78397ceca5e880880c2e414c414598b18f2`。

## 验收边界与下一步

该夹具不代表可运行 Gin 章，不产生教材、候选、批准、VerificationRun 或冻结构建记录。
没有真实提供商调用、教学代码执行、新依赖安装、服务部署或现有母稿变更。

下一步接入项目/来源/清单哈希的显式选择，再验证真实 Gin 在既有隔离预算中的编译、
测试与浏览器运行合同。依赖许可仍待审，hands_on 章仍为草稿；原计划三项综合验收不勾选。
