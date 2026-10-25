package app

import (
	"Q-Solver/pkg/answersummary"
	"context"
	"sync"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/questionbank"
)

type questionSolveCapture struct {
	mu          sync.Mutex
	app         *App
	bank        *questionBankService
	store       *questionbank.Store
	ctx         context.Context
	recordID    string
	sessionID   string
	result      answersummary.Result
	answer      string
	terminal    bool
	released    bool
	warned      bool
	passthrough func(string, ...interface{})
}

func (a *App) beginQuestionSolveCapture(ctx context.Context, cfg config.Config, images []string, passthrough func(string, ...interface{})) *questionSolveCapture {
	capture := &questionSolveCapture{app: a, ctx: ctx, passthrough: passthrough}
	b := a.questionBank
	if b == nil {
		return capture
	}
	b.mu.Lock()
	if b.closing || b.store == nil {
		b.mu.Unlock()
		return capture
	}
	capture.bank = b
	capture.store = b.store
	id, beginErr := capture.store.BeginWithContext(images, cfg.Model, cfg.DomainId, cfg.KeepContext)
	capture.recordID = id
	if id != "" {
		if session, _ := capture.store.Snapshot(); session != nil {
			capture.sessionID = session.ID
		}
		b.captures++
		b.workers.Add(1)
	}
	if beginErr != nil {
		b.message = "截图归档未完全保存，请勿退出并在题库面板重试保存"
	}
	b.mu.Unlock()
	if id != "" {
		a.notifyQuestionBank()
	}
	capture.warn(beginErr)
	if id == "" && beginErr != nil {
		a.EmitEvent("toast", "截图归档失败，本次解题仍继续", "error")
	}
	return capture
}

func (c *questionSolveCapture) warn(err error) {
	if err == nil || c.bank == nil {
		return
	}
	c.bank.mu.Lock()
	c.bank.message = "本场有未保存记录，请在题库面板重试保存，勿直接退出"
	c.bank.mu.Unlock()
	if !c.warned {
		c.warned = true
		c.app.EmitEvent("toast", "题库写入失败，请在题库面板重试保存", "error")
		c.app.notifyQuestionBank()
	}
}

func (c *questionSolveCapture) Emit(event string, data ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var text string
	if len(data) > 0 {
		text, _ = data[0].(string)
	}
	if c.terminal {
		return
	}
	if event == "solution" {
		c.answer = text
		c.result, _ = answersummary.Parse(text)
		return
	}
	if c.recordID == "" || c.store == nil {
		c.passthrough(event, data...)
		return
	}
	switch event {
	case "solution-stream-chunk":
		c.warn(c.store.Append(c.recordID, text))
	case "solution-error":
		status := questionbank.Failed
		if c.ctx.Err() != nil {
			status = questionbank.Cancelled
		}
		c.warn(c.store.Finish(c.recordID, "", status))
		c.terminal = true
	}
	c.passthrough(event, data...)
}

func (c *questionSolveCapture) RequiresPersistence() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.recordID != "" && c.store != nil
}

func (c *questionSolveCapture) Answer() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.answer
}

func (c *questionSolveCapture) Complete(solved bool, snapshot *questionbank.ConversationSnapshot) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	persisted := false
	if c.recordID == "" || c.store == nil {
		if !c.terminal && solved && c.answer != "" {
			c.passthrough("solution", c.answer, c.result)
		}
		c.terminal = true
		c.releaseLocked()
		return false
	}
	if !c.terminal {
		if solved {
			err := c.store.FinishWithConversationResult(c.recordID, c.answer, c.result.Items, questionbank.Completed, snapshot)
			c.warn(err)
			persisted = err == nil && c.answer != ""
			if persisted {
				c.passthrough("solution", c.answer, c.result)
			} else {
				c.passthrough("solution-error", "题库保存失败，请在题库面板重试保存")
			}
		} else {
			c.warn(c.store.Finish(c.recordID, "", questionbank.Interrupted))
		}
		c.terminal = true
	}
	c.releaseLocked()
	return persisted
}

func (c *questionSolveCapture) releaseLocked() {
	if c.released || c.bank == nil || c.recordID == "" {
		return
	}
	c.released = true
	c.bank.mu.Lock()
	c.bank.captures--
	c.bank.mu.Unlock()
	c.bank.workers.Done()
	c.app.notifyQuestionBank()
}
