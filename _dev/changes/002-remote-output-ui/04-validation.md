# 验证记录 04 — 002-remote-output-ui

> 记录自动验证链与验收标准对照。真机验收需用户 `wails build` 后在 iPhone / iPad 执行，见末节。

## 1. 自动验证链（2026-09-27 最后确认）

| 项 | 命令 | 结果 |
|---|---|---|
| 语法护栏（真实 Script 目标） | `node _dev/mobile-check/check-raw-syntax.js` | ✅ exit 0「内联脚本通过真实 Script 目标语法检查」 |
| DOM/行为护栏 | `node _dev/mobile-check/dom-mobile-page.js` | ✅ exit 0「DOM 行为断言通过（14 全过）」 |
| 后端静态检查 | `go vet ./app/... ./pkg/mobile/...` | ✅ exit 0（无警告输出） |
| 后端回归 | `go test ./app/... ./pkg/mobile/...` | ✅ ok（app / pkg/mobile，缓存命中） |
| 后端构建 | `go build ./...` | ✅ exit 0 |

## 2. DOM 行为断言明细（dom-mobile-page.js，14 项全过）

| # | 断言概要 | 覆盖（00 §4 验收行） |
|---|---|---|
| 1 | idle → 业务状态「待命」，Answer 不显示 | Phone 首屏 / 状态 |
| 2a/2b | streaming + stage=thinking（带 analysis）→ 「处理中」，**不显示分析正文**（仍等待） | 「不显示 analysis」 |
| 3 | streaming + content → Answer 即时渲染 | 实时流式 |
| 4a/4b | done → Answer 显示 + History 入 1 卡 | 自动入史 + 选中 |
| 5 | 同 `seq` 重复 done → History 仍 1 卡（去重） | 「同 seq 不重复」 |
| 6 | 新 done → History 2 卡 + 最新 seq=11 高亮 | append + 最新选中 |
| 7a/7b/7c | 点击旧卡 seq=10 → 右侧切为「答案一」+ 高亮切到 10 | 「点击历史切换」 |
| 8a/8b | 新 done(seq=12) → 追加 3 卡 + 显示「答案三」 | 最新自动显示 |

## 3. 验收标准对照（00-request §4）

| 验收项 | 状态 | 说明 |
|---|---|---|
| Phone（~390px）单栏 Answer-first | ✅ 代码层（<899 单列；Answer 区位于 header 下方） | 真实渲染待真机 |
| Header 紧凑不占首屏 | ✅ CSS（flex-wrap、字号/边距紧凑） | 真机目检 |
| Tablet ≥900px 双栏（历史 ~200–240px \| Answer） | ✅ CSS `@media (min-width:900px)`：`.history-panel` 显示、`width: clamp(200px,22vw,240px)` | 断点逻辑已固定，尺寸真实渲染待真机 |
| Tablet Portrait 768–899 单列 + 紧凑 strip | ✅ `.history-strip` 在该范围显示（`<900px` 共享手机布局） | 见 03「无 768 断点」决策 |
| done → 自动入史 + 选中 + 显示最新 | ✅ 断言 4/6/8 | |
| 同 seq 重复 done 不重复 | ✅ 断言 5 | |
| 点击历史切换 + 高亮 | ✅ 断言 7a–7c | |
| streaming thinking 不显示 analysis / content 直接渲染 | ✅ 断言 2a/2b/3 | |
| 刷新 → 重连 → 最新快照（历史丢失=接受） | ✅ 首帧 `applySnapshot` 恢复 latest（D-UI-004/005） | 手动 |
| Markdown 安全链 escape→transform→render | ✅ 代码审查 + 渲染断言 3/4（标题/解析标签） | |
| 两层 JS 护栏 | ✅ 见 §1 | |
| `go test`/`go build` 回归 | ✅ 见 §1 | |
| **真机 iPhone/iPad** | ⏳ **用户执行**（`wails build -debug -windowsconsole` → `.\build\bin\Q-Solver.exe` → 手机/平板打开 `http://<LAN-IP>:<port>/?token=…`） | 测试用例：A 连接、B 解题 done 流、C 多题历史交互/去重、D 刷新恢复最新、E 390/768/1024/1366 视口三形态 |

## 4. 验证骨架说明

- 认证脚本基于 `_dev/mobile-check/`（001 遗留护栏）：`dom-mobile-page.js` 用极简 DOM shim 执行内联 `<script>`，逐帧喂 `snapshot` 事件断言运行时行为；`check-raw-syntax.js` 用 `vm.Script` 兜底 「new Function 包裹放行顶层 return」的语法盲区（001 缺陷经验）。
- 本验证链不启动 SSE 服务、不依赖真实网卡/显示器，纯 Node——可随时重复。