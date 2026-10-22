package app

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/questionbank"
	"context"
)

// captureQuestion records before desktop/mobile routing and retains its own record ID.
func (a *App) captureQuestion(ctx context.Context, cfg config.Config, images []string, passthrough func(string, ...interface{})) (func(string, ...interface{}), func()) {
	b := a.questionBank
	if b == nil {
		return passthrough, func() {}
	}
	b.mu.Lock()
	if b.closing || b.store == nil {
		b.mu.Unlock()
		return passthrough, func() {}
	}
	store := b.store
	id, beginErr := store.Begin(images, cfg.Model, cfg.DomainId)
	if beginErr != nil {
		b.message = "截图归档未完全保存，请勿退出并在题库面板重试保存"
	}
	if id == "" {
		b.mu.Unlock()
		if beginErr != nil {
			a.EmitEvent("toast", "截图归档失败，本次解题仍继续", "error")
		}
		return passthrough, func() {}
	}
	b.captures++
	b.workers.Add(1)
	b.mu.Unlock()
	a.notifyQuestionBank()
	warned := false
	warn := func(err error) {
		if err == nil {
			return
		}
		b.mu.Lock()
		b.message = "本场有未保存记录，请在题库面板重试保存，勿直接退出"
		b.mu.Unlock()
		if !warned {
			warned = true
			a.EmitEvent("toast", "题库写入失败，请在题库面板重试保存", "error")
			a.notifyQuestionBank()
		}
	}
	warn(beginErr)
	emit := func(event string, data ...interface{}) {
		var text string
		if len(data) > 0 {
			text, _ = data[0].(string)
		}
		switch event {
		case "solution-stream-chunk":
			warn(store.Append(id, text))
		case "solution":
			warn(store.Finish(id, text, questionbank.Completed))
		case "solution-error":
			status := questionbank.Failed
			if ctx.Err() != nil {
				status = questionbank.Cancelled
			}
			warn(store.Finish(id, "", status))
		}
		passthrough(event, data...)
	}
	finish := func() {
		warn(store.Finish(id, "", questionbank.Interrupted))
		b.mu.Lock()
		b.captures--
		b.mu.Unlock()
		b.workers.Done()
		a.notifyQuestionBank()
	}
	return emit, finish
}
