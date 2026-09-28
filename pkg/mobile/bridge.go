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
	mu      sync.RWMutex
	state   Snapshot
	model   string
	subs    map[chan Snapshot]struct{}
	seq     int
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

// Reset 清空当前解题状态，回到空闲（供下一次解题开始）。
func (b *Bridge) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	b.state = Snapshot{Seq: b.seq, Ts: nowMillis(), Status: StatusIdle, Stage: StageContent}
	b.state.Payload.Model = b.model
	b.broadcastLocked()
}

// Publish 处理一条桌面解题输出事件。
func (b *Bridge) Publish(event string, data ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch event {
	case "solution-stream-start":
		b.seq++
		b.state = Snapshot{Seq: b.seq, Ts: nowMillis(), Status: StatusWaiting, Stage: StageThinking}
		b.state.Payload.Model = b.model

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

	b.broadcastLocked()
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
	return b.state
}

// Subscribe 注册一个新订阅者，返回接收快照的 buffered channel。
func (b *Bridge) Subscribe() chan Snapshot {
	ch := make(chan Snapshot, 64)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[ch] = struct{}{}
	return ch
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

// broadcastLocked 向全部订阅者广播当前快照（非阻塞，慢消费者丢帧）。
// 调用前必须持有 b.mu。
func (b *Bridge) broadcastLocked() {
	snapshot := b.state
	for ch := range b.subs {
		select {
		case ch <- snapshot:
		default: // 消费者过慢时丢弃这一帧，避免阻塞解题主流程
		}
	}
}

