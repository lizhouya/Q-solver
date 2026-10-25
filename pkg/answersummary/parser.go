package answersummary

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxItems           = 50
	maxLabelRunes      = 80
	maxValueRunes      = 2000
	maxStructuredBytes = 32 * 1024
	summaryHeading     = "## 最终答案汇总"
)

var kindByDisplayName = map[string]string{
	"单选": KindSingleChoice,
	"多选": KindMultipleChoice,
	"填空": KindFill,
	"判断": KindJudgment,
	"简答": KindShortAnswer,
	"计算": KindCalculation,
	"代码": KindCode,
	"其他": KindOther,
}

var validKinds = map[string]struct{}{
	KindSingleChoice:   {},
	KindMultipleChoice: {},
	KindFill:           {},
	KindJudgment:       {},
	KindShortAnswer:    {},
	KindCalculation:    {},
	KindCode:           {},
	KindOther:          {},
}

func Parse(markdown string) (Result, error) {
	lines := markdownLines(markdown)
	heading := lastSummaryHeading(lines)
	if heading < 0 {
		return Result{}, fmt.Errorf("缺少末尾 %s", summaryHeading)
	}
	if heading+3 >= len(lines) {
		return Result{}, fmt.Errorf("答案汇总表不完整")
	}

	header, err := parseTableRow(lines[heading+1])
	if err != nil || !equalCells(header, []string{"题号", "题型", "可填写答案"}) {
		return Result{}, fmt.Errorf("答案汇总表头必须为固定三列")
	}
	separator, err := parseTableRow(lines[heading+2])
	if err != nil || !equalCells(separator, []string{"---", "---", "---"}) {
		return Result{}, fmt.Errorf("答案汇总分隔行必须为固定三列")
	}

	items := make([]Item, 0, len(lines)-heading-3)
	for i := heading + 3; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			return Result{}, fmt.Errorf("答案汇总表中不得包含空行")
		}
		cells, rowErr := parseTableRow(lines[i])
		if rowErr != nil || len(cells) != 3 {
			return Result{}, fmt.Errorf("答案汇总第 %d 行不是固定三列表格", i-heading-2)
		}
		item, rowErr := itemFromCells(cells)
		if rowErr != nil {
			return Result{}, fmt.Errorf("答案汇总第 %d 行无效: %w", i-heading-2, rowErr)
		}
		items = append(items, item)
	}

	if err := validateItems(items, false); err != nil {
		return Result{}, err
	}
	result := Result{Items: items}
	for _, item := range items {
		if item.Status == StatusReview {
			return result, nil
		}
	}
	result.CopyText, err = BuildCopyText(items)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func BuildCopyText(items []Item) (string, error) {
	if err := validateItems(items, true); err != nil {
		return "", err
	}
	values := make([]string, len(items))
	for i, item := range items {
		values[i] = item.Value
	}
	return strings.Join(values, "\n"), nil
}

func itemFromCells(cells []string) (Item, error) {
	label := strings.TrimSpace(cells[0])
	displayKind := strings.TrimSpace(cells[1])
	value := strings.TrimSpace(cells[2])
	kind, ok := kindByDisplayName[displayKind]
	if !ok {
		return Item{}, fmt.Errorf("未知题型 %q", displayKind)
	}

	status := StatusReady
	if strings.Contains(label, "待核对") || strings.Contains(value, "待核对") {
		status = StatusReview
	}
	if !strings.Contains(value, "待核对") {
		var err error
		switch kind {
		case KindSingleChoice:
			value, err = normalizeChoice(value, true)
		case KindMultipleChoice:
			value, err = normalizeChoice(value, false)
		default:
			err = validateDirectValue(value)
		}
		if err != nil {
			return Item{}, err
		}
	}
	return Item{Label: label, Kind: kind, Value: value, Status: status}, nil
}

func normalizeChoice(value string, single bool) (string, error) {
	var normalized strings.Builder
	seen := make(map[rune]struct{})
	for _, r := range strings.TrimSpace(value) {
		if unicode.IsSpace(r) || strings.ContainsRune(",，、;；/\\|+", r) {
			continue
		}
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("选项只能包含字母和常见分隔符")
		}
		if _, exists := seen[r]; exists {
			continue
		}
		seen[r] = struct{}{}
		normalized.WriteRune(r)
	}
	result := normalized.String()
	if result == "" {
		return "", fmt.Errorf("选项不能为空")
	}
	if single && len(result) != 1 {
		return "", fmt.Errorf("单选答案必须为一个选项字母")
	}
	return result, nil
}

func validateDirectValue(value string) error {
	if value == "" {
		return fmt.Errorf("可填写答案不能为空")
	}
	if strings.HasPrefix(value, "答案：") || strings.HasPrefix(value, "答案:") {
		return fmt.Errorf("可填写答案不得包含答案前缀")
	}
	if strings.HasPrefix(value, "#") {
		return fmt.Errorf("可填写答案不得包含 Markdown 标题")
	}
	return nil
}

func validateItems(items []Item, requireReady bool) error {
	if len(items) == 0 {
		return fmt.Errorf("答案汇总至少需要一项")
	}
	if len(items) > maxItems {
		return fmt.Errorf("答案汇总最多允许 %d 项", maxItems)
	}

	labels := make(map[string]struct{}, len(items))
	totalBytes := 0
	for i, item := range items {
		if item.Label == "" || utf8.RuneCountInString(item.Label) > maxLabelRunes || hasControl(item.Label) {
			return fmt.Errorf("第 %d 项题号为空、含控制字符或超过 %d 个字符", i+1, maxLabelRunes)
		}
		if strings.TrimSpace(item.Label) != item.Label {
			return fmt.Errorf("第 %d 项题号未规范化", i+1)
		}
		if _, exists := labels[item.Label]; exists {
			return fmt.Errorf("题号 %q 重复", item.Label)
		}
		labels[item.Label] = struct{}{}

		if _, ok := validKinds[item.Kind]; !ok {
			return fmt.Errorf("第 %d 项题型无效", i+1)
		}
		if item.Status != StatusReady && item.Status != StatusReview {
			return fmt.Errorf("第 %d 项状态无效", i+1)
		}
		if (strings.Contains(item.Label, "待核对") || strings.Contains(item.Value, "待核对")) && item.Status != StatusReview {
			return fmt.Errorf("第 %d 项待核对状态不一致", i+1)
		}
		if requireReady && item.Status != StatusReady {
			return fmt.Errorf("第 %d 项仍待核对", i+1)
		}
		if item.Value == "" || utf8.RuneCountInString(item.Value) > maxValueRunes || hasControl(item.Value) {
			return fmt.Errorf("第 %d 项答案为空、含控制字符或超过 %d 个字符", i+1, maxValueRunes)
		}
		if strings.TrimSpace(item.Value) != item.Value {
			return fmt.Errorf("第 %d 项答案未规范化", i+1)
		}
		if !strings.Contains(item.Value, "待核对") {
			switch item.Kind {
			case KindSingleChoice:
				normalized, err := normalizeChoice(item.Value, true)
				if err != nil || normalized != item.Value {
					return fmt.Errorf("第 %d 项单选答案不是规范值", i+1)
				}
			case KindMultipleChoice:
				normalized, err := normalizeChoice(item.Value, false)
				if err != nil || normalized != item.Value {
					return fmt.Errorf("第 %d 项多选答案不是规范值", i+1)
				}
			default:
				if err := validateDirectValue(item.Value); err != nil {
					return fmt.Errorf("第 %d 项答案无效: %w", i+1, err)
				}
			}
		}
		totalBytes += len(item.Label) + len(item.Kind) + len(item.Value) + len(item.Status)
		if totalBytes > maxStructuredBytes {
			return fmt.Errorf("结构化答案总量超过 32 KiB")
		}
	}
	return nil
}

func parseTableRow(line string) ([]string, error) {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '|' || line[len(line)-1] != '|' {
		return nil, fmt.Errorf("不是 Markdown 表格行")
	}
	inner := line[1 : len(line)-1]
	cells := make([]string, 0, 3)
	var cell strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) && inner[i+1] == '|' {
			cell.WriteByte('|')
			i++
			continue
		}
		if inner[i] == '|' {
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteByte(inner[i])
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells, nil
}

func markdownLines(markdown string) []string {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")
	lines := strings.Split(markdown, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func lastSummaryHeading(lines []string) int {
	last := -1
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			fence = "```"
			continue
		}
		if strings.HasPrefix(trimmed, "~~~") {
			fence = "~~~"
			continue
		}
		if line == summaryHeading {
			last = i
		}
	}
	return last
}

func equalCells(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func hasControl(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}
