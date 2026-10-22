package mobile

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed web
var webFS embed.FS

// DefaultPort 是移动端 SSE 服务默认监听端口（局域网）。
const DefaultPort = 38511

// ErrPortUnavailable 表示固定监听端口当前不可用，供 App 映射为稳定状态码。
var ErrPortUnavailable = errors.New("mobile port unavailable")

// Server 是一个本地 HTTP + SSE 服务，供同一局域网内的手机浏览器接收解题快照。
// 生命周期与 Wails App 绑定（App.Startup 启动 / App.OnShutdown 停止）。
// 约束二：Server 属于附加能力，任一启动失败仅上报，绝不影响 Q-Solver 主功能。
type Server struct {
	bridge *Bridge

	mu       sync.Mutex
	httpSrv  *http.Server
	listener net.Listener
	port     int    // 实际生效端口
	url      string // 接入 URL http://<lan-ip>:<port>/?token=xxx
	token    string // 随机 session token（内存态，不落盘不打印）
	lanIP    string
	started  bool
}

// NewServer 创建 SSE 服务实例，绑定到给定 Bridge。
// 若取局域网 IP 失败，回退 127.0.0.1（手机不可达但桌面不受影响，约束二）。
func NewServer(b *Bridge) (*Server, error) {
	if b == nil {
		return nil, fmt.Errorf("bridge 不能为 nil")
	}
	ip := resolveLANIP()
	if ip == "" {
		ip = "127.0.0.1"
		logPrintf("警告: 未解析到局域网 IP，回退 %s", ip)
	}
	token, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	return &Server{
		bridge: b,
		port:   DefaultPort,
		token:  token,
		lanIP:  ip,
	}, nil
}

func (s *Server) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// URL 返回带 token 的接入 URL（手机扫码或手输用）。token 属敏感值，调用方不得落日志。
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

// SafeURL 返回脱敏 URL（token 已替换为 SET），供日志与状态展示。
func (s *Server) SafeURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.safeURLLocked()
}

// safeURLLocked 在已持有 s.mu 时生成脱敏 URL。
func (s *Server) safeURLLocked() string {
	if s.token == "" || s.url == "" {
		return ""
	}
	return strings.Replace(s.url, s.token, "SET", 1)
}

// SetStartedLocked 仅用于单元测试注入。
func (s *Server) lambda() {}

// Start 启动 SSE 服务，监听局域网端口（0.0.0.0）。
// 失败返回 error 由调用方降级（约束二：不影响 Q-Solver 主功能）。
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}

	mux := http.NewServeMux()
	srv := &http.Server{
		Handler:           s.routes(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	requestedPort := s.port
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", requestedPort))
	if err != nil {
		return fmt.Errorf("%w: 端口 %d 监听失败: %v", ErrPortUnavailable, requestedPort, err)
	}
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		s.port = tcpAddr.Port
	}

	s.httpSrv = srv
	s.listener = listener
	s.url = fmt.Sprintf("http://%s:%d/?token=%s", s.lanIP, s.port, s.token)
	s.started = true

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logPrintf("SSE 服务异常退出: %v", err)
		}
	}()

	// 持锁期间已脱敏，避免 deadlock（SafeURL 会再取锁）
	logPrintf("移动端 SSE 服务已启动: %s", s.safeURLLocked())
	return nil
}

// Stop 关闭 SSE 服务、释放端口（App.OnShutdown 调用）。
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(ctx)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.started = false
	logPrintf("SSE 服务已停止，端口释放")
}

// tokPublic 暴露真实 token 供 URL 拼接（仅内部）。
func (s *Server) connectToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

// routes 组装路由表：
//
//	/          → 手机端单页（HTML）
//	/sse       → SSE 事件流
//	/health    → 健康检查
func (s *Server) routes(mux *http.ServeMux) http.Handler {
	mux.HandleFunc("/", s.handleRoute)
	return mux
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/" && r.Method == http.MethodGet:
		s.handleIndex(w, r)
	case r.URL.Path == "/sse" && r.Method == http.MethodGet:
		s.handleSSE(w, r)
	case r.URL.Path == "/health" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"mobile":"Q-Solver"}`))
	default:
		http.NotFound(w, r)
	}
}

// handleIndex 返回手机端单页（Step 4 实现真实页面；当前为占位页）。
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(webIndex())
}

// handleSSE 处理 EventSource 长连接：
//  1. 校验 token（?token=）
//  2. 立即推送当前快照（断线重连后手机端自动对齐最新状态）
//  3. 持续推送后续事件快照 + 心跳保活
//  4. 连接断开即注销，释放资源
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if !s.authToken(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "event-stream unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 原子完成订阅注册与当前快照读取，避免两步之间发布的终态丢失。
	sub, initial := s.bridge.Subscribe()
	defer s.bridge.Unsubscribe(sub)

	// 连接即拉取当前全量快照。首帧只同步 ControlSeq，绝不重放旧的一次性控制。
	initial.Control = ""
	s.writeEvent(w, flusher, "snapshot", initial)

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	reqCtx := r.Context()
	for {
		select {
		case snap := <-sub:
			s.writeEvent(w, flusher, "message", snap)
		case <-heartbeat.C:
			// SSE 保活注释行
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-reqCtx.Done():
			return
		}
	}
}

func (s *Server) writeEvent(w http.ResponseWriter, flusher http.Flusher, event string, snap Snapshot) {
	data, err := json.Marshal(&snap)
	if err != nil {
		return
	}
	if event != "" {
		_, _ = fmt.Fprintf(w, "event: %s\n", event)
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// authToken 校验 SSE 请求中的 token 与会话 token 一致。
func (s *Server) authToken(r *http.Request) bool {
	got := r.URL.Query().Get("token")
	if got == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return got == s.token
}

// logPrintf 集中打点（日志内服务使用 SafeURL，绝不打印真实 token）。
var logPrintf = func(format string, args ...interface{}) {
	log.Printf("[mobile] "+format, args...)
}

// resolveLANIP 枚举网卡，选择最可能是真实 Wi-Fi/以太网接口的 IPv4 作为接入地址。
// 策略：不是简单取第一个 RFC1918，而是对接口名做加权评分——
// 物理网络优先，WSL/Hyper-V/Mihomo/VPN/TUN 等虚拟接口强降权。
// 找不到理想候选时回退任意非环回 IPv4，仍失败返回 ""（上层回退 127.0.0.1，约束二）。
func resolveLANIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var candidates []ifaceCandidate
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 {
			continue // 只考虑启用接口
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			ipv4 := ipnet.IP.To4()
			if ipv4 == nil {
				continue
			}
			candidates = append(candidates, ifaceCandidate{Name: ifc.Name, IPv4: ipv4.String()})
		}
	}
	return chooseBestIPv4(candidates)
}

// ifaceCandidate 网卡地址候选（纯数据，便于单测注入不同网卡/地址组合）。
type ifaceCandidate struct {
	Name string // 接口名，如 "WLAN"、"vEthernet (WSL …)"、"Mihomo"
	IPv4 string // 接口的 IPv4 地址字符串
}

// chooseBestIPv4 从候选接口中选择最优的 IPv4（评分最高者）。
// 纯函数，不依赖真实网络；无候选或全为虚拟接口时返回 ""。
func chooseBestIPv4(cands []ifaceCandidate) string {
	best := ""
	bestScore := 0
	fallback := ""
	for _, c := range cands {
		ip := net.ParseIP(c.IPv4)
		if ip == nil {
			continue
		}
		// 排除：链路本地 APIPA（169.254/16，无实际路由意义）
		if isLinkLocalIPv4(ip) {
			continue
		}
		if isPrivateIPv4(ip.To4()) {
			// 只有私有网段才参与最高分竞争
			if fallback == "" {
				fallback = c.IPv4
			}
			score := scoreIfaceName(c.Name)
			if score > bestScore {
				bestScore = score
				best = c.IPv4
			}
		} else if fallback == "" {
			fallback = c.IPv4
		}
	}
	if best == "" {
		return fallback
	}
	return best
}

// scoreIfaceName 对接口名评分。
// 物理网络接口（WLAN/以太网/Ethernet）高权重；虚拟接口强负分。
// 用名 + flags 子串匹配，不硬编码具体机器信息。
func scoreIfaceName(name string) int {
	lower := strings.ToLower(name)

	// 虚拟/隧道/代理接口集合——明确排除或者强降权
	virtualMarkers := []string{
		"vEthernet", "vethernet",
		"wsl", "hyper-v", "hyperv", "hyper",
		"mihomo", "meta", "clash", "tun", "tap", "vpn",
		"tunnel", "docker", "virtual", "loopback", "bridge",
		"tailscale", "zerotier", "wireguard", "utun",
	}
	for _, m := range virtualMarkers {
		if strings.Contains(lower, m) {
			return -100
		}
	}

	// 物理无线网络接口（最高优先级）
	wifiMarkers := []string{"wlan", "wi-fi", "wifi", "wireless", "802.11"}
	for _, m := range wifiMarkers {
		if strings.Contains(lower, m) {
			return 100
		}
	}
	// 物理以太网接口
	ethMarkers := []string{"ethernet", "以太网", "eth"}
	for _, m := range ethMarkers {
		if strings.Contains(lower, m) {
			return 80
		}
	}
	return 10 // 未知类型：低分兜底
}

// isPrivateIPv4 是否为 RFC1918 私有网段（10/8、172.16/12、192.168/16）。
func isPrivateIPv4(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	if v4[0] == 10 {
		return true
	}
	if v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31 {
		return true
	}
	if v4[0] == 192 && v4[1] == 168 {
		return true
	}
	return false
}

// isLinkLocalIPv4 是否为链路本地地址（APIPA，169.254/16），无真实连接。
func isLinkLocalIPv4(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 169 && v4[1] == 254
}

// webIndex 返回手机端单页 HTML（从 embed 加载，绝对路径 web/index.html）。
func webIndex() []byte {
	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		return []byte(`<!doctype html><html><body><h3>页面资源缺失</h3></body></html>`)
	}
	return data
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}
