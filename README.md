<div align="center">
  <img src="assets/banner.jpg" alt="Q-Solver Banner" width="100%">

  <h1>Q-Solver</h1>
  <p><strong>AI 答题助手，截图即提问，悬浮即解答。</strong></p>
  <p>一个面向桌面场景的 AI 答题助手。支持隐藏模式、鼠标穿透、不抢焦点，用你自己的 API Key 就能直接开始。</p>

  <p>
    <a href="https://github.com/jym66/Q-solver/stargazers"><img src="https://img.shields.io/github/stars/jym66/Q-solver?color=ffcb6b&style=for-the-badge&labelColor=30363d" alt="Stars"></a>
    <a href="https://github.com/jym66/Q-solver/releases"><img src="https://img.shields.io/github/v/release/jym66/Q-solver?color=89d185&style=for-the-badge&labelColor=30363d" alt="Release"></a>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white&labelColor=30363d" alt="Go">
    <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=for-the-badge&logo=vue.js&logoColor=white&labelColor=30363d" alt="Vue">
    <img src="https://img.shields.io/badge/Wails-v2-E30613?style=for-the-badge&logo=wails&logoColor=white&labelColor=30363d" alt="Wails">
  </p>

  <p>
    <img src="https://img.shields.io/badge/macOS-000000?style=flat-square&logo=apple&logoColor=white" alt="macOS">
    <img src="https://img.shields.io/badge/Windows-0078D6?style=flat-square&logo=windows&logoColor=white" alt="Windows">
  </p>

  <p>
    <a href="README_EN.md">English</a>
  </p>
</div>

---

## 当前预发布版本

**[v1.1.0-rc.1 — 个人 fork 预发布](https://github.com/lizhouya/Q-solver/releases/tag/v1.1.0-rc.1)** 包含题库、API 配置与实时同步修复。请从个人 fork 的 Release 下载，而非下方历史上游链接。升级前正常退出旧实例并备份自己的配置/题库；本地统一入口为 `build/bin/Q-Solver.exe`，旧验证版保存在其 `archive/` 子目录。

本版本不是稳定版：race 环境阻塞及部分人工验收仍未完成；macOS 附件未经人工 GUI 验证。详见 [变更日志](CHANGELOG.md) 与 [发布说明](_dev/releases/v1.1.0-rc.1.md)。

## 这是什么

Q-Solver 是一个桌面端 AI 做题助手。

当前仓库是 [lizhouya/Q-solver](https://github.com/lizhouya/Q-solver) 个人 fork，基于 [jym66/Q-solver](https://github.com/jym66/Q-solver) 二次开发；下述题库归档是本 fork 的扩展，不代表上游发行版已包含。许可仍为 CC BY-NC 4.0，仅限非商业个人学习与研究。

你看到题目、代码报错、公式、图表或者英文内容时，不需要再切网页复制粘贴，直接截图就能把内容发送给模型分析。

它的核心特点是：

- 截图即问，适合算法题、笔试题、面试题和代码问题
- 支持隐藏模式，更适合低存在感使用
- 悬浮展示，不抢当前应用焦点
- 支持鼠标穿透，方便挂在屏幕边上随看随用
- 对多数录屏 / 共享场景更不显眼

---

## 五张图看懂 Q-Solver

### 1. 欢迎页与运行状态：打开就知道现在能不能用

<img src="assets/img1.png" alt="Q-Solver 欢迎页与运行状态" width="100%">

- 欢迎页直接告诉你快捷键怎么用
- 右上角状态面板能看到 API Key、Base URL、当前模型和隐藏模式
- 很适合做成一个常驻桌面的 AI 答题助手

### 2. 截图设置：决定你是更要清晰度，还是更要速度

<img src="assets/img2.png" alt="Q-Solver 截图设置" width="100%">

- 支持区域截图和全屏截图
- 可控制是否原图上传，以及压缩质量、锐化、灰度模式
- 让你按题目类型和 token 成本来调节截图策略

### 3. 模型与场景：同一个模型，也能切不同解题风格

<img src="assets/img3.png" alt="Q-Solver 模型与场景" width="100%">

- 选择当前要使用的模型
- 切换不同角色 / 场景 prompt
- 让同一张截图输出更贴合当前任务

### 4. API 配置：选择提供商，填入你自己的 Key

<img src="assets/img4.png" alt="Q-Solver API 配置" width="100%">

- 支持 OpenAI-compatible 接口
- 预设 OpenAI、Google、Anthropic、DeepSeek、Alibaba Cloud、Moonshot、OpenRouter 等提供商
- 选择 `自定义` 时可填写自己的 Base URL

### 5. 结果页：真正适合连续做题和追问的阅读体验

<img src="assets/img5.png" alt="Q-Solver 结果页" width="100%">

- 左侧历史记录方便回看上一题
- 右侧输出是结构化分析，不是一句简单答案
- 更适合边做题边看思路、公式、代码和关键步骤

---

## 核心功能

- AI 答题助手：截取桌面内容并直接发送给模型分析
- 历史记录：保留最近结果，方便连续查看
- 模型切换：自动拉取当前接口可用模型
- 场景预设：针对不同题型切换不同提示词风格
- API 自定义：支持任意 OpenAI-compatible 服务
- 截图调节：压缩、锐化、灰度、原图上传可配置
- 简历解析：导入 PDF 简历并调用你自己的模型整理为 Markdown
- 桌面浮窗：支持隐藏模式、鼠标穿透、快捷键操作
- 低打扰体验：尽量不抢焦点，适合挂在桌面边缘随时查看

---

## 笔试题库归档（个人 fork）

题库记录独立于前端历史、`KeepContext` 和桌面/手机输出模式。日常做题不需要手动开始场次：

1. 在主“设置 → 题库”中选择 **Git 仓库外的默认题库目录**。设置页只保存默认目录和“默认启用模型整理”开关，不长期保存单位、题型或题库名称。
2. 完成第一道题时会懒创建“当前段”。后续解题继续追加到当前段；顶部的“本段 N 次提交”表示提交次数，一次提交可能含多道题。
3. 点击答案旁“导出题库”只会先打开确认窗口。可为本次填写单位、题型、题库名称、临时导出位置，并决定本次是否调用模型；打开或取消窗口不会封存分段、上传内容或产生模型费用。
4. 默认名称按“单位_题型”生成；单位不明确时使用稳定编号，例如 `笔试_20261020_01_综合笔试`。手动修改名称后，单位或题型变化不再覆盖它。
5. 关闭本次模型整理时，只在本地生成 `原始记录.md` 和 `assets/`。开启后，提交确认才会把该段截图和原答案逐条发送给当前支持图片的模型，提炼题干、题目大类、具体题型、参考答案、知识点和解题要点；这会产生额外调用，但不发送简历、完整配置、聊天历史或做题角色，也不覆盖原答案。
6. “结束本段”只封存当前段；下一道题自动进入新段。导出成功也会自动封段。“清空本段”是软清空，原始文件仍保留，但该段不再参与导出。
7. 历史分段通过题库面板中的“重新导出…”进入同一个确认窗口，不会先切换当前可写段。重复导出使用 `_02`、`_03` 等后缀，不覆盖旧版本。

推荐工作区布局如下，实际迁移状态见 [Change 011](_dev/changes/011-job-workspace-layout/06-notes.md)：

```text
求职_AI/
├─ 求职笔试/                              # 多种笔试工具的分类目录
│  └─ Q-Solver/                           # 当前工具的完整源码、.git 与开发记录
├─ 求职网申/                              # 预留目录，本次没有网申程序
└─ 输出/
   ├─ 求职笔试/
   │  └─ 题库/                            # 设置中的默认 ArchiveDir
   │     ├─ _记录/<分段ID>/               # 恢复与重新导出的持久原始记录
   │     │  ├─ session.json
   │     │  └─ assets/
   │     └─ <日期_时间_题库名称>[_02]/
   │        ├─ <题库名称>.md               # 无图片试卷：前半试题，后半参考答案与解析
   │        ├─ 原始记录.md                 # 全部截图引用和完整原答案
   │        └─ assets/
   └─ 求职网申/
```

**边界与恢复：**

- 目录配置完成后，第一道题会自动开始记录；此前未录制或已丢失的历史无法事后补全。归档不受模型最近 10 轮上下文限制。
- 一次提交可能包含多题；原稿按提交顺序完整保存，整理后的每道题回链对应原始提交。失败、取消、空答案或中断会明确标记，不当作正确完整答案。
- 整理稿不嵌入图片，只按“试题 → 参考答案与解析”组织；图片只保留在 `原始记录.md` 和 `assets/`。模型提炼内容均待人工核对，不是官方标准答案。
- 模型整理失败或取消后仍会导出原稿和已完成的整理结果；未成功结构化的提交会在整理稿中列出。
- 重启后会恢复最近的活动分段。旧版 `会话记录` 和 v1 清单仅兼容读取，不自动移动、删除或覆盖。
- 写盘失败时不要直接退出；恢复目录可写后重试保存。强制终止进程仍可能丢失未落盘数据。
- 输出文件是本地明文资料，可能包含私人截图，应自行管理访问和备份。只有在导出确认窗口中启用模型整理并提交后，相关截图和答案才会再次发送给已配置服务商。
- 题库偏好单独保存在系统用户配置目录的 `Q-Solver/questionbank-settings.json`，只承担默认目录、默认整理开关和兼容迁移信息，不与含 API Key 的主配置文件耦合。

**角色/场景：** 现有设置支持手动选择并保存，新场景用于后续请求；已发出的请求不会自动重答。`KeepContext` 开启时旧问答仍可能影响后续题目。本次未实现自动场景切换；“通用解题”识别题意不等同于程序自动切换 `domainId`。详见 [可行性分析](_dev/audits/010-question-bank-and-scenes.md)。

其他同类工具以后分别放在“求职笔试”下，与 Q-Solver 同级；不要混入 Q-Solver 的仓库。Q-Solver 已完成复制、校验和旧副本清理，分类输出目录现位于仓库外。

---

## 快速开始

### 方式一：直接下载

前往 [上游 Releases](https://github.com/jym66/Q-solver/releases) 下载官方版本；不保证包含本 fork 的题库归档扩展。本 fork 的新功能请使用当前开发分支自行构建，不将上游与个人发布混用。

> [!NOTE]
> macOS 首次运行如果提示无法打开，可执行：
> ```bash
> xattr -cr /Applications/Q-Solver.app
> chmod +x /Applications/Q-Solver.app/Contents/MacOS/Q-Solver
> ```

### 方式二：源码运行

环境要求：

- Go 1.25+
- Node.js 22+
- Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone --branch dev/qsolver-custom https://github.com/lizhouya/Q-solver.git
cd Q-solver
npm --prefix frontend ci

wails dev
```

构建发布版：

```bash
wails build -ldflags "-s -w" -tags prod
```

---

## 配置方式

第一次使用建议按这个顺序：

1. 打开右上角设置
2. 在 `API` 页面选择模型提供商并填写 `API Key`
3. 如果是自定义兼容接口，再填写 `Base URL`
4. 在 `模型` 页面刷新模型列表并选择一个模型
5. 在 `模型` 页面选择你想使用的场景
6. 在 `截图` 页面按需调整截图参数
7. 回到主界面开始截图解题

配置加载失败时会显示“重新读取配置”，保存暂时禁用，以免默认空值覆盖原配置。场景列表失败不影响 API 配置读取；未改动的题库偏好不参与保存，题库保存失败也会与普通设置结果分别提示。

切换服务商或自定义地址后，在“模型”页主动点击刷新，再选择模型。刷新和连接测试使用本次填写的地址及 API Key，不必先保存；启动、打开设置和保存本身不会自动请求模型列表。显式连接测试会向所选服务商发送测试请求，可能产生费用。

---

## 快捷键

默认快捷键如下：

| 动作 | Windows | macOS |
|:---|:---:|:---:|
| 截图 | `F8` | `Cmd + 1` |
| 发送解题 | `Ctrl + J` | `Cmd + J` |
| 显示 / 隐藏窗口 | `F9` | `Cmd + 2` |
| 鼠标穿透 | `F10` | `Cmd + 3` |
| 微调窗口位置 | `Alt + 方向键` | `Cmd + Option + 方向键` |
| 滚动内容 | `Alt + PgUp / PgDn` | `Cmd + Option + Shift + 方向键` |

> macOS 当前使用固定快捷键；Windows 支持在设置中录制和调整。
> 手机同步端会显示当前实际生效的快捷键；“滚动内容”按已保存的 `desktop`、`mobile`、`dual` 输出模式路由。手机服务不可用时，滚动控制安全回退到桌面。

### 手机连接与滚动控制

- 在“设置 → 输出”查看手机连接入口。每次打开设置或进入该标签都会刷新服务状态；读取失败、服务未启动、端口不可用和服务就绪会分别提示，可重试状态可直接在页面恢复。
- 输出模式卡片中的临时选择只有点击“保存”后才生效；页面会同时显示当前实际生效模式，避免把未保存的 `mobile` 选择误认为已经改变路由。
- 顶部上下箭头用于滚动答案。可点击箭头、使用已保存的全局快捷键，或把鼠标放在两个箭头组成的控制组上滚动滑轮。滑轮不会接管设置窗口、普通答案阅读区或系统其他区域。
- 手机正在查看历史答案时，桌面滚动命令只滚动当前手机页面，不会把历史选择切回最新答案；断线重连也不会重放旧滚动命令。

## 技术栈

- Core: Go
- Desktop Binding: Wails
- Frontend: Vue 3 + Pinia
- LLM Access: OpenAI-compatible API + OpenAI SDK

---

## Star History

<div align="center">
  <a href="https://star-history.com/#jym66/Q-solver&Date">
    <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=jym66/Q-solver&type=Date" />
  </a>
</div>

---

## License

本项目采用 **CC BY-NC 4.0** 许可公开源码，仅供非商业个人学习与研究使用。

---

<div align="center">
  <p>如果你喜欢这种“截图即解题”的桌面 AI 工作流，欢迎点一个 Star。</p>
  <p><a href="https://github.com/jym66">jym66</a></p>
</div>
