package solution

import (
	"fmt"
	"testing"

	"Q-Solver/pkg/llm"
)

// countImages 返回一条消息中的图片 part 数量。
func countImages(m llm.Message) int {
	n := 0
	for _, p := range m.Parts {
		if p.Type == llm.ContentImage {
			n++
		}
	}
	return n
}

// imageProfile 返回一次请求中「消息角色 → 图片数」的画像，以及图片总数。
func imageProfile(ms []llm.Message) ([]string, int) {
	profile := make([]string, 0, len(ms))
	total := 0
	for _, m := range ms {
		n := countImages(m)
		total += n
		profile = append(profile, fmt.Sprintf("%s:%d", m.Role, n))
	}
	return profile, total
}

// =====================================================================
// T-VIS-01 测量：KeepContext=false 单轮多图。
// =====================================================================
func TestConversationImageComposition_KeepFalseSingleRound(t *testing.T) {
	fp := &fakeProvider{reply: "A1"}
	s := NewSolver(fp)

	solveOnce(s, false, []string{"b1", "b2"})

	profile, total := imageProfile(fp.received[0])
	t.Logf("KeepContext=false 单轮 2 图 → 画像=%v 图片总数=%d", profile, total)
}

// =====================================================================
// T-VIS-02 测量：KeepContext=true 连续 3 轮，每轮 2 图。
// 关注第 3 轮请求中的图片总数是否随轮次线性增长。
// =====================================================================
func TestConversationImageComposition_KeepTrueThreeRounds(t *testing.T) {
	fp := &fakeProvider{reply: "A"}
	s := NewSolver(fp)

	for round := 1; round <= 3; round++ {
		solveOnce(s, true, []string{fmt.Sprintf("r%d-a", round), fmt.Sprintf("r%d-b", round)})
	}

	if len(fp.received) != 3 {
		t.Fatalf("应收到 3 次请求，实际 %d", len(fp.received))
	}
	for i, ms := range fp.received {
		profile, total := imageProfile(ms)
		t.Logf("KeepContext=true 第 %d 轮 → 画像=%v 图片总数=%d 消息数=%d", i+1, profile, total, len(ms))
	}
}

// =====================================================================
// T-VIS-03 测量：KeepContext=true 超过 MaxConversationRounds 轮，
// 观察图片总数上限与当前轮截图是否仍完整存在。
// =====================================================================
func TestConversationImageComposition_KeepTrueBeyondMaxRounds(t *testing.T) {
	fp := &fakeProvider{reply: "A"}
	s := NewSolver(fp)

	rounds := MaxConversationRounds + 3
	for round := 1; round <= rounds; round++ {
		solveOnce(s, true, []string{fmt.Sprintf("r%d-a", round), fmt.Sprintf("r%d-b", round)})
	}

	last := fp.received[len(fp.received)-1]
	profile, total := imageProfile(last)
	t.Logf("KeepContext=true 第 %d 轮（Max=%d）→ 画像=%v 图片总数=%d 消息数=%d",
		rounds, MaxConversationRounds, profile, total, len(last))

	// 当前轮截图必须完整存在于最后一条 user 消息中。
	current := last[len(last)-1]
	if got := countImages(current); got != 2 {
		t.Fatalf("当前轮应含 2 张截图，实际 %d", got)
	}
}

// =====================================================================
// T-VIS-04 KeepContext=true 时，请求携带的历史轮数不得超过
// MaxConversationRounds（含当前轮）。当前轮截图必须完整存在。
// =====================================================================
func TestKeepContextRequestDoesNotExceedHistoryRoundBudget(t *testing.T) {
	const shotsPerRound = 2
	fp := &fakeProvider{reply: "A"}
	s := NewSolver(fp)

	for round := 1; round <= MaxConversationRounds+3; round++ {
		solveOnce(s, true, []string{
			fmt.Sprintf("r%d-a", round), fmt.Sprintf("r%d-b", round),
		})
	}

	last := fp.received[len(fp.received)-1]
	_, total := imageProfile(last)

	// 当前轮截图必须完整。
	current := last[len(last)-1]
	if got := countImages(current); got != shotsPerRound {
		t.Fatalf("当前轮应含 %d 张截图，实际 %d", shotsPerRound, got)
	}

	// 历史轮数（不含当前轮）不得超过 MaxConversationRounds-1。
	historyImages := total - shotsPerRound
	maxHistoryImages := (MaxConversationRounds - 1) * shotsPerRound
	if historyImages > maxHistoryImages {
		t.Fatalf("历史图片应不超过 %d 张（%d 轮），实际 %d 张；请求总图片 %d 张",
			maxHistoryImages, MaxConversationRounds-1, historyImages, total)
	}

	t.Logf("最后一轮请求：图片总数=%d，其中历史 %d 张、当前轮 %d 张",
		total, historyImages, shotsPerRound)
}
