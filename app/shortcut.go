package app

// StartRecordingKey 开始录制快捷键
func (a *App) StartRecordingKey(action string) {
	a.shortcutService.StartRecording(action)
}

// StopRecordingKey 停止录制快捷键
func (a *App) StopRecordingKey() {
	if a.shortcutService != nil {
		a.shortcutService.StopRecording()
	}
}

// ScrollContent 按 OutputMode 将滚动控制路由到桌面、手机或双端。
// Mobile 服务不可用时始终回退桌面；只接受 up/down，不转发其他事件或任意 Toast。
func (a *App) ScrollContent(direction string) {
	if direction != "up" && direction != "down" {
		return
	}

	decision := decideScrollRoute(a.mobileOutputMode(), a.mobileOutputEnabled())
	if decision.needMobile {
		if bridge := a.mobileBridgeForUse(); bridge != nil {
			bridge.PublishControl(direction)
		}
	}
	if decision.needDesktop {
		runtimeEmit(a.ctx, "scroll-content", direction)
	}
}

// CopyCode 复制代码
func (a *App) CopyCode() {
	a.EmitEvent("copy-code")
}
