package answersummary

import (
	"strings"
	"testing"
)

func TestParseNormalizesAllKindsAndBuildsCopyText(t *testing.T) {
	markdown := `## 逐题解析
正文中的 A、B、C 不是汇总。

## 最终答案汇总
| 题号 | 题型 | 可填写答案 |
|---|---|---|
| 1 | 单选 | b |
| 2 | 多选 | a、 c；A |
| 3 | 填空 | 第一空\|含竖线；第二空 |
| 4 | 判断 | 正确 |
| 5 | 简答 | 直接结论 |
| 6 | 计算 | 42 |
| 7 | 代码 | fmt.Println("ok") |
| 8 | 其他 | 补充 |`

	got, err := Parse(markdown)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	wantKinds := []string{
		KindSingleChoice,
		KindMultipleChoice,
		KindFill,
		KindJudgment,
		KindShortAnswer,
		KindCalculation,
		KindCode,
		KindOther,
	}
	wantValues := []string{"B", "AC", "第一空|含竖线；第二空", "正确", "直接结论", "42", `fmt.Println("ok")`, "补充"}
	if len(got.Items) != len(wantKinds) {
		t.Fatalf("Items 数量 = %d, want %d", len(got.Items), len(wantKinds))
	}
	for i, item := range got.Items {
		if item.Label != string(rune('1'+i)) || item.Kind != wantKinds[i] || item.Value != wantValues[i] || item.Status != StatusReady {
			t.Errorf("Items[%d] = %+v", i, item)
		}
	}
	if want := strings.Join(wantValues, "\n"); got.CopyText != want {
		t.Fatalf("CopyText = %q, want %q", got.CopyText, want)
	}
}

func TestParseSingleQuestionProducesOneLineCopyText(t *testing.T) {
	got, err := Parse(`解答正文

## 最终答案汇总
| 题号 | 题型 | 可填写答案 |
|---|---|---|
| 12 | 单选 | c |`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.CopyText != "C" {
		t.Fatalf("单题解析结果不正确: %+v", got)
	}
}

func TestParseUsesLastExactTerminalSummary(t *testing.T) {
	got, err := Parse(`## 最终答案汇总
这不是固定表格。

## 正文
继续解答。

## 最终答案汇总
| 题号 | 题型 | 可填写答案 |
|---|---|---|
| 2 | 判断 | 错误 |`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Label != "2" || got.CopyText != "错误" {
		t.Fatalf("没有使用末尾最后一个精确汇总: %+v", got)
	}
}

func TestParseReviewDisablesWholeRoundCopy(t *testing.T) {
	tests := []struct {
		name     string
		row      string
		wantItem Item
	}{
		{
			name:     "label review",
			row:      "| 待核对 | 判断 | 正确 |",
			wantItem: Item{Label: "待核对", Kind: KindJudgment, Value: "正确", Status: StatusReview},
		},
		{
			name:     "value review",
			row:      "| 9 | 填空 | 待核对 |",
			wantItem: Item{Label: "9", Kind: KindFill, Value: "待核对", Status: StatusReview},
		},
		{
			name:     "label contains review marker",
			row:      "| 待核对-1 | 判断 | 正确 |",
			wantItem: Item{Label: "待核对-1", Kind: KindJudgment, Value: "正确", Status: StatusReview},
		},
		{
			name:     "value contains review marker",
			row:      "| 10 | 填空 | 待核对：截图不清 |",
			wantItem: Item{Label: "10", Kind: KindFill, Value: "待核对：截图不清", Status: StatusReview},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse("## 最终答案汇总\n| 题号 | 题型 | 可填写答案 |\n|---|---|---|\n" + tt.row)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(got.Items) != 1 || got.Items[0] != tt.wantItem || got.CopyText != "" {
				t.Fatalf("review 结果 = %+v", got)
			}
			if copyText, err := BuildCopyText(got.Items); err == nil || copyText != "" {
				t.Fatalf("BuildCopyText(review) = %q, %v", copyText, err)
			}
		})
	}
}

func TestParseRejectsInvalidContracts(t *testing.T) {
	validHead := "## 最终答案汇总\n| 题号 | 题型 | 可填写答案 |\n|---|---|---|\n"
	longLabel := strings.Repeat("题", 81)
	longValue := strings.Repeat("值", 2001)
	manyRows := make([]string, 51)
	for i := range manyRows {
		manyRows[i] = "| " + strings.Repeat("x", i+1) + " | 其他 | ok |"
	}
	largeRows := make([]string, 17)
	for i := range largeRows {
		largeRows[i] = "| " + strings.Repeat("x", i+1) + " | 其他 | " + strings.Repeat("a", 2000) + " |"
	}

	tests := []struct {
		name     string
		markdown string
	}{
		{name: "missing summary", markdown: "正文只有答案 A"},
		{name: "inexact heading", markdown: "### 最终答案汇总\n| 题号 | 题型 | 可填写答案 |\n|---|---|---|\n| 1 | 单选 | A |"},
		{name: "heading in fenced code", markdown: "```markdown\n" + validHead + "| 1 | 单选 | A |\n```"},
		{name: "missing column", markdown: "## 最终答案汇总\n| 题号 | 可填写答案 |\n|---|---|\n| 1 | A |"},
		{name: "wrong header", markdown: "## 最终答案汇总\n| 编号 | 题型 | 答案 |\n|---|---|---|\n| 1 | 单选 | A |"},
		{name: "wrong separator", markdown: "## 最终答案汇总\n| 题号 | 题型 | 可填写答案 |\n|:---|---|---|\n| 1 | 单选 | A |"},
		{name: "extra column", markdown: validHead + "| 1 | 单选 | A | 解析 |"},
		{name: "duplicate label", markdown: validHead + "| 1 | 单选 | A |\n| 1 | 单选 | B |"},
		{name: "unknown kind", markdown: validHead + "| 1 | 选择 | A |"},
		{name: "invalid single choice", markdown: validHead + "| 1 | 单选 | AB |"},
		{name: "invalid multiple choice", markdown: validHead + "| 1 | 多选 | A、2 |"},
		{name: "answer prefix", markdown: validHead + "| 1 | 简答 | 答案：结论 |"},
		{name: "markdown heading value", markdown: validHead + "| 1 | 其他 | ## 结论 |"},
		{name: "empty rows", markdown: strings.TrimSuffix(validHead, "\n")},
		{name: "too many rows", markdown: validHead + strings.Join(manyRows, "\n")},
		{name: "label too long", markdown: validHead + "| " + longLabel + " | 其他 | ok |"},
		{name: "value too long", markdown: validHead + "| 1 | 其他 | " + longValue + " |"},
		{name: "structured total too large", markdown: validHead + strings.Join(largeRows, "\n")},
		{name: "trailing content", markdown: validHead + "| 1 | 单选 | A |\n\n## 附注\n更多内容"},
		{name: "blank line within rows", markdown: validHead + "| 1 | 单选 | A |\n\n| 2 | 单选 | B |"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.markdown)
			if err == nil {
				t.Fatalf("Parse() = %+v, want error", got)
			}
			if len(got.Items) != 0 || got.CopyText != "" {
				t.Fatalf("失败时泄漏了部分结果: %+v", got)
			}
		})
	}
}

func TestBuildCopyTextValidatesCanonicalItems(t *testing.T) {
	ready := []Item{
		{Label: "12", Kind: KindSingleChoice, Value: "B", Status: StatusReady},
		{Label: "13", Kind: KindMultipleChoice, Value: "AC", Status: StatusReady},
		{Label: "14", Kind: KindFill, Value: "第一空；第二空", Status: StatusReady},
	}
	got, err := BuildCopyText(ready)
	if err != nil || got != "B\nAC\n第一空；第二空" {
		t.Fatalf("BuildCopyText() = %q, %v", got, err)
	}

	tests := []struct {
		name  string
		items []Item
	}{
		{name: "empty", items: nil},
		{name: "review", items: []Item{{Label: "1", Kind: KindOther, Value: "待核对", Status: StatusReview}}},
		{name: "unknown status", items: []Item{{Label: "1", Kind: KindOther, Value: "值", Status: "unknown"}}},
		{name: "noncanonical single", items: []Item{{Label: "1", Kind: KindSingleChoice, Value: "b", Status: StatusReady}}},
		{name: "noncanonical multiple", items: []Item{{Label: "1", Kind: KindMultipleChoice, Value: "A,C", Status: StatusReady}}},
		{name: "duplicate multiple option", items: []Item{{Label: "1", Kind: KindMultipleChoice, Value: "AAC", Status: StatusReady}}},
		{name: "duplicate label", items: []Item{{Label: "1", Kind: KindOther, Value: "甲", Status: StatusReady}, {Label: "1", Kind: KindOther, Value: "乙", Status: StatusReady}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyText, err := BuildCopyText(tt.items)
			if err == nil || copyText != "" {
				t.Fatalf("BuildCopyText() = %q, %v, want empty error", copyText, err)
			}
		})
	}
}
