# Change 023 事前现状

日期：2026-10-25

- Change 022 已提供每轮 answerItems（label/kind/value/status）及后端校验的 answerCopyText。
- Desktop ActionBar 的复制答案只使用 answerCopyText，另有复制全文；clipboard helper 已支持三层 fallback 和成功/失败返回。
- Mobile setCopyTarget 保存当前/所选历史轮的纯答案，copyDisplayedAnswers 复用浏览器 fallback，并用 copyVersion 防止异步反馈串轮。
- 无需修改后端数据、再次解析 Markdown 或调用模型；只对已校验的数据增加显示格式。
- 当前 dev/qsolver-custom 有 Change 016～022 累积未提交改动；全部保留。
