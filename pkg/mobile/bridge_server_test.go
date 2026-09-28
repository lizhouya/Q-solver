package mobile

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
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

// TestSSEHandshakeAndPublish 端到端验证 SSE：token 校验 + 连接即快照 + 实时推送。
func TestSSEHandshakeAndPublish(t *testing.T) {
	b := NewBridge()
	s, err := NewServer(b)
	if err != nil {
		t.Fatal(err)
	}

	// 找一个空闲端口供测试（避免 38511 冲突）
	freePort := getFreePort(t)
	s.port = freePort

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

	// 等待 SSE handler 完成订阅注册（避免 start 事件在订阅前发出而丢失）
	time.Sleep(100 * time.Millisecond)

	// 发送一帧事件，应很快被收到
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
	s.port = getFreePort(t)
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
//  HTTP 200 + text/html + body 含固定文本，且不依赖 token（HTML 页面本身无鉴权，token 只保护 /sse）。
func TestIndexHTML(t *testing.T) {
	b := NewBridge()
	s, _ := NewServer(b)
	s.port = getFreePort(t)
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
	for _, mark := range []string{"Q-Solver", "手机同步端", "EventSource"} {
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
	s.port = getFreePort(t)
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

// getFreePort 用 net.Listen 探测临时空闲端口。
func getFreePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

