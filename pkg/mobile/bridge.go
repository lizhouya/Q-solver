package mobile

import (
	"Q-Solver/pkg/answersummary"
	"sync"
)

// Bridge 是桌面端事件进 Mobile 端的唯一入口。
// 它维护当前全量快照，并把白名单业务事件广播给 SSE 订阅者。
type Bridge struct {
	mu         sync.RWMutex
	state      Snapshot
	model      string
	hints      []ShortcutHint
	subs       map[chan Snapshot]struct{}
	sessionID  string
	clearSeq   uint64
	seq        int
	controlSeq int
	closed     bool
}

// NewBridge 创建事件接入桥。每个实例使用独立的随机 sessionId；initialClearSeq
// 来自已持久化的 Mobile 状态，使首帧即可重放当前清屏边界。
func NewBridge(initialClearSeq uint64) (*Bridge, error) {
	sessionID, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	b := &Bridge{
		subs:      make(map[chan Snapshot]struct{}),
		sessionID: sessionID,
		clearSeq:  initialClearSeq,
	}
	b.state = b.baseSnapshot(StatusIdle, StageContent)
	return b, nil
}

// CurrentClearSeq 返回当前显示代际。
func (b *Bridge) CurrentClearSeq() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.clearSeq
}

// SetModel 记录当前模型名（配置变更时由 App 调用），写入快照载荷。
func (b *Bridge) SetModel(model string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.model = model
	b.state.Payload.Model = model
}

// SetHints 更新手机端可重放的快捷键提示。
// 调用方只能传入动作标签和显示名；本方法复制切片，避免外部后续修改快照内容。
func (b *Bridge) SetHints(hints []ShortcutHint) {
	clean := make([]ShortcutHint, 0, len(hints))
	for _, hint := range hints {
		if hint.Label == "" || hint.KeyName == "" {
			continue
		}
		clean = append(clean, ShortcutHint{Label: hint.Label, KeyName: hint.KeyName})
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || hintsEqual(b.hints, clean) {
		return
	}
	b.hints = clean
	b.state.Hints = cloneHints(clean)
	b.broadcastLocked(false)
}

// Reset 清空当前解题状态，回到空闲，但不推进 clearSeq。
// ClearDisplay 用于需要清除远端页面会话内容的可靠清屏。
func (b *Bridge) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.seq++
	b.state = b.baseSnapshot(StatusIdle, StageContent)
	b.broadcastLocked(false)
}

// Publish 处理一条桌面解题输出事件，并使用调用时的当前显示代际。
func (b *Bridge) Publish(event string, data ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.publishLocked(event, data)
}

// PublishIfCurrent 仅在 expected 与当前 clearSeq 相等时处理事件。
// 代际检查、状态变更和广播在同一把锁内完成，因此 ClearDisplay 之后的旧任务
// 无法在检查与发布之间穿透清屏边界。
func (b *Bridge) PublishIfCurrent(expected uint64, event string, data ...interface{}) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || expected != b.clearSeq {
		return false
	}
	return b.publishLocked(event, data)
}

// publishLocked 应用一条白名单业务事件。调用前必须持有 b.mu。
func (b *Bridge) publishLocked(event string, data []interface{}) bool {
	switch event {
	case "solution-stream-start":
		b.seq++
		b.state = b.baseSnapshot(StatusWaiting, StageThinking)

	case "solution-stream-thinking":
		b.applyStreamBody(&b.state.Payload.Analysis, data)
		b.seq++
		b.state.Ts = nowMillis()
		b.state.Status = StatusStreaming
		b.state.Stage = StageThinking

	case "solution-stream-chunk":
		b.applyStreamBody(&b.state.Payload.Answer, data)
		b.seq++
		b.state.Ts = nowMillis()
		b.state.Status = StatusStreaming
		b.state.Stage = StageContent

	case "solution":
		b.seq++
		b.state.Ts = nowMillis()
		b.state.Status = StatusDone
		b.state.Stage = StageContent
		if len(data) > 0 {
			if value, ok := data[0].(string); ok {
				b.state.Payload.Answer = value
			}
		}
		b.state.Error = ""

	case "solution-error":
		b.seq++
		b.state.Ts = nowMillis()
		b.state.Status = StatusError
		b.state.Stage = StageContent
		if len(data) > 0 {
			if value, ok := data[0].(string); ok {
				b.state.Error = value
			}
		}

	default:
		return false
	}

	if event != "solution" || len(data) < 2 {
		b.state.Payload.AnswerItems = nil
		b.state.Payload.AnswerCopyText = ""
	} else {
		b.state.Payload.AnswerItems = nil
		b.state.Payload.AnswerCopyText = ""
		if result, ok := data[1].(answersummary.Result); ok {
			b.state.Payload.AnswerItems = append([]answersummary.Item(nil), result.Items...)
			b.state.Payload.AnswerCopyText, _ = answersummary.BuildCopyText(result.Items)
		}
	}
	b.state.Seq = b.seq
	b.broadcastLocked(false)
	return true
}

// ClearDisplay 推进显示代际，清空所有订阅者的旧积压，并强制放入一帧空白屏障。
// model、hints 与 controlSeq 是独立的可重放元数据，清屏时予以保留。
func (b *Bridge) ClearDisplay(next uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || next <= b.clearSeq {
		return
	}

	b.clearSeq = next
	b.seq++
	b.state = b.baseSnapshot(StatusIdle, StageContent)
	barrier := cloneSnapshot(b.state)
	for ch := range b.subs {
		drainSnapshots(ch)
		// 队列刚刚在同一把锁内排空，必定能容纳屏障帧。
		ch <- barrier
	}
}

// PublishControl 广播一次手机端滚动控制。
// 仅接受 up/down；控制序号独立于解题 Seq，并在广播后立即清空 Control，避免后续快照重放。
func (b *Bridge) PublishControl(direction string) bool {
	if direction != "up" && direction != "down" {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return false
	}
	b.controlSeq++
	b.state.ControlSeq = b.controlSeq
	b.state.Control = direction
	b.broadcastLocked(true)
	b.state.Control = ""
	return true
}

func (b *Bridge) applyStreamBody(dst *string, data []interface{}) {
	if len(data) == 0 {
		return
	}
	if value, ok := data[0].(string); ok {
		*dst += value
	}
}

// GetState 返回当前快照副本（供 SSE 连接初始对齐使用）。
func (b *Bridge) GetState() Snapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return cloneSnapshot(b.state)
}

// Subscribe 原子注册新订阅者并返回与注册时刻一致的当前快照。
// Bridge 已关闭时返回一个已关闭 channel，避免关闭期间注册泄漏。
func (b *Bridge) Subscribe() (ch chan Snapshot, initial Snapshot) {
	ch = make(chan Snapshot, 64)
	b.mu.Lock()
	defer b.mu.Unlock()
	initial = cloneSnapshot(b.state)
	if b.closed {
		close(ch)
		return ch, initial
	}
	b.subs[ch] = struct{}{}
	return ch, initial
}

// Unsubscribe 注销订阅者并关闭其 channel。
func (b *Bridge) Unsubscribe(ch chan Snapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subs[ch]; ok {
		delete(b.subs, ch)
		close(ch)
	}
}

// Close 关闭全部订阅并拒绝后续发布。该操作幂等且不等待网络 I/O。
func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}
}

// baseSnapshot 构造新解题快照，同时保留会话、清屏边界、提示和控制序号。
// 调用前必须持有 b.mu，构造初始化状态时除外。
func (b *Bridge) baseSnapshot(status, stage string) Snapshot {
	return Snapshot{
		SessionID:  b.sessionID,
		ClearSeq:   b.clearSeq,
		Seq:        b.seq,
		Ts:         nowMillis(),
		Status:     status,
		Stage:      stage,
		Payload:    Payload{Model: b.model},
		Hints:      cloneHints(b.hints),
		ControlSeq: b.controlSeq,
	}
}

func hintsEqual(left, right []ShortcutHint) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func cloneHints(hints []ShortcutHint) []ShortcutHint {
	if len(hints) == 0 {
		return nil
	}
	return append([]ShortcutHint(nil), hints...)
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.Hints = cloneHints(snapshot.Hints)
	snapshot.Payload.AnswerItems = append([]answersummary.Item(nil), snapshot.Payload.AnswerItems...)
	return snapshot
}

func drainSnapshots(ch chan Snapshot) {
	for {
		select {
		case <-ch:
			continue
		default:
			return
		}
	}
}

// broadcastLocked 向全部订阅者广播当前快照且不阻塞解题主流程。
// 控制帧在队列满时替换积压快照；流式中间帧可丢弃；done/error 必须排队，
// 并保留尚未发送的控制帧。调用前必须持有 b.mu。
func (b *Bridge) broadcastLocked(control bool) {
	snapshot := cloneSnapshot(b.state)
	for ch := range b.subs {
		select {
		case ch <- cloneSnapshot(snapshot):
			continue
		default:
		}
		if !control {
			if snapshot.Status != StatusDone && snapshot.Status != StatusError {
				continue
			}
			pending := make([]Snapshot, 0, cap(ch))
			for len(ch) > 0 {
				select {
				case queued := <-ch:
					if queued.Control != "" {
						pending = append(pending, queued)
					}
				default:
				}
			}
			if len(pending) >= cap(ch) {
				pending = pending[len(pending)-cap(ch)+1:]
			}
			for _, queued := range pending {
				ch <- queued
			}
			ch <- cloneSnapshot(snapshot)
			continue
		}
		drainSnapshots(ch)
		select {
		case ch <- cloneSnapshot(snapshot):
		default:
		}
	}
}
