# 验证记录

日期：2026-10-22，Windows amd64。

| 检查 | 结果 | 证据与范围 |
|---|---|---|
| 修复前 mock | 已复现 | 场景失败丢弃有效配置；题库失败阻断普通保存 |
| 修复前 Go 测试 | 预期失败 | loaded credentials were reset；corrupt config must return an error；单次接口未定义 |
| node frontend/settings_regression_test.mjs | PASS | 独立加载、失败保存阻断、重读恢复、完整草稿参数、原状态不变、未展示字段保留、旧响应丢弃、题库部分失败 |
| node pkg/mobile/web_dom_test.mjs | PASS | 既有手机 DOM 控制回归 |
| go test ./pkg/config ./pkg/llm ./app -count=1 | PASS | 坏配置保护、幂等加载、初次保存、启动前读配置、本地双端点隔离 |
| go test ./... -count=1 | PASS | 全量回归 |
| go vet ./... | PASS | 静态检查 |
| go build ./... | PASS | Go 包构建 |
| npm --prefix frontend run build | PASS | 123 modules，主 JS 818.21 kB，保留既有 chunk 大小告警 |
| Wails bindings 生成与中间构建 | PASS | 新方法正式生成，旧方法兼容保留 |
| wails build -s -o Q-Solver-api-fix.exe -skipbindings -nosyncgomod -m | PASS | 最终前端资产已嵌入；修复版保留供用户验收 |
| go test -race ./pkg/config ./pkg/llm ./app -count=1 | BLOCKED | 三个测试进程启动即 exit status 0xc0000139，不冒充通过 |

## 人工验收

- [ ] 先保留原配置备份，正常退出旧实例，启动修复版；原 Key/地址/模型应被读入。
- [ ] 切换接口并点击模型刷新，确认新端点生效；此步骤由用户主动调用真实服务。
- [ ] 保存 API 设置不受未改动的题库目录影响；修改题库但保存失败时普通设置仍保存且提示部分成功。
- [ ] 重新启动后普通设置仍保持。
- [ ] 配置不可读时显示重新读取入口且禁止保存，不覆盖原文件。

测试只使用临时目录及 localhost 模拟服务、合成密钥；没有调用真实模型。构建不能代替 WebView2 实测。状态：待人工验收、未提交、未关闭。

## 最终差异检查

- `git diff --check`：PASS；新增测试与 Change 014 文档空白/冲突标记扫描：PASS。
- `go.mod`、`go.sum`、前端依赖清单及 Wails runtime 无内容差异；未升级依赖。
- 暂存区为空；原有 Change 010～013 修改仍保留，未提交或发布。
- 最终修复版 EXE 存在，23,580,160 字节；未自动启动，旧程序未覆盖。
