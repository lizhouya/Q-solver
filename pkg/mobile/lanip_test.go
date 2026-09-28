package mobile

import (
	"testing"
)

// LAN IP 自动发现单测：不依赖真实网卡，直接注入候选接口数据。
// 覆盖 02-plan §LAN IP 评分策略 的核心决策分支。

func TestChooseBestIPv4_Case1_WLAN优先于WSL和Mihomo(t *testing.T) {
	cands := []ifaceCandidate{
		{Name: "vEthernet (WSL (Hyper-V firewall))", IPv4: "172.22.16.1"},
		{Name: "Mihomo", IPv4: "198.18.0.1"},
		{Name: "WLAN", IPv4: "192.168.1.102"},
	}
	got := chooseBestIPv4(cands)
	if got != "192.168.1.102" {
		t.Fatalf("期望选择 WLAN 192.168.1.102，得到 %s", got)
	}
}

func TestChooseBestLANIP_CaseEthernet优先于Docker(t *testing.T) {
	cands := []ifaceCandidate{
		{Name: "vEthernet (Docker)", IPv4: "172.18.0.1"},
		{Name: "Ethernet", IPv4: "192.168.0.5"},
	}
	got := chooseBestIPv4(cands)
	if got != "192.168.0.5" {
		t.Fatalf("期望选择 Ethernet 192.168.0.5，得到 %s", got)
	}
}

func TestChooseBestIPv4_Case3_单个合法接口(t *testing.T) {
	cands := []ifaceCandidate{
		{Name: "WLAN", IPv4: "10.0.0.3"},
	}
	if got := chooseBestIPv4(cands); got != "10.0.0.3" {
		t.Fatalf("唯一合法接口应被选中，得到 %s", got)
	}
}

func TestChooseBestIPv4_Case4_无合法LAN地址不panic(t *testing.T) {
	// 全虚拟接口，无真实物理网卡：
	// - 不 panic
	// - 不作为最佳返回（score>0 才入选，虚拟接口 -100 不会成为 best）
	// - 允许安全降级：返回 fallback 地址或空串（由调用方回退 127.0.0.1）
	cands := []ifaceCandidate{
		{Name: "Mihomo", IPv4: "198.18.0.1"},
		{Name: "vEthernet (WSL)", IPv4: "172.22.16.1"},
	}
	got := chooseBestIPv4(cands)
	switch got {
	case "": // unavailable 路径：合法
	case "172.22.16.1", "198.18.0.1": // fallback 安全降级路径：也合法
		// best 分支不会选择非 physically 分>0 的地址；此处仅为 fallback 兜底
	default:
		t.Fatalf("虚拟全接口时出现未知返回 %q", got)
	}
}

func TestChooseBestIPv4_空候选返回空(t *testing.T) {
	if got := chooseBestIPv4(nil); got != "" {
		t.Fatalf("空候选应返回空串，得到 %s", got)
	}
}

func TestChooseBestIPv4_排除链路本地APIPA(t *testing.T) {
	cands := []ifaceCandidate{
		{Name: "本地连接* 10", IPv4: "169.254.197.243"}, // APIPA 应被忽略
		{Name: "WLAN", IPv4: "192.168.1.102"},
	}
	if got := chooseBestIPv4(cands); got != "192.168.1.102" {
		t.Fatalf("APIPA 不应被选中，得到 %s", got)
	}
}

func TestChooseBestIPv4_WLAN高于Ethernet(t *testing.T) {
	cands := []ifaceCandidate{
		{Name: "Ethernet", IPv4: "192.168.0.5"},
		{Name: "WLAN", IPv4: "192.168.1.102"},
	}
	if got := chooseBestIPv4(cands); got != "192.168.1.102" {
		t.Fatalf("WLAN 应优先于 Ethernet，得到 %s", got)
	}
}

func TestScoreIfaceName(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"WLAN", 100},
		{"Wi-Fi", 100},
		{"wireless", 100},
		{"Ethernet", 80},
		{"以太网", 80},
		{"vEthernet (WSL (Hyper-V firewall))", -100},
		{"Mihomo", -100},
		{"Meta Tunnel", -100},
		{"Tailscale", -100}, // 含 tun
		{"DockerNAT", -100},
		{"eth0", 80},
	}
	for _, c := range cases {
		if got := scoreIfaceName(c.name); got != c.want {
			t.Errorf("scoreIfaceName(%q)=%d，期望 %d", c.name, got, c.want)
		}
	}
}