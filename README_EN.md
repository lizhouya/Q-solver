<div align="center">
  <img src="assets/banner.jpg" alt="Q-Solver Banner" width="100%">

  <h1>Q-Solver</h1>
  <p><strong>An AI desktop problem-solving assistant.</strong></p>
  <p>Capture a question, keep the window floating, and get answers fast. Supports hidden mode, click-through, and low-distraction use with your own API key.</p>

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
    <a href="README.md">中文</a>
  </p>
</div>

---

## Current Prerelease

**[v1.1.0-rc.2 — personal fork prerelease](https://github.com/lizhouya/Q-solver/releases/tag/v1.1.0-rc.2)** includes question-bank, API settings, and live-sync fixes. Download from this fork's Release rather than the historical upstream links below. Exit the old instance and back up your own configuration/archive before upgrading. The local entry point is `build/bin/Q-Solver.exe`; older validation builds are retained in `archive/` beneath it.

This is not a stable release: local race testing remains blocked and some manual acceptance checks are pending; macOS GUI behavior has not been manually verified. See the [changelog](CHANGELOG.md) and [release notes](_dev/releases/v1.1.0-rc.2.md).

## What This Is

Q-Solver is an AI desktop problem-solving assistant.

This repository is the personal [lizhouya/Q-solver fork](https://github.com/lizhouya/Q-solver) of [jym66/Q-solver](https://github.com/jym66/Q-solver). The question-bank feature below is a fork extension, not a claim about upstream releases. The CC BY-NC 4.0 non-commercial license still applies.

When you see a question, a code error, a formula, a chart, or English content on screen, you do not need to switch tabs and copy-paste everything into a chatbot. Just take a screenshot and send it straight to your model for analysis.

Its core strengths are:

- screenshot-first solving for coding questions, tests, interviews, and debugging
- hidden mode for lower-profile use
- floating output that does not keep stealing focus
- click-through support so it can stay at the edge of your screen
- less noticeable in many screen recording or screen sharing setups

---

## Q-Solver in 5 Screens

### 1. Welcome and Runtime Status: know your setup immediately

<img src="assets/img1.png" alt="Q-Solver Welcome and Runtime Status" width="100%">

- the welcome screen shows the core shortcuts right away
- the status panel shows your API key status, base URL, active model, and hidden mode
- a good fit for an always-available desktop solving assistant

### 2. Screenshot Settings: decide whether you want more clarity or more speed

<img src="assets/img2.png" alt="Q-Solver Screenshot Settings" width="100%">

- supports region capture and fullscreen capture
- lets you choose original upload or compressed upload
- adjustable compression, sharpening, and grayscale settings

### 3. Model and Scene Presets: same model, different solving styles

<img src="assets/img3.png" alt="Q-Solver Model and Scene Presets" width="100%">

- choose the model you want to use
- switch between different role or scene prompts
- make the same screenshot produce output that better matches the current task

### 4. API Setup: choose a provider and enter your own key

<img src="assets/img4.png" alt="Q-Solver API Setup" width="100%">

- supports OpenAI-compatible APIs
- includes presets for OpenAI, Google, Anthropic, DeepSeek, Alibaba Cloud, Moonshot, OpenRouter, and more
- when you choose `Custom`, you can enter your own `Base URL`

### 5. Result View: built for continuous reading and follow-up questions

<img src="assets/img5.png" alt="Q-Solver Result View" width="100%">

- the history panel on the left makes it easy to revisit previous questions
- the answer area is structured analysis, not just a one-line reply
- better suited for reading reasoning, formulas, code, and key steps while you work

---

## Core Features

- AI problem-solving assistant: capture content from your screen and send it directly to your model
- History panel: keep recent results for continuous review
- Model switching: automatically load available models from your current endpoint
- Scene presets: switch prompt styles for different question types
- Custom API support: works with any OpenAI-compatible service
- Screenshot tuning: compression, sharpening, grayscale, and original upload options
- Resume parsing: import a PDF resume and turn it into Markdown with your own model
- Desktop floating window: hidden mode, click-through, and shortcut-based control
- Low-distraction workflow: designed to stay available without constantly stealing focus

---

## Question-bank archiving (personal fork)

Question-bank records are independent of frontend history, `KeepContext`, and desktop/mobile output routing. Normal solving does not require manually starting a session:

1. In **Settings → Question Bank**, choose a default archive directory **outside the Git repository**. Settings only retain the default directory and default model-organization toggle; organization, issuer, type, and question-bank name are per-export fields.
2. The first solve lazily creates the current segment. Later solves append to it. The top-bar label “N submissions in this segment” counts solve submissions, not extracted questions; one submission may contain multiple questions.
3. Clicking **Export question bank** beside an answer first opens a confirmation dialog. For that export you can edit the recruiting organization, issuer or assessment provider, question type, name, temporary destination, and model-organization choice. Opening or canceling the dialog does not seal the segment, upload content, or call the model.
4. The suggested name uses `organization_issuer_type`, omitting empty or duplicate parts. If both organization and issuer are unknown, it uses a stable number such as `笔试_20261020_01_综合笔试`. A manually edited name remains protected until **Restore automatic naming** is selected.
5. With model organization off, export is local-only and creates `原始记录.md` plus `assets/`. With it on, **Organize and generate suggestions** processes records that still lack structured questions and returns question data plus metadata candidates. Every candidate remains pending human review. If every record fails, the app blocks a zero-question paper and offers retry-failed or raw-only actions. With partial results, you can retry only failures, explicitly export the available questions, or switch this export to raw-only. The final confirmed export is local-only and never calls the model again. The flow never sends the resume, full configuration, chat history, or solving persona, and never overwrites the original answer.
6. **End segment** only seals the current segment; the next solve creates a new one automatically. A successful export also seals it. **Clear segment** is a soft clear: the live desktop answer history returns to a clean waiting state, while source records, screenshots, prior exports, and model context remain intact. The cleared segment cannot be organized or exported again.
7. The Question Bank panel shows at most the latest 20 segments. Clicking a row still opens an independent read-only view. For an `ended`, `exported`, or `paused` segment, **Continue this segment** performs a real switch of the writable current segment and restores all submissions, one to three screenshots per submission, and full answers in the main screenshot-solving view. The prior active segment is safely paused and can be resumed later; `discarded` segments remain permanently read-only. Segment 21 and older stay on disk but are hidden and never trigger automatic organization, export, or deletion. Continuing an exported segment preserves every old export; another export uses `_02`, `_03`, and later suffixes.
8. After finishing one exam set, click **Start next exam** in the answer action bar or press `Ctrl + Shift + N`. After confirmation, the app safely seals the current segment and clears pending screenshots, desktop history, and retained model conversation. Durable source records, archived screenshots, previous exports, and the current Mobile display remain intact. The action is rejected while solving, archiving, organizing, exporting, or saving is still active; the next segment is created lazily by the next real solve.

Recommended layout; see [Change 011](_dev/changes/011-job-workspace-layout/06-notes.md) for migration details:

```text
求职_AI/
├─ 求职笔试/
│  └─ Q-Solver/
├─ 求职网申/
└─ 输出/
   ├─ 求职笔试/
   │  └─ 题库/                            # Default ArchiveDir from Settings
   │     ├─ _记录/<segment-id>/            # Durable source records for recovery/re-export
   │     │  ├─ session.json
   │     │  └─ assets/
   │     └─ <date_time_question-bank-name>[_02]/
   │        ├─ <question-bank-name>.md     # Image-free exam, then reference answers and notes
   │        ├─ 原始记录.md                 # Every screenshot reference and full original answer
   │        └─ assets/
   └─ 求职网申/
```

- Once the default directory is configured, the first solve starts recording automatically. History that was never recorded or has already been lost cannot be reconstructed. Archiving is not limited to the model's last ten conversation rounds.
- A submission may contain multiple questions. Source records preserve submission order; each organized question links back to its source submission. Failed, canceled, interrupted, or missing answers are labeled explicitly.
- The organized paper contains no images. Each question is immediately followed by its reference answer, knowledge points, explanation notes, source, and review note. Screenshots remain only in `原始记录.md` and `assets/`. Extracted content requires human review and is not an official answer key.
- Organization reports completed, failed, and pending counts separately. An all-failed run cannot create an empty organized paper; **Retry failed records** never reprocesses successful records. Partial results are exported only after an explicit choice, with unstructured submissions linked back to the full source draft. Raw-only export remains available at every failure state.
- On restart, the current `active` or `paused` segment and any recoverable model context are restored. New Change 018 records persist only up to ten non-system Session/Record references; API keys, full configuration, prompts, resume content, and thinking are never stored there. The next request always uses the current model, scene, prompt, and resume configuration. Legacy data can only be reconstructed from its own completed records and is labeled as context reconstructed from question-bank records that may differ from the original context.
- Continuation is rejected while a solve goroutine is still running, screenshots are waiting to be sent, data is unsaved, or organization/export is active. A conversation epoch rejects late desktop events from the old segment; the Mobile clear-display epoch remains independent.
- After a write failure, restore directory access and use **Retry save** instead of exiting. The final answer and conversation snapshot retry atomically, and successful recovery synchronizes the model context. A forced process termination can still lose unsaved data.
- Legacy `会话记录` directories and v1 manifests remain readable but are not moved, deleted, or overwritten automatically.
- Exported files are local plaintext and may contain private screenshots. Relevant screenshots and answers are sent to the configured provider only after model organization is enabled and the export dialog is submitted.
- Question-bank preferences live in a separate user-level `Q-Solver/questionbank-settings.json` and cover defaults plus migration compatibility, not credentials or per-export metadata.

**Scenes:** Existing Settings allow manual scene selection for subsequent requests; already-sent requests are not regenerated. Prior dialogue may still influence answers with `KeepContext` on. Automatic scene switching is **not implemented** in this change. See the [feasibility audit](_dev/audits/010-question-bank-and-scenes.md).

Other exam tools can later have their own sibling directories alongside Q-Solver, without sharing its repository. The categorized output directories remain outside the repository.

---

## Quick Start

### Option 1: Download the App

[Upstream releases](https://github.com/jym66/Q-solver/releases) provide the official app and may not include fork extensions. Build this fork's development branch for the question-bank feature; do not confuse upstream and personal releases.

> [!NOTE]
> If macOS blocks the app on first launch, run:
> ```bash
> xattr -cr /Applications/Q-Solver.app
> chmod +x /Applications/Q-Solver.app/Contents/MacOS/Q-Solver
> ```

### Option 2: Run from Source

Requirements:

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

Build a production package:

```bash
wails build -ldflags "-s -w" -tags prod
```

---

## Configuration

Recommended first-run flow:

1. Open Settings
2. In the `API` tab, choose a provider and enter your `API Key`
3. If you use a custom compatible endpoint, also enter `Base URL`
4. In the `Model` tab, refresh the model list and choose one
5. In the `Model` tab, choose the scene you want to use
6. In the `Screenshot` tab, adjust screenshot settings if needed
7. Go back to the main window and start solving

If configuration loading fails, use **Reload configuration**; saving is temporarily blocked to protect the existing file from empty defaults. Scene-list failures do not block API configuration loading. Unchanged question-bank preferences are not saved, and question-bank failures are reported separately from ordinary settings.

After changing provider or endpoint, explicitly refresh the model list in the Model tab. Refresh and connection tests use the current draft endpoint and API key without requiring a save first. Startup, opening Settings, and saving do not automatically fetch models. An explicit connection test sends a request to the selected provider and may incur charges.

---

## Shortcuts

Default shortcuts:

| Action | Windows | macOS |
|:---|:---:|:---:|
| Screenshot | `F8` | `Cmd + 1` |
| Send to solve | `Ctrl + J` | `Cmd + J` |
| Show / Hide window | `F9` | `Cmd + 2` |
| Click-through | `F10` | `Cmd + 3` |
| Move window slightly | `Alt + Arrow Keys` | `Cmd + Option + Arrow Keys` |
| Scroll content | `Alt + PgUp / PgDn` | `Cmd + Option + Shift + Arrow Keys` |
| Clear mobile display | `Shift + C` | — |
| Start next exam | `Ctrl + Shift + N` | — |

> macOS currently uses fixed shortcuts. Windows supports recording and adjusting shortcuts in settings.
> The mobile view shows the shortcuts that are actually configured. **Scroll content** follows the saved `desktop`, `mobile`, or `dual` output route and safely falls back to the desktop when the mobile service is unavailable.
> **Clear mobile display** is Windows-only and defaults specifically to left `Shift + C`; it can be re-recorded under **Settings → General**. The shortcut only opens the confirmation dialog, and the global hook consumes the matching key combination. During upgrade, if a legacy action such as Hide Window still owns that default combination, the legacy action is unbound and the combination is migrated to Clear Mobile Display so `Shift+C` cannot hide the main window.
> **Start next exam** also has a Windows-only default and can be re-recorded under **Settings → General**. It shares the same confirmation flow as the main-view button, never deletes durable question-bank data, and does not clear Mobile output. An existing user action that owns the default combination is never displaced during upgrade.

### Mobile connection and scrolling

- Open **Settings → Output** to view the phone connection entry. The page refreshes service status whenever Settings opens or the Output tab is selected. Read failures, not-started service, unavailable port, and ready service are reported separately, and retryable failures can be recovered in place.
- Stable links are enabled by default. The link changes once on the first upgrade to this feature; after that, a full restart reuses it while the app configuration directory, LAN IP, and port remain unchanged. The Token and port format are stable, not the network address, so DHCP, VPN, or network changes may still change the full URL.
- **Retry service** only restores listening; it does not rotate the link or clear the display. **Reset sync link** immediately revokes the old link and active SSE connections and starts the new link from a blank state. **Clear mobile display** clears answers, analysis, errors, and in-page history without changing the link.
- Clear and Reset do not delete desktop answers, question-bank records, historical segments, screenshots, exports, normal settings, or model context, and they do not cancel an in-flight solve. Late Mobile output from a pre-clear task is rejected by the display boundary, while new tasks continue normally.
- The mobile page uses an HTTP Token link. Anyone who can reach the address and has the complete link may view synchronized content. Copies, screenshots, and offline saves cannot be remotely revoked.
- A successful Clear message means the server state was committed and broadcast. SSE has no client acknowledgement, so it does not prove that every device has visibly rendered the blank state.
- Selecting an output-mode card is only a draft until you click **Save**. The page also shows the currently active mode so an unsaved `mobile` selection cannot be mistaken for an active route.
- The up/down arrows in the top bar scroll answer content. You can click them, use the saved global shortcuts, or place the pointer over the two-arrow control group and use the mouse wheel. Wheel handling does not take over Settings, the normal answer-reading area, or other system areas.
- While the phone is showing a historical answer, desktop scroll commands only move the current phone page; they do not switch the selection back to the latest answer. Reconnecting also does not replay an old scroll command.

---

## 仅复制可填写答案与题号规则

- 需要核对题目对应关系时使用“复制题号＋答案”，例如 `16. C`、`20. AB`；使用汇总中的实际题号，不重新编号。桌面和同步端均支持，原有纯答案复制及桌面全文复制保留。

- 每轮解题（包含 1～3 张截图）要求在全文末尾输出“最终答案汇总”，单题也适用。桌面保留“复制全文”，新增“复制答案”；同步端参考答案卡也提供“复制答案”。
- “复制答案”只取本轮汇总中的可填写值：单选如 `B`、多选如 `AC`；多空题以 `；` 分隔在一行；多题按显示顺序逐行复制，不带题号、题干、解析或“答案：”前缀。
- 含“待核对”、重复题号、缺失或格式不合规时，复制答案不可用，不会改成复制全文。旧记录只有符合固定三列表格时才可临时恢复复制，原文件不变。
- 同一题库分段保存题号参考；关闭“保持上下文”、重启或真正续接后仍可使用。该参考只含受限题号，不含旧答案或历史聊天；截图的显式题号/章节号优先，不能确定时标“待核对”。它是辅助提示，不是 OCR 正确性保证。
- “开始下一套”后，新分段不继承上一套题号。完整原答案、截图与旧导出继续保留，不为提取答案增加第二次模型请求。
- 同步端“答题端快捷键”显示已保存的实际绑定，包含清空同步端、开始下一套；未绑定动作不显示。按键在答题电脑使用，不是在浏览器执行。
- 浏览旧轮时复制的是所选轮。浏览器权限或 HTTP 环境不允许复制且兼容路径失败时，会提示失败，请手动选择文本；不显示虚假的成功提示。

## Tech Stack

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

This project is published under **CC BY-NC 4.0** as source-available software for personal, non-commercial use.

---

<div align="center">
  <p>If you like this screenshot-first AI workflow, a Star would mean a lot.</p>
  <p><a href="https://github.com/jym66">jym66</a></p>
</div>
