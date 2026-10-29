# 验证记录 04

## 构建验证

| 命令 | 结果 | 备注 |
|---|---|---|
| `npm --prefix frontend run build` | PASS | Vite v4.5.14，132 个模块转换成功，耗时 3.54 秒；未安装依赖 |
| `wails build -s -o Q-Solver-change030-latest-validation.exe -skipbindings -skipembedcreate -nosyncgomod -m -nocolour -v 1 -ldflags "-s -w" -tags prod` | PASS | Wails CLI v2.16.0，生产模式独立构建耗时 15.214 秒 |
| `git diff --check` | BLOCKED（既有差异） | 返回 2；`frontend/wailsjs/go/models.ts` 存在构建前已有尾随空白，构建前后哈希一致，本 Change 未修改 |

## 产物

- 路径：`build/bin/Q-Solver-change030-latest-validation.exe`
- 大小：42,333,184 字节
- 修改时间：2026-10-28 23:39:19.537 +08:00
- SHA-256：`5924B7D00CE0B00A4BE5A0169EDDD8D900F9442E96B621ACE7F374DD28AB07EF`
- 嵌入前端入口：`frontend/dist/index.html`，构建时间 2026-10-28 23:38:49 +08:00
- 主要资产：`index-df04f70a.js`、`index-cbf54d81.css`

## 运行验证

| 检查项 | 预期 | 实际 | 通过 |
|---|---|---|---|
| EXE 直接启动 | 不依赖 Vite/Wails DevServer | 启动 8 秒后进程仍存活；进程树只有应用与 WebView2 | ☑ |
| WebView2 初始化 | 正常 | 创建 6 个 `msedgewebview2.exe` 子进程 | ☑ |
| 配置初始化 | 不崩溃且不输出 secret | 应用完成启动并持续存活；验证过程未读取或输出 secret | ☑ |
| 开发服务器依赖 | 不启动 Node/Vite | 进程树中 Node/Vite 数量为 0 | ☑ |
| 模型调用边界 | 不连接 CCR | 观察期已建立连接为 0，CCR 3456 端口连接为 0 | ☑ |
| 进程退出 | 无本次残留进程 | `CloseMainWindow` 成功，退出码 0，未强制终止，应用及直接子进程均为 0 | ☑ |

## 验收标准对照

- [x] 最新前端资产已嵌入。
- [x] 唯一命名 EXE 已生成。
- [x] EXE 已完成自动化独立启动验证。
- [x] 11 个关键保护文件构建前后哈希不变。
- [x] Git 语义索引与保护副本一致，staged 条目为 0；二进制 index stat cache 刷新已单列记录。
- [x] 未安装依赖、未覆盖旧产物、未提交或推送。
- [ ] Windows GUI 人工双击与功能体验待用户验收。

## 回归与副作用检查

- 构建前后均为 53 个 tracked 修改、87 个 untracked、0 个 staged；本次产物受现有忽略规则保护，不进入 Git 状态。
- `wails.json`、`main.go`、`go.mod`、`go.sum`、前端包清单/锁文件和 5 个既有 Wails bindings/runtime 文件哈希全部不变。
- `.git/index` 的二进制 SHA-256 因一次未设置 `GIT_OPTIONAL_LOCKS=0` 的只读 Git 检查发生 stat-cache 刷新：`FE0948DA...` → `5EC93489...`。当前与保护副本的 1,194 条 `git ls-files --stage` 条目 SHA-256 均为 `DB01E541F4E2FD18173AE578904C77CE07E01F9F3B79D838016A07A3568324A5`，staged 为 0，因此索引语义未变化。
- 新 EXE 启动退出后 SHA-256 与大小保持不变。
- Vite 有一个既有大 chunk 警告（主 JS 约 863.89 kB，gzip 264.46 kB），不阻断构建。
- 全树 `git diff --check` 的既有尾随空白失败单独保留，不错误标记为 PASS。

## 结论

Change 030 的前端构建、独立 Wails 构建、产物哈希、独立启动、WebView2 初始化、无开发服务器依赖和正常退出均已自动验证通过。产物可交付用户双击进行 GUI 人工验收；本 Change 不 Close、不提交、不推送、不发布。
