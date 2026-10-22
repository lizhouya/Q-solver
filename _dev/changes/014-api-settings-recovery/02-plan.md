# API 配置恢复实施计划

**目标：** 恢复配置读取、单次 API 接入和独立保存。
**架构：** ConfigManager 加载幂等且失败关闭保存通道；前端配置与场景独立加载。新增显式单次连接 API，旧绑定保留兼容；题库保存与普通保存分别呈现结果。
**技术栈：** Go、Wails v2.11、Vue/Pinia、Node 标准库测试，无新增依赖。
**依据：** 00-request.md、01-current-state.md。主线程逐项执行，不派发代理，不提交。

## 任务 1：配置安全读取
文件：pkg/config/manager.go、pkg/config/manager_test.go、app/settings.go、app/settings_test.go、app/app.go。
- [x] 临时目录测试加密配置成功读取、坏文件报错且 Save/Update/Patch 不覆盖、读取失败后修复文件可重读、重复 Load 不重置状态。
- [x] Load 成功后标记 loaded；失败返回固定错误；GetSettings 调用 Load 并向 Wails 返回错误；Startup 将配置读取移至题库恢复之前。
- [x] Save 未加载时拒绝；UpdateFromJSON/Patch 在更新内存前确保 Load 成功。

```go
func (a *App) GetSettings() (config.Config, error) {
    if err := a.configManager.Load(); err != nil { return config.Config{}, err }
    return a.configManager.Get(), nil
}
```

## 任务 2：单次接入参数
文件：pkg/llm/service.go、pkg/llm/service_test.go、app/llm.go、frontend/src/services/api.js、frontend/wailsjs/go/app/App.js、App.d.ts。
- [x] httptest 使用两个本地 server 验证新 Base URL 确实命中草稿端点，已保存 provider 不变。
- [x] 新增 GetModelsForConnection(apiKey, baseURL string) 与 TestConnectionForConnection(apiKey, baseURL, model string)，App 到 Service 同名传递，旧方法保持兼容。
- [x] 单次配置仅取 API Key/Base URL/模型，拒绝空 Key、无效 http(s) URL 与 URL 内嵌凭据；15 秒 timeout；错误不回显私人地址或鉴权信息。

## 任务 3：前端状态与独立保存
文件：frontend/src/stores/settings.js、frontend/src/components/SettingsModal.vue、frontend/settings_regression_test.mjs。
- [x] Node VM 加实际 Vue/Pinia 的无网络测试覆盖场景失败仍加载、加载失败禁止显式/静默保存、重试恢复、草稿地址传递、失败保存不更新本地状态。
- [x] 提供 settingsLoaded/settingsLoading/settingsLoadError 与重读入口，加载不再隐式获取模型；合并后端快照保留未展示字段。
- [x] saveSettings({close = true}) 返回布尔；SettingsModal 只保存 dirty 的题库偏好，普通保存独立执行，部分成功保持窗口并提示。
- [x] 模型请求使用冻结的草稿参数；请求版本号防止旧结果覆盖新设置。

## 验证与收尾
- [x] node frontend/settings_regression_test.mjs；go test ./pkg/config ./pkg/llm ./app -count=1；go test ./... -count=1；go vet ./...；go build ./...；npm --prefix frontend run build。
- [x] Wails 备用输出名生产构建；构建前保存 runtime 生成文件，构建后只恢复工具链噪音。不覆盖旧 EXE，新修复版保留供用户启动。
- [x] git diff --check，文档/审计/README 同步。人工验收与 race 阻塞单列。

## 回退
定向撤销本 Change 增量，保留既有题库/Mobile 修改，不触碰原密钥配置或旧输出。
