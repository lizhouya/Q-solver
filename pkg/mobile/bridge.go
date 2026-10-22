package mobile

import (
	"sync"
)

// Bridge 是桌面端事件进 Mobile 端的唯一入口。
// 职责：消费 App 的解题输出事件（solution-stream-*、solution、solution-error），
// 维护「当前最新快照」，并广播给所有已注册的 SSE 订阅者。
//
// 事件 → 快照 映射规则（对应 01-current-state §2.3 的事件名）：
//   - solution-stream-start    → 置为等待，清空载荷
//   - solution-stream-thinking → 流式中，stage=thinking，Analysis 累积
//   - solution-stream-chunk    → 流式中，stage=content，Answer 累积
//   - solution                  → 完成，Answer = 最终正文
//   - solution-error            → 出错，Error
type Bridge struct {
	mu         sync.RWMutex
	state      Snapshot
	model      string
	hints      []ShortcutHint
	subs       map[chan Snapshot]struct{}
	seq        int
	controlSeq int
}

// NewBridge 创建事件接入桥。
func NewBridge() *Bridge {
	return &Bridge{
		state: Snapshot{Status: StatusIdle, Stage: StageContent},
		subs:  make(map[chan Snapshot]struct{}),
	}
}

// SetModel 记录当前模型名（配置变更时由 App 调用），写入快照载荷。
func (b *Bridge) SetModel(model string) {
	b.mu.Lock()
	defer b.mu.Unlock()
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
	if hintsEqual(b.hints, clean) {
		return
	}
	b.hints = clean
	b.state.Hints = cloneHints(clean)
	b.broadcastLocked(false)
}

// Reset 清空当前解题状态，回到空闲（供下一次解题开始）。
func (b *Bridge) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	b.state = b.baseSnapshot(StatusIdle, StageContent)
	b.broadcastLocked(false)
}

// Publish 处理一条桌面解题输出事件。
func (b *Bridge) Publish(event string, data ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

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
			if s, ok := data[0].(string); ok {
				b.state.Payload.Answer = s
			}
		}
		b.state.Error = ""

	case "solution-error":
		b.seq++
		b.state.Ts = nowMillis()
		b.state.Status = StatusError
		b.state.Stage = StageContent
		if len(data) > 0 {
			if s, ok := data[0].(string); ok {
				b.state.Error = s
			}
		}

	default:
		// 非解题输出事件不进入 Bridge（路由过滤在上层 App.EmitEvent 完成）。
		return
	}

	// 每个业务事件的快照都使用最新序号，供浏览器与控制帧区分。
	b.state.Seq = b.seq
	b.broadcastLocked(false)
}

// PublishControl 广播一次手机端滚动控制。
// 仅接受 up/down；控制序号独立于解题 Seq，并在广播后立即清空 Control，避免后续快照重放。
func (b *Bridge) PublishControl(direction string) bool {
	if direction != "up" && direction != "down" {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()
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
	if s, ok := data[0].(string); ok {
		*dst += s
	}
}

// GetState 返回当前快照副本（供 SSE 连接初始对齐使用）。
func (b *Bridge) GetState() Snapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return cloneSnapshot(b.state)
}

// Subscribe 原子注册新订阅者并返回与注册时刻一致的当前快照。
// 注册完成后的 Publish 要么已体现在 initial 中，要么会进入返回的 channel。
func (b *Bridge) Subscribe() (ch chan Snapshot, initial Snapshot) {
	ch = make(chan Snapshot, 64)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[ch] = struct{}{}
	initial = cloneSnapshot(b.state)
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

// baseSnapshot 构造新解题快照，同时保留可重放提示和独立控制序号。
// 调用前必须持有 b.mu。
func (b *Bridge) baseSnapshot(status, stage string) Snapshot {
	return Snapshot{
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
	return snapshot
}

// broadcastLocked 向全部订阅者广播当前快照且不阻塞解题主流程。
// 控制帧在队列满时替换积压的全量业务快照，避免滚动命令被静默丢弃；
// 流式中间帧可丢弃；done/error 必须排队，因为后续可能不再有业务帧。
// 调用前必须持有 b.mu。
func (b *Bridge) broadcastLocked(control bool) {
	snapshot := cloneSnapshot(b.state)
	for ch := range b.subs {
		select {
		case ch <- snapshot:
			continue
		default:
		}
		if !control {
			if snapshot.Status != StatusDone && snapshot.Status != StatusError {
				continue
			}
			// 终态取代过时业务快照，同时保留尚未发送的滚动命令。
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
			// 极端全控制积压时保留最新命令，为最终答案预留一格。
			if len(pending) >= cap(ch) {
				pending = pending[len(pending)-cap(ch)+1:]
			}
			for _, queued := range pending {
				ch <- queued
			}
			ch <- snapshot
			continue
		}
		for {
			select {
			case <-ch:
				continue
			default:
			}
			break
		}
		select {
		case ch <- snapshot:
		default:
		}
	}
}
