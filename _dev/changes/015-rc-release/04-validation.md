# 验证记录

日期：2026-10-22。除明确标注的人工验收缺口与 race 阻塞外，发布前本地检查已完成。

## 本地自动化

使用 `GOTOOLCHAIN=go1.25.13`：

- `go test ./... -count=1`：PASS。
- `go vet ./...`：PASS。
- `go build ./...`：PASS。
- `node frontend/settings_regression_test.mjs`：PASS。
- `node pkg/mobile/web_dom_test.mjs`：PASS。
- `npm --prefix frontend run build`：PASS；保留既有主 chunk `818.21 kB` 警告。
- Wails CLI `v2.11.0` Windows amd64 生产构建：PASS。
- `actionlint v1.7.7 .github/workflows/release.yml`：PASS。

本次 Wails 生产构建使用 `-skipbindings -nosyncgomod -m -platform windows/amd64 -ldflags "-s -w" -tags prod`，同时实际执行前端依赖与生产构建。构建前后 `go.mod`、`go.sum` 内容一致；runtime shim 的临时行尾改写已恢复。

## 已知阻塞与验收边界

- 系统 Go `1.27.1` + Wails CLI `v2.11.0`：FAIL，属于旧 Wails 包加载器与新 Go 工具链组合问题；发布基线固定 Go `1.25`，隔离 `1.25.13` 构建已通过。
- `go test -race`：BLOCKED，Windows 进程启动返回 `0xc0000139`；普通测试通过不替代 race 结果。
- 用户已确认 API 解题、同一 SSE 连接跨电脑实时显示无需刷新、答题端控制同步端滚动可用。
- 题库导出弹窗与目录选择完整矩阵、真实模型整理、Mobile 端口恢复/重连完整矩阵、三种滚动输入 × 三种输出模式、macOS GUI 尚未全部人工验收，不改写为 PASS。

## 尚待远端验证

Tag 推送后必须等待 Windows、macOS arm64、macOS amd64 和 Release 四个 job。最终发布成功以远端 workflow、Release `prerelease=true`、`make_latest=false`、Tag 目标和四个附件实际核对结果为准；本地通过不能代替该结果。
