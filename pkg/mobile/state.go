package mobile

import (
	"time"
)

// 解题状态枚举（镜像桌面事件语义，供手机端展示）
const (
	StatusIdle      = "idle"      // 空闲
	StatusWaiting   = "waiting"   // 等待/开始
	StatusStreaming = "streaming" // 流式中
	StatusDone      = "done"      // 完成
	StatusError     = "error"     // 出错
)

// Stage 阶段标识（用于手机页区分分析过程 / 正文）
const (
	StageThinking = "thinking" // 思考链 / 分析过程
	StageContent  = "content"  // 正文 / 最终内容
)

// Payload 手机端展示载荷。D-Mobile-003：仅「答案 + 简短分析」，不传输完整思考链。
type Payload struct {
	Question string `json:"question"` // 当前题目（第一版为空，占位；不传输截图）
	Analysis string `json:"analysis"` // 简短分析（映射 solution-stream-thinking 累积）
	Answer   string `json:"answer"`   // 最终答案（映射 solution-stream-chunk / solution）
	Model    string `json:"model"`    // 当前模型名
}

// ShortcutHint 是可安全发送到手机端的快捷键提示。
// 只包含用户可见的动作标签和快捷键显示名，不包含 VK/ComboID 或其他配置。
type ShortcutHint struct {
	Label   string `json:"label"`
	KeyName string `json:"keyName"`
}

// Snapshot 统一状态快照（全量，手机端断线重连时先收到当前 Snapshot 自动对齐）。
// Control/ControlSeq 与解题 Seq 相互独立：Control 是一次性滚动命令，ControlSeq 用于去重。
type Snapshot struct {
	Seq        int            `json:"seq"`               // 解题快照序号，单调递增
	Ts         int64          `json:"ts"`                // Unix 毫秒
	Status     string         `json:"status"`            // waiting | streaming | done | error | idle
	Stage      string         `json:"stage"`             // thinking | content
	Payload    Payload        `json:"payload,omitempty"` // 载荷
	Error      string         `json:"error,omitempty"`   // 错误信息
	Hints      []ShortcutHint `json:"hints,omitempty"`   // 可重放的快捷键显示元数据
	ControlSeq int            `json:"controlSeq"`        // 一次性控制序号，独立单调递增
	Control    string         `json:"control,omitempty"` // 仅实时帧携带：up | down
}

// newSnapshot 构造一帧全量快照。
func newSnapshot(seq int, status, stage string, payload Payload) Snapshot {
	return Snapshot{
		Seq:     seq,
		Status:  status,
		Stage:   stage,
		Payload: payload,
	}
}

// heartbeat/keepalive 常量（SSE 注释行保活）
const sseHeartbeatInterval = 20 * time.Second
