package answersummary

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxNumberingLabels = 64
	maxNumberingBytes  = 4 * 1024
)

func FormatNumberingContext(ctx NumberingContext) string {
	labels := recentUniqueLabels(ctx.PreviousLabels)
	suggested, suggestedOK := safeNumberingLabel(ctx.SuggestedNextLabel)
	if len(labels) == 0 && !suggestedOK {
		return ""
	}

	for {
		formatted := formatNumberingContext(labels, suggested, suggestedOK)
		if len([]byte(formatted)) <= maxNumberingBytes {
			return formatted
		}
		if len(labels) == 0 {
			return ""
		}
		labels = labels[1:]
	}
}

func ExtractLegacyLabels(markdown string) []string {
	lines := markdownLines(markdown)
	heading := lastSummaryHeading(lines)
	if heading < 0 {
		return nil
	}
	end := len(lines)
	for i := heading + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "#") {
			end = i
			break
		}
	}
	section := lines[heading+1 : end]
	first := 0
	for first < len(section) && strings.TrimSpace(section[first]) == "" {
		first++
	}
	if first >= len(section) {
		return nil
	}

	if header, err := parseTableRow(section[first]); err == nil && len(header) >= 1 && strings.TrimSpace(header[0]) == "题号" && first+1 < len(section) {
		if separator, sepErr := parseTableRow(section[first+1]); sepErr == nil && len(separator) == len(header) {
			labels := make([]string, 0, len(section)-first-2)
			for _, line := range section[first+2:] {
				if strings.TrimSpace(line) == "" {
					continue
				}
				cells, rowErr := parseTableRow(line)
				if rowErr != nil || len(cells) != len(header) {
					continue
				}
				labels = append(labels, cells[0])
			}
			return recentUniqueLabels(labels)
		}
	}

	raw := strings.Join(section, "\n")
	chunks := strings.FieldsFunc(raw, func(r rune) bool { return r == '；' || r == ';' })
	labels := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunkLabels := make([]string, 0, 1)
		validChunk := true
		for _, line := range strings.Split(chunk, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			label, ok := legacyColonLabel(line)
			if !ok {
				validChunk = false
				break
			}
			chunkLabels = append(chunkLabels, label)
		}
		if validChunk {
			labels = append(labels, chunkLabels...)
		}
	}
	return recentUniqueLabels(labels)
}

func formatNumberingContext(labels []string, suggested string, includeSuggested bool) string {
	var b strings.Builder
	b.WriteString("【当前题库编号参考】\n")
	b.WriteString("仅当当前截图确实没有题号时参考；截图中的显式题号或章节号始终优先。\n")
	if len(labels) > 0 {
		b.WriteString("最近已用题号（由早到晚）：[")
		for i, label := range labels {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.QuoteToGraphic(label))
		}
		b.WriteString("]\n")
	}
	if includeSuggested {
		b.WriteString("安全的下一题号建议：")
		b.WriteString(strconv.QuoteToGraphic(suggested))
		b.WriteByte('\n')
	}
	b.WriteString("若编号规则不同或无法确定，请按截图实际内容显示，或标记“待核对”，不得强行续号。")
	return b.String()
}

func recentUniqueLabels(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	capacity := len(input)
	if capacity > maxNumberingLabels {
		capacity = maxNumberingLabels
	}
	reversed := make([]string, 0, capacity)
	for i := len(input) - 1; i >= 0 && len(reversed) < maxNumberingLabels; i-- {
		label, ok := safeNumberingLabel(input[i])
		if !ok {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		reversed = append(reversed, label)
	}
	labels := make([]string, len(reversed))
	for i := range reversed {
		labels[len(reversed)-1-i] = reversed[i]
	}
	return labels
}

func safeNumberingLabel(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "待核对") || utf8.RuneCountInString(value) > maxLabelRunes || hasControl(value) {
		return "", false
	}
	if strings.Contains(value, "答案") || strings.ContainsRune(value, '|') {
		return "", false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !strings.ContainsRune(" -_./()（）[]【】、", r) {
			return "", false
		}
	}
	return value, true
}

func legacyColonLabel(line string) (string, bool) {
	line = strings.TrimSpace(line)
	line = strings.TrimSpace(strings.TrimLeft(line, "-*+"))
	index := strings.IndexAny(line, "：:")
	if index <= 0 {
		return "", false
	}
	return safeNumberingLabel(line[:index])
}
