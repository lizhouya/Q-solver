package app

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/questionbank"
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"time"
)

// BeforeClose covers the title-bar, tray and frontend quit paths.
func (a *App) BeforeClose(ctx context.Context) bool {
	b := a.questionBank
	if b == nil {
		return false
	}
	b.mu.Lock()
	var message string
	if b.store != nil {
		if err := b.store.Flush(); err != nil {
			message = "题库还有未保存内容。仍然退出会丢失尚未落盘的截图或答案，建议返回题库面板重试保存。"
		}
	}
	if message == "" && (b.captures > 0 || b.organizing) {
		message = "题库记录或整理仍在处理中。退出会中断当前任务；已保存的记录下次可恢复。"
	}
	b.mu.Unlock()
	if message == "" {
		return false
	}
	selected, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type: runtime.WarningDialog, Title: "题库尚未处理完成", Message: message,
		Buttons: []string{"返回保存", "仍然退出"}, DefaultButton: "返回保存", CancelButton: "返回保存",
	})
	return err != nil || selected != "仍然退出"
}
func (a *App) shutdownQuestionBank() {
	b := a.questionBank
	if b == nil {
		return
	}
	b.mu.Lock()
	b.closing = true
	if b.cancel != nil {
		b.cancel()
	}
	b.mu.Unlock()
	if a.taskManager != nil {
		a.taskManager.CancelCurrentTask()
	}
	done := make(chan struct{})
	go func() { b.workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.store != nil {
		session, _ := b.store.Snapshot()
		if session != nil {
			for _, r := range session.Records {
				if r.Status == questionbank.Running {
					_ = b.store.Finish(r.ID, "", questionbank.Interrupted)
				}
			}
		}
		if err := b.store.Flush(); err != nil {
			logger.Println("题库记录关闭时保存失败；未保存的数据可能丢失")
		}
	}
}
