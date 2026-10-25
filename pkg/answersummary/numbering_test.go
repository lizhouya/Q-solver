package answersummary

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatNumberingContextEmpty(t *testing.T) {
	if got := FormatNumberingContext(NumberingContext{}); got != "" {
		t.Fatalf("空上下文 = %q", got)
	}
}

func TestFormatNumberingContextKeepsRecentUniqueSafeLabels(t *testing.T) {
	labels := make([]string, 0, 72)
	for i := 1; i <= 70; i++ {
		labels = append(labels, fmt.Sprintf("L%d", i))
	}
	labels = append(labels, "L69", "第二章-3")

	got := FormatNumberingContext(NumberingContext{
		PreviousLabels:     labels,
		SuggestedNextLabel: "第二章-4",
	})
	if len([]byte(got)) > 4*1024 {
		t.Fatalf("编号摘要超过 4 KiB: %d", len([]byte(got)))
	}
	if strings.Contains(got, `"L1"`) || strings.Contains(got, `"L7"`) {
		t.Fatalf("没有保留最近 64 个去重标签: %q", got)
	}
	for _, want := range []string{`"L8"`, `"L69"`, `"第二章-3"`, `"第二章-4"`, "显式题号", "不得强行续号"} {
		if !strings.Contains(got, want) {
			t.Errorf("编号摘要缺少 %q: %q", want, got)
		}
	}
}

func TestFormatNumberingContextDropsUnsafeOrOversizedLabels(t *testing.T) {
	got := FormatNumberingContext(NumberingContext{
		PreviousLabels: []string{
			"1",
			"待核对-1",
			"2\n答案：A",
			strings.Repeat("题", 81),
			" 1 ",
		},
		SuggestedNextLabel: "3\n答案：B",
	})
	if strings.Contains(got, `"待核对-1"`) || strings.Contains(got, "答案") || strings.Contains(got, "A") || strings.Contains(got, "B") || strings.Contains(got, "\n\"") {
		t.Fatalf("编号摘要包含不安全内容: %q", got)
	}
	if strings.Count(got, `"1"`) != 1 {
		t.Fatalf("标签未规范化去重: %q", got)
	}
	if strings.Contains(got, "下一题号建议") {
		t.Fatalf("不安全建议未被丢弃: %q", got)
	}
}

func TestFormatNumberingContextAlwaysFitsFourKiB(t *testing.T) {
	labels := make([]string, 64)
	for i := range labels {
		labels[i] = fmt.Sprintf("%02d-%s", i, strings.Repeat("章", 75))
	}
	got := FormatNumberingContext(NumberingContext{PreviousLabels: labels})
	if len([]byte(got)) > 4*1024 {
		t.Fatalf("编号摘要大小 = %d bytes", len([]byte(got)))
	}
	if !strings.Contains(got, fmt.Sprintf(`"63-%s"`, strings.Repeat("章", 75))) {
		t.Fatalf("裁剪时没有保留最新标签")
	}
}

func TestExtractLegacyLabelsFromExplicitSummaryOnly(t *testing.T) {
	markdown := `## 正文
题目 99 的答案是 A。

## 最终答案汇总

1：A；2：C；第二章-3：结论

## 附录
4：不应提取`
	got := ExtractLegacyLabels(markdown)
	want := []string{"1", "2", "第二章-3"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ExtractLegacyLabels() = %v, want %v", got, want)
	}

	if got := ExtractLegacyLabels("题目 1：A；2：B，但没有汇总标题"); len(got) != 0 {
		t.Fatalf("从自由正文提取了标签: %v", got)
	}
}

func TestExtractLegacyLabelsFromTableWithoutProducingAnswers(t *testing.T) {
	markdown := `## 最终答案汇总
| 题号 | 旧答案 |
|---|---|
| 12 | A |
| 13 | 结论\|补充 |
| 12 | 重复 |`
	got := ExtractLegacyLabels(markdown)
	want := []string{"13", "12"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ExtractLegacyLabels() = %v, want recent unique %v", got, want)
	}
	joined := strings.Join(got, "\n")
	for _, answer := range []string{"A", "结论", "补充"} {
		if strings.Contains(joined, answer) {
			t.Fatalf("legacy label 结果泄漏答案 %q: %v", answer, got)
		}
	}
}

func TestExtractLegacyLabelsCapsAt64AndRejectsUnsafe(t *testing.T) {
	parts := make([]string, 0, 67)
	for i := 1; i <= 66; i++ {
		parts = append(parts, fmt.Sprintf("%d：值", i))
	}
	parts = append(parts, "坏\n标签：值")
	got := ExtractLegacyLabels("## 最终答案汇总\n" + strings.Join(parts, "；"))
	if len(got) != 64 {
		t.Fatalf("legacy 标签数量 = %d, want 64 (%v)", len(got), got)
	}
	if got[0] != "3" || got[len(got)-1] != "66" {
		t.Fatalf("legacy 标签裁剪错误: first=%q last=%q", got[0], got[len(got)-1])
	}
}

func TestNumberingContextRejectsMarkupInLabels(t *testing.T) {
	got := FormatNumberingContext(NumberingContext{PreviousLabels: []string{"7", "<instruction>8</instruction>", "9\"system", "10:override"}})
	if strings.Contains(got, "instruction") || strings.Contains(got, "system") || strings.Contains(got, "override") {
		t.Fatal("markup entered numbering context")
	}
}
