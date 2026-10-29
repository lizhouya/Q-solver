# Change 027 验证记录

日期：2026-10-25
状态：自动验证完成；人工验收未执行

## 自动验证

| 项目 | 结果 |
|---|---|
| 测量测试 `TestConversationImageComposition_*` | PASS |
| `TestKeepContextRequestDoesNotExceedHistoryRoundBudget` | PASS（RED→GREEN） |
| `TestSystemBehaviorPromptRequiresExplicitVisionFailure` | PASS（RED→GREEN） |
| `go test ./... -count=1` | PASS（22 包 ok） |
| `go vet ./...` | PASS（无输出） |
| `go build ./...` | PASS |
| `go mod verify` | PASS（all modules verified） |
| 前端回归（13 个） | PASS（13/13） |
| `npm --prefix frontend run build` | PASS（4.86s） |
| `git diff --check` | PASS（仅 CRLF 提示） |
| `go test -race ./pkg/solution ./pkg/domain ./app` | **BLOCKED** |
| 独立验证程序构建 | PASS |

### race 详情

```
exit status 0xc0000139
FAIL	Q-Solver/pkg/solution	1.409s
exit status 0xc0000139
FAIL	Q-Solver/pkg/domain	0.370s
exit status 0xc0000139
FAIL	Q-Solver/app	2.032s
```

`0xc0000139`（STATUS_ENTRYPOINT_NOT_FOUND）为本机既有环境阻塞，
在 Change 021～026 中一致复现，**不是**本 Change 引入。记 `BLOCKED`，不写 PASS。

### 关键测试输出

```
KeepContext=false 单轮 2 图 → 画像=[system:0 user:2] 图片总数=2
KeepContext=true 第 1 轮 → 图片总数=2
KeepContext=true 第 2 轮 → 图片总数=4
KeepContext=true 第 3 轮 → 图片总数=6
KeepContext=true 第 13 轮（Max=10）→ 图片总数=22   ← 修复前
最后一轮请求：图片总数=20，其中历史 18 张、当前轮 2 张  ← 修复后
```

## 产物

| 项目 | 值 |
|---|---|
| 路径 | `build/bin/Q-Solver-change027-vision-failure-detection.exe` |
| 大小 | 60,950,016 字节 |
| SHA-256 | `f75e27398dfe3562ae7ac140bf33ec15d4ce6736ae50210d7981a1bb90c9c90a` |

## 保护核对

| 项目 | 结果 |
|---|---|
| `go.mod` / `go.sum` 哈希 | 与基线一致 |
| `frontend/package.json` / lockfile 哈希 | 与基线一致 |
| `Q-Solver.exe` 哈希 | 与基线一致 |
| Change 025/026 验证程序哈希 | 与基线一致 |
| 题库 `_记录`、截图、旧导出 | 未触碰 |
| 用户配置 | 未读取、未修改 |

## 网络与隐私审计

| 项目 | 结果 |
|---|---|
| 是否调用真实模型/CCR | 否 |
| 是否使用真实 API Key | 否 |
| 测试是否全部 loopback / fake | 是（`fakeProvider`，无网络） |
| 是否输出 Key/Token/URL/prompt/响应体 | 否 |

## 人工验收（NEEDS_USER）

| 项目 | 状态 |
|---|---|
| Direct 三协议多图解题 | 未执行 |
| CCR + DeepSeek 多图解题 | 未执行 |
| KeepContext 开启连续多轮不串题 | 未执行 |
| `StartNextExam` 后新分段不含旧内容 | 未执行 |
| 上游丢图时模型明确报错 | 未执行 |
| 用户 CCR 侧核对 alias 上游 | 未执行 |

## 根因声明边界

本 Change **不宣称**已定位“Claude 返回不相关题目”的唯一根因。已确认的事实是：

1. KeepContext 开启时旧截图会随历史重发，修复前请求携带 11 轮图片（本 Change 已收敛到 10 轮）。
2. prompt 缺少视觉失败指令（本 Change 已补）。

但同一请求会同时发往 Claude 与 DeepSeek，而仅 Claude 表现异常，
说明**仍有 Q-Solver 之外的因子**（CCR 路由、该 alias 的实际上游或视觉透传）。
该因子须由用户在 CCR 侧核对，本 Change 不代为断言。
