package mobile

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

const DefaultPort = 38511

var ErrPortUnavailable = errors.New("mobile port unavailable")

// Server 提供局域网 HTTP 页面与 SSE 快照服务。
type Server struct {
	bridge *Bridge

	mu             sync.Mutex
	httpSrv        *http.Server
	listener       net.Listener
	port           int
	url            string
	token          string
	lanIP          string
	started        bool
	stopping       bool
	authGeneration uint64
	nextLeaseID    uint64
	leases         map[uint64]sseLease
	handlerWG      sync.WaitGroup
}

type sseLease struct {
	generation uint64
	cancel     context.CancelFunc
}

// NewServer 使用调用方提供的 32 位十六进制 token 创建服务。
func NewServer(b *Bridge, token string) (*Server, error) {
	if b == nil {
		return nil, fmt.Errorf("bridge 不能为 nil")
	}
	if !validToken(token) {
		return nil, fmt.Errorf("mobile token 格式无效")
	}
	ip := resolveLANIP()
	if ip == "" {
		ip = "127.0.0.1"
		logPrintf("警告: 未解析到局域网 IP，回退 %s", ip)
	}
	return &Server{
		bridge:         b,
		port:           DefaultPort,
		token:          token,
		lanIP:          ip,
		authGeneration: 1,
		leases:         make(map[uint64]sseLease),
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

// URL 返回带 token 的接入 URL；调用方不得记录它。
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

func (s *Server) SafeURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.safeURLLocked()
}

func (s *Server) safeURLLocked() string {
	if s.token == "" || s.url == "" {
		return ""
	}
	return strings.Replace(s.url, s.token, "SET", 1)
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.stopping {
		return errors.New("mobile server 正在停止")
	}

	srv := &http.Server{
		Handler:           s.routes(http.NewServeMux()),
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
	logPrintf("移动端 SSE 服务已启动: %s", s.safeURLLocked())
	return nil
}

// Stop 只在锁内切换状态；取消、Shutdown 和 Wait 均在锁外执行。
func (s *Server) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	srv := s.httpSrv
	listener := s.listener
	s.httpSrv = nil
	s.listener = nil
	s.url = ""
	s.started = false
	s.stopping = true
	s.authGeneration++
	cancels := s.takeLeaseCancelsLocked(s.authGeneration)
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
	}
	if listener != nil {
		_ = listener.Close()
	}
	s.handlerWG.Wait()

	s.mu.Lock()
	s.stopping = false
	s.mu.Unlock()
	logPrintf("SSE 服务已停止，端口释放")
}

func (s *Server) connectToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

// RotateToken 无 I/O 地切换认证代次，并撤销旧代次 SSE lease。
func (s *Server) RotateToken(newToken string) {
	s.mu.Lock()
	s.token = newToken
	s.authGeneration++
	if s.started {
		s.url = fmt.Sprintf("http://%s:%d/?token=%s", s.lanIP, s.port, s.token)
	}
	cancels := s.takeLeaseCancelsLocked(s.authGeneration)
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func (s *Server) takeLeaseCancelsLocked(beforeGeneration uint64) []context.CancelFunc {
	cancels := make([]context.CancelFunc, 0, len(s.leases))
	for id, lease := range s.leases {
		if lease.generation >= beforeGeneration {
			continue
		}
		delete(s.leases, id)
		cancels = append(cancels, lease.cancel)
	}
	return cancels
}

func (s *Server) routes(mux *http.ServeMux) http.Handler {
	mux.HandleFunc("/", s.handleRoute)
	return mux
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	switch {
	case r.URL.Path == "/" && r.Method == http.MethodGet:
		if !s.authToken(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
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

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(webIndex())
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	leaseCtx, leaseID, ok := s.acquireSSELease(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer s.releaseSSELease(leaseID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "event-stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub, initial := s.bridge.Subscribe()
	defer s.bridge.Unsubscribe(sub)
	select {
	case <-leaseCtx.Done():
		return
	default:
	}
	initial.Control = ""
	s.writeEvent(w, flusher, "snapshot", initial)

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case snap, open := <-sub:
			if !open {
				return
			}
			select {
			case <-leaseCtx.Done():
				return
			default:
			}
			s.writeEvent(w, flusher, "message", snap)
		case <-heartbeat.C:
			select {
			case <-leaseCtx.Done():
				return
			default:
			}
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-leaseCtx.Done():
			return
		}
	}
}

// 鉴权与 lease 登记在同一临界区内，封闭 Token 轮换竞态窗口。
func (s *Server) acquireSSELease(r *http.Request) (context.Context, uint64, bool) {
	got := r.URL.Query().Get("token")
	s.mu.Lock()
	defer s.mu.Unlock()
	equal := constantTimeTokenEqual(got, s.token)
	if s.stopping || got == "" || s.token == "" || !equal {
		return nil, 0, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	s.nextLeaseID++
	id := s.nextLeaseID
	s.leases[id] = sseLease{generation: s.authGeneration, cancel: cancel}
	s.handlerWG.Add(1)
	return ctx, id, true
}

func (s *Server) releaseSSELease(id uint64) {
	s.mu.Lock()
	lease, exists := s.leases[id]
	if exists {
		delete(s.leases, id)
	}
	s.mu.Unlock()
	if exists {
		lease.cancel()
	}
	s.handlerWG.Done()
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

func (s *Server) authToken(r *http.Request) bool {
	got := r.URL.Query().Get("token")
	s.mu.Lock()
	defer s.mu.Unlock()
	equal := constantTimeTokenEqual(got, s.token)
	return !s.stopping && got != "" && s.token != "" && equal
}

func constantTimeTokenEqual(left, right string) bool {
	leftDigest := sha256.Sum256([]byte(left))
	rightDigest := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftDigest[:], rightDigest[:]) == 1
}

func validToken(token string) bool {
	if len(token) != 32 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

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
