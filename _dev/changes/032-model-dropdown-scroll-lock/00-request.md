# Change 032：模型下拉滚动锁定

## 用户反馈

设置窗口的模型列表打开后，在模型列表内使用鼠标滚轮或触控板时，外层模型设置页仍会滚动，导致外层内容滑到顶部或发生位置跳动。

## 已批准行为

用户选择“锁定外层页面”：

- 模型下拉打开时，外层模型设置页不响应滚轮、触控板或触摸滚动；
- 模型列表保持独立滚动；
- 列表滚动到顶部或底部时，不向外层传递滚动链；
- 选择模型、点击外部、切换页签或关闭设置窗口后恢复外层滚动。

## 范围

- 修改 `frontend/src/components/ModelSelect.vue`。
- 修改 `frontend/src/components/SettingsModal.vue`。
- 修改 `frontend/model_layout_regression_test.mjs`。
- 不修改 Go、CCR、Provider、模型请求、配置格式、题库或依赖。

## 成功标准

1. 下拉打开期间 `.model-tab` 被锁定。
2. 模型列表本身仍可滚动。
3. 滚轮、触控板和触摸滚动不会穿透到外层。
4. 所有关闭路径均解除锁定。
5. 前端回归、Vite、Go 测试和构建通过。
6. 生成独立 Change 032 候选，不覆盖当前稳定版。

## Git 边界

用户批准设计和本地实施验证；本 Change 当前不包含 commit、push、Tag、Release 或 RC 授权。
