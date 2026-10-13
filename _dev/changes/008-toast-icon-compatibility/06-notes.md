# 总结与注意事项 06 · 008-toast-icon-compatibility

## 变更摘要

Change 008（用户命名 Change 007.1）将 Toast 前置状态图标从共享内联 SVG 改为 CSS 圆框和稳定字符，消除了默认 `error → x-circle` 在 Windows WebView2 中呈现为 `×` 的问题来源。Toast 事件、Store、文案、type、时序和 Go 调用链均未修改。

## 经验与踩坑

| 现象/坑 | 原因 | 结论/建议 |
|---|---|---|
| 成功状态文案前显示 `×` | 后端只传字符串，Store 默认 type 为 error，展示层映射到 `x-circle` | 在不改调用链约束下，限定修复于 Toast 表现层 |
| 看似图标兼容问题 | 实际是内联 SVG 正常绘制的错误型语义，不是字体缺字 | 审计应同时检查类型默认值与实际 SVG path |
| 仓库已有 `style.css` Toast 样式 | 入口只导入 `styles/global.css` | 只修改真实生效样式，避免无效双改 |
| Wails build 污染 bindings/runtime | CLI v2.16 高于 module v2.11.0 | 构建后先审计，再精确恢复生成噪音 |

## 对本项目后续的影响

- Toast 视觉不再依赖共享 Icon 组件；共享图标的其他使用点不受影响。
- 不新增依赖、配置、协议或持久化。
- 若未来允许调整 Toast 调用契约，可让成功型消息显式传入 `success`；本次因范围约束没有这样做。

## 遗留项与 TODO

- `NEEDS_USER`：在 Windows WebView2 真机上触发“鼠标穿透：已开启/已关闭”和失败提示，肉眼确认前置标记不再为 `×` 且排版正常。
- Change 007 与 008 均仍未提交，提交边界应保持二者可区分，且需用户明确授权才可 staging/commit。

## 参考资源

- `frontend/src/App.vue`
- `frontend/src/styles/global.css`
- `frontend/src/components/Icon.vue`
- `_dev/changes/007-clickthrough-toggle-feedback/04-validation.md`
