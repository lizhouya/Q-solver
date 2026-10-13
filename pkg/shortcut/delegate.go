package shortcut

// ServiceDelegate 定义了 Shortcut Service 需要 App 配合做的事情
type ServiceDelegate interface {
	TriggerSolve()
	TriggerScreenshot()
	TriggerSend()
	TriggerDeleteScreenshot()
	ToggleVisibility()
	ToggleClickThrough() (bool, error)
	MoveWindow(dx, dy int)
	ScrollContent(direction string)
	// ToggleWindowVisible 切换窗口实体可见性（003 Phase 2）。
	// 返回切换后状态字符串与错误；与 F9 防录屏隐身（ToggleVisibility）完全正交。
	ToggleWindowVisible() (string, error)
	EmitEvent(eventName string, data ...interface{})
}
