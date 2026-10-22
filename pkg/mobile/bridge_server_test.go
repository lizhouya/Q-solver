package mobile

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestBridgeEventSequence 验证 Bridge 状态机：事件流 → 期望快照。
func TestBridgeEventSequence(t *testing.T) {
	b := NewBridge()
	b.SetModel("gpt-4o")

	// 1) 开始
	b.Publish("solution-stream-start")
	b.Publish("solution-stream-thinking", "先分析题干")
	b.Publish("solution-stream-chunk", "答案第一部分")
	b.Publish("solution-stream-chunk", "答案第二部分")
	b.Publish("solution")

	st := b.GetState()
	if st.Status != StatusDone {
		t.Fatalf("期望 done，得到 %s", st.Status)
	}
	if st.Payload.Model != "gpt-4o" {
		t.Fatalf("期望模型 gpt-4o, 得到 %s", st.Payload.Model)
	}
	if !strings.Contains(st.Payload.Analysis, "先分析题干") {
		t.Fatalf("分析累积缺失: %q", st.Payload.Analysis)
	}
	if st.Payload.Answer != "答案第一部分答案第二部分" {
		t.Fatalf("答案累积错误: %q", st.Payload.Answer)
	}
	if st.Stage != StageContent {
		t.Fatalf("期望 stage=content, 得到 %s", st.Stage)
	}

	// 2) 出错
	b.Publish("solution-error", "context canceled")
	st = b.GetState()
	if st.Status != StatusError || !strings.Contains(st.Error, "canceled") {
		t.Fatalf("错误状态不正确: %+v", st)
	}
}

type subscriptionObserverWriter struct {
	header     http.Header
	bridge     *Bridge
	firstFlush chan int
	once       sync.Once
}

func (w *subscriptionObserverWriter) Header() http.Header {
	return w.header
}

func (w *subscriptionObserverWriter) WriteHeader(statusCode int) {}

func (w *subscriptionObserverWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (w *subscriptionObserverWriter) Flush() {
	w.bridge.mu.RLock()
	count := len(w.bridge.subs)
	w.bridge.mu.RUnlock()
	w.once.Do(func() { w.firstFlush <- count })
}

func TestSSESubscribesBeforeInitialFlush(t *testing.T) {
	b := NewBridge()
	s, err := NewServer(b)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/sse?token="+s.connectToken(), nil).WithContext(ctx)
	writer := &subscriptionObserverWriter{
		header:     make(http.Header),
		bridge:     b,
		firstFlush: make(chan int, 1),
	}
	done := make(chan struct{})
	go func() {
		s.handleSSE(writer, req)
		close(done)
	}()

	select {
	case count := <-writer.firstFlush:
		if count != 1 {
			t.Fatalf("首次 Flush 时应已有 1 个订阅者，实际 %d", count)
		}
	case <-time.After(time.Second):
		t.Fatal("等待首次 SSE Flush 超时")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("取消请求后 SSE handler 未退出")
	}
}

func TestBridgeHintsAndOneShotControls(t *testing.T) {
	b := NewBridge()
	initialSeq := b.GetState().Seq
	b.SetHints([]ShortcutHint{
		{Label: "向上滚动", KeyName: "Alt+PgUp"},
		{Label: "向下滚动", KeyName: "Alt+PgDn"},
		{}, // 空提示不会进入快照
	})
	if got := b.GetState().Seq; got != initialSeq {
		t.Fatalf("提示元数据不得递增解题 Seq：更新前 %d，更新后 %d", initialSeq, got)
	}
	b.Publish("solution-stream-start")

	before := b.GetState()
	if len(before.Hints) != 2 || before.Hints[0].Label != "向上滚动" || before.Hints[0].KeyName != "Alt+PgUp" {
		t.Fatalf("快捷键提示不正确: %+v", before.Hints)
	}
	answerSeq := before.Seq

	sub, _ := b.Subscribe()
	defer b.Unsubscribe(sub)
	if !b.PublishControl("up") {
		t.Fatal("up 应为有效控制")
	}
	control := <-sub
	if control.Control != "up" || control.ControlSeq != 1 {
		t.Fatalf("首次控制应为 up/controlSeq=1，实际 %+v", control)
	}
	if control.Seq != answerSeq {
		t.Fatalf("控制不得递增解题 Seq：控制前 %d，控制后 %d", answerSeq, control.Seq)
	}

	if !b.PublishControl("down") {
		t.Fatal("down 应为有效控制")
	}
	control = <-sub
	if control.Control != "down" || control.ControlSeq != 2 {
		t.Fatalf("第二次控制应为 down/controlSeq=2，实际 %+v", control)
	}

	state := b.GetState()
	if state.ControlSeq != 2 || state.Control != "" {
		t.Fatalf("当前快照应保留控制序号但清除一次性命令，实际 %+v", state)
	}
	if b.PublishControl("left") {
		t.Fatal("非法方向不得发布")
	}
	if got := b.GetState().ControlSeq; got != 2 {
		t.Fatalf("非法方向不得递增 controlSeq，实际 %d", got)
	}

	b.Reset()
	state = b.GetState()
	if state.ControlSeq != 2 || state.Control != "" || len(state.Hints) != 2 {
		t.Fatalf("Reset 应保留提示和控制基线且不重放命令，实际 %+v", state)
	}
}

func TestControlReplacesBacklogInsteadOfBeingDropped(t *testing.T) {
	b := NewBridge()
	sub, _ := b.Subscribe()
	defer b.Unsubscribe(sub)
	for i := 0; i < cap(sub); i++ {
		b.Publish("solution-stream-chunk", "x")
	}
	if len(sub) != cap(sub) {
		t.Fatalf("测试前提：订阅队列未填满，len=%d cap=%d", len(sub), cap(sub))
	}
	if !b.PublishControl("down") {
		t.Fatal("down 应为有效控制")
	}
	select {
	case snapshot := <-sub:
		if snapshot.Control != "down" || snapshot.ControlSeq != 1 {
			t.Fatalf("积压队列中的控制帧错误: %+v", snapshot)
		}
	default:
		t.Fatal("队列满时滚动控制被丢弃")
	}
}

func TestBridgeRejectsUnlistedEvents(t *testing.T) {
	b := NewBridge()
	before := b.GetState()
	b.Publish("toast", "不应发送到手机")
	after := b.GetState()
	if after.Seq != before.Seq || after.ControlSeq != before.ControlSeq || after.Error != before.Error {
		t.Fatalf("非白名单事件不应改变 Mobile 快照：before=%+v after=%+v", before, after)
	}
}

func readSSEFrame(t *testing.T, reader *bufio.Reader) (string, Snapshot, string) {
	t.Helper()
	var event string
	var data string
	var raw strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("读取 SSE 帧失败: %v，已读取 %q", err, raw.String())
		}
		raw.WriteString(line)
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		t.Fatalf("解析 SSE 快照失败: %v，原始 %q", err, raw.String())
	}
	return event, snapshot, raw.String()
}

func TestSSEInitialSnapshotOnlySyncsControlSequence(t *testing.T) {
	b := NewBridge()
	b.SetHints([]ShortcutHint{{Label: "向下滚动", KeyName: "Alt+PgDn"}})
	if !b.PublishControl("down") {
		t.Fatal("测试前提：down 应发布成功")
	}

	s, err := NewServer(b)
	if err != nil {
		t.Fatal(err)
	}
	s.port = 0
	if err := s.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	defer s.Stop()

	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(s.port)+"/sse?token="+s.connectToken(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE 连接失败: %v", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)

	event, initial, raw := readSSEFrame(t, reader)
	if event != "snapshot" {
		t.Fatalf("首帧事件应为 snapshot，实际 %q，原始 %q", event, raw)
	}
	if initial.ControlSeq != 1 || initial.Control != "" {
		t.Fatalf("首帧只能同步 controlSeq，不得重放旧控制，实际 %+v", initial)
	}
	if !strings.Contains(raw, `"controlSeq":1`) || strings.Contains(raw, `"control":"down"`) {
		t.Fatalf("首帧 JSON 控制字段不符合约束，原始 %q", raw)
	}

	b.PublishControl("up")
	event, realtime, raw := readSSEFrame(t, reader)
	if event != "message" || realtime.ControlSeq != 2 || realtime.Control != "up" {
		t.Fatalf("实时帧应携带新控制，实际 event=%q snapshot=%+v raw=%q", event, realtime, raw)
	}
}

// TestSSEHandshakeAndPublish 端到端验证 SSE：token 校验 + 连接即快照 + 实时推送。
func TestSSEHandshakeAndPublish(t *testing.T) {
	b := NewBridge()
	s, err := NewServer(b)
	if err != nil {
		t.Fatal(err)
	}

	// 找一个空闲端口供测试（避免 38511 冲突）
	s.port = 0

	if err := s.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	defer s.Stop()

	// 用真实 token 连 SSE
	req, _ := http.NewRequest("GET", "http://127.0.0.1:"+strconv.Itoa(s.port)+"/sse?token="+s.connectToken(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE 连接失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("期望 200, 得到 %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("期望 text/event-stream, 得到 %s", ct)
	}

	// 发送一帧事件；HTTP 响应可见前应已完成订阅注册，无需人为等待。
	b.Publish("solution-stream-start")

	done := make(chan struct{})
	timeout := time.After(3 * time.Second)
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 512)
	go func() {
		for {
			n, err := resp.Body.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				if strings.Contains(string(buf), `"waiting"`) {
					close(done)
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-done:
		raw := string(buf)
		if !strings.Contains(raw, `"status":"waiting"`) {
			t.Fatalf("SSE 数据中应包含 waiting 帧, 原始: %q", raw)
		}
		// 验证「连接即快照」语义：应同时收到初始 snapshot 帧（idle）与服务端推送的 message 帧
		if !strings.Contains(raw, `event: snapshot`) {
			t.Fatalf("缺少初始 snapshot 帧, 原始: %q", raw)
		}
	case <-timeout:
		t.Fatalf("等待 SSE 数据超时，收到: %q", string(buf))
	}
}

// TestUnauthorizedSSE 验证错误 token 返回 401。
func TestUnauthorizedSSE(t *testing.T) {
	b := NewBridge()
	s, _ := NewServer(b)
	s.port = 0
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(s.port) + "/sse?token=WRONG")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("期望 401, 得到 %d", resp.StatusCode)
	}
}

// TestIndexHTML 验证根路径返回 embed 手机页：
//
//	HTTP 200 + text/html + body 含固定文本，且不依赖 token（HTML 页面本身无鉴权，token 只保护 /sse）。
func TestIndexHTML(t *testing.T) {
	b := NewBridge()
	s, _ := NewServer(b)
	s.port = 0
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// 带 token
	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(s.port) + "/?token=" + s.connectToken())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("期望 200, 得到 %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("期望 Content-Type text/html, 得到 %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatal("返回空 body（空白页根因候选：embed 未命中或读为空）")
	}
	for _, mark := range []string{
		"Q-Solver", "手机同步端", "EventSource",
		"shortcut-hints", "syncControlBaseline", "applyRealtimeControl", "viewportHeight * 0.7",
	} {
		if !strings.Contains(string(body), mark) {
			t.Fatalf("页面缺少固定文本 %q（可能命中降级占位页）", mark)
		}
	}

	// 不带 token 也应返回首页（token 只保护 /sse）
	resp2, err := http.Get("http://127.0.0.1:" + strconv.Itoa(s.port) + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("无 token 时应返回首页 200, 得到 %d", resp2.StatusCode)
	}
}

// TestHealth 健康端点。
func TestHealth(t *testing.T) {
	b := NewBridge()
	s, _ := NewServer(b)
	s.port = 0
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(s.port) + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("健康检查异常: %s", body)
	}
}

// 真实 Bridge 的业务帧必须逐帧增序；不能只用手造递增序号验证浏览器。
func TestBusinessSnapshotSequenceAdvancesOnEveryEvent(t *testing.T) {
	b := NewBridge()
	sub, initial := b.Subscribe()
	defer b.Unsubscribe(sub)
	previous := initial.Seq
	for _, event := range []string{"solution-stream-start", "solution-stream-thinking", "solution-stream-chunk", "solution", "solution-stream-start", "solution-error"} {
		b.Publish(event, "synthetic")
		snap := <-sub
		if snap.Seq <= previous {
			t.Fatalf("%s snapshot seq=%d, previous=%d", event, snap.Seq, previous)
		}
		if snap.Seq != b.GetState().Seq {
			t.Fatal("published and stored snapshot diverged")
		}
		previous = snap.Seq
	}
}

func TestTerminalSnapshotSurvivesFullQueue(t *testing.T) {
	for _, terminal := range []string{"solution", "solution-error"} {
		t.Run(terminal, func(t *testing.T) {
			b := NewBridge()
			sub, _ := b.Subscribe()
			defer b.Unsubscribe(sub)
			b.PublishControl("down")
			for len(sub) < cap(sub) {
				b.Publish("solution-stream-chunk", "x")
			}
			b.Publish(terminal, "synthetic-final")
			foundTerminal, foundControl := false, false
			for len(sub) > 0 {
				snap := <-sub
				if snap.Control == "down" {
					foundControl = true
				}
				if snap.Status == StatusDone || snap.Status == StatusError {
					foundTerminal = true
				}
			}
			if !foundTerminal {
				t.Fatal("full queue dropped final result")
			}
			if !foundControl {
				t.Fatal("terminal delivery discarded pending scroll control")
			}
		})
	}
}

func TestRealtimeSSEToDOMWithoutRefresh(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required for SSE-to-DOM integration")
	}
	b := NewBridge()
	s, err := NewServer(b)
	if err != nil {
		t.Fatal("failed to create local test server")
	}
	server := httptest.NewServer(s.routes(http.NewServeMux()))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/sse?token="+s.connectToken(), nil)
	if err != nil {
		t.Fatal("failed to create synthetic SSE request")
	}
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal("local SSE connection failed")
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	type frame struct {
		Event    string   `json:"event"`
		Snapshot Snapshot `json:"snapshot"`
	}
	var frames []frame
	capture := func() {
		event, snap, _ := readSSEFrame(t, reader)
		frames = append(frames, frame{event, snap})
	}
	capture()
	for _, answer := range []string{"synthetic-first", "synthetic-second"} {
		for _, event := range []string{"solution-stream-start", "solution-stream-thinking", "solution-stream-chunk", "solution"} {
			b.Publish(event, answer)
			capture()
		}
	}
	raw, err := json.Marshal(frames)
	if err != nil {
		t.Fatal("cannot encode synthetic frames")
	}
	path := filepath.Join(t.TempDir(), "frames.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal("cannot write synthetic frames")
	}
	output, err := exec.CommandContext(ctx, node, "web_dom_test.mjs", path).CombinedOutput()
	if err != nil {
		t.Fatalf("real SSE frames failed DOM regression: %s", output)
	}
	t.Log("Real Bridge → SSE → DOM: two answers updated without refresh")
}
