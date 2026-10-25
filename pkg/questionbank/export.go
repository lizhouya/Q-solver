package questionbank

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func statusLabel(status string) string {
	switch status {
	case Completed:
		return "完成（模型参考答案，非官方标准答案）"
	case Failed:
		return "请求失败，以下可能为部分内容"
	case Cancelled:
		return "已取消，以下可能为部分内容"
	case Interrupted:
		return "异常中断，以下可能为部分内容"
	case Empty:
		return "答案缺失"
	default:
		return "处理中"
	}
}

var externalURLPattern = regexp.MustCompile(`(?i)\b(?:(?:https?|ftp)://|www\.)`)

func modelMarkdown(value string) string {
	value = strings.NewReplacer(
		"\x00", "",
		"<", "&lt;",
		">", "&gt;",
		"](", `]\(`,
	).Replace(value)
	return externalURLPattern.ReplaceAllStringFunc(value, func(match string) string {
		if index := strings.Index(match, ":"); index >= 0 {
			return match[:index] + "：" + match[index+1:]
		}
		return strings.Replace(match, ".", "．", 1)
	})
}

func heading(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "#", "").Replace(modelMarkdown(value))
}

func rawAnswerMarkdown(value string) string {
	longest := 0
	current := 0
	for _, char := range value {
		if char == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	if longest < 3 {
		longest = 3
	} else {
		longest++
	}
	fence := strings.Repeat("`", longest)
	return fmt.Sprintf("%s%c%s%c%s%c%c", fence, 10, value, 10, fence, 10, 10)
}

func normalizedExportName(value string, session *Session) string {
	name := strings.TrimSpace(value)
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".md"), ".MD")
	if name == "" {
		name = strings.TrimSpace(session.Company)
	}
	if name == "" {
		name = "笔试题库"
	}
	return safeName(name)
}

func organizedFilename(name string, legacy bool) string {
	if legacy {
		return "题库.md"
	}
	if strings.EqualFold(name, "原始记录") {
		name += "_题库"
	}
	return name + ".md"
}

func (s *Store) Export(opts ExportOptions) (ExportResult, error) {
	s.mu.Lock()
	if err := s.ensureWriteAllowedLocked(); err != nil {
		s.mu.Unlock()
		return ExportResult{}, err
	}
	if s.current == nil {
		s.mu.Unlock()
		return ExportResult{}, fmt.Errorf("当前没有可导出的分段")
	}
	id := s.current.ID
	s.mu.Unlock()
	return s.ExportSession(id, opts)
}

func (s *Store) ExportSession(id string, opts ExportOptions) (ExportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureWriteAllowedLocked(); err != nil {
		return ExportResult{}, err
	}
	session, base, err := s.sessionSnapshotLocked(id)
	if err != nil {
		return ExportResult{}, err
	}
	switch session.State {
	case Ended, Exported:
	case Discarded:
		return ExportResult{}, fmt.Errorf("已清空的分段不参与导出")
	default:
		return ExportResult{}, fmt.Errorf("请先封存当前分段再导出")
	}
	for _, record := range session.Records {
		if record.Status == Running {
			return ExportResult{}, fmt.Errorf("分段仍有题目正在处理")
		}
	}

	exportRoot := s.root
	if strings.TrimSpace(opts.Directory) != "" {
		exportRoot, err = validateArchiveRoot(opts.Directory)
		if err != nil {
			return ExportResult{}, err
		}
	}
	legacyFilename := strings.TrimSpace(opts.Name) == ""
	name := normalizedExportName(opts.Name, session)
	stem := session.StartedAt.Format("2006-01-02_150405") + "_" + name
	if len([]rune(filepath.Join(exportRoot, stem))) > 220 {
		return ExportResult{}, fmt.Errorf("输出路径过长，请选择较短的题库目录或题库名称")
	}
	dest, err := nextExportDirectory(exportRoot, stem)
	if err != nil {
		return ExportResult{}, err
	}
	temp, err := os.MkdirTemp(exportRoot, ".exporting-")
	if err != nil {
		return ExportResult{}, err
	}
	defer os.RemoveAll(temp)
	if err = os.Mkdir(filepath.Join(temp, "assets"), 0700); err != nil {
		return ExportResult{}, err
	}

	raw, organized, questionCount, err := s.renderExport(session, base, name, opts, temp)
	if err != nil {
		return ExportResult{}, err
	}
	if err = os.WriteFile(filepath.Join(temp, "原始记录.md"), []byte(raw), 0600); err != nil {
		return ExportResult{}, err
	}
	organizedName := ""
	if !opts.RawOnly {
		organizedName = organizedFilename(name, legacyFilename)
		if err = os.WriteFile(filepath.Join(temp, organizedName), []byte(organized), 0600); err != nil {
			return ExportResult{}, err
		}
	}
	if err = os.Rename(temp, dest); err != nil {
		return ExportResult{}, err
	}

	result := ExportResult{
		Directory:   dest,
		RawMarkdown: filepath.Join(dest, "原始记录.md"),
		Records:     len(session.Records),
		Questions:   questionCount,
	}
	if organizedName != "" {
		result.Markdown = filepath.Join(dest, organizedName)
	}
	now := time.Now()
	appendExportSnapshot(session, now, dest)
	session.State = Exported
	session.ExportedAt = &now
	session.ExportDirectory = dest
	if s.current != nil && s.current.ID == id {
		s.current.State = session.State
		s.current.ExportedAt = session.ExportedAt
		s.current.ExportDirectory = session.ExportDirectory
		s.current.Exports = append([]ExportSnapshot(nil), session.Exports...)
		s.dirty = true
		if err = s.saveLocked(); err != nil {
			return result, fmt.Errorf("题库已导出，但分段状态保存失败: %w", err)
		}
	} else if err = s.saveSessionLocked(session, base, nil); err != nil {
		return result, fmt.Errorf("题库已导出，但分段状态保存失败: %w", err)
	}
	return result, nil
}

func (s *Store) sessionSnapshotLocked(id string) (*Session, string, error) {
	if err := s.ensureWriteAllowedLocked(); err != nil {
		return nil, "", err
	}
	if !validID(id) {
		return nil, "", fmt.Errorf("分段 ID 无效")
	}
	if s.current != nil && s.current.ID == id {
		if err := s.saveLocked(); err != nil {
			return nil, "", fmt.Errorf("原始记录尚未保存: %w", err)
		}
		return cloneSession(s.current), s.currentBase, nil
	}
	session, base, err := s.readSession(id)
	if err != nil {
		return nil, "", err
	}
	return session, base, nil
}

func nextExportDirectory(root, stem string) (string, error) {
	for index := 1; index <= 999; index++ {
		name := stem
		if index > 1 {
			name = fmt.Sprintf("%s_%02d", stem, index)
		}
		candidate := filepath.Join(root, name)
		_, err := os.Lstat(candidate)
		if os.IsNotExist(err) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("同一时间的导出版本过多，请稍后重试")
}

type organizedQuestion struct {
	question    Question
	recordIndex int
	number      int
}

func exportMetadata(session *Session, opts ExportOptions) (string, string, string) {
	company := strings.TrimSpace(opts.Company)
	if company == "" {
		company = strings.TrimSpace(session.Company)
	}
	issuer := strings.TrimSpace(opts.Issuer)
	if issuer == "" {
		issuer = strings.TrimSpace(session.Issuer)
	}
	kind := strings.TrimSpace(opts.Type)
	if kind == "" {
		kind = strings.TrimSpace(session.Type)
	}
	if kind == "" || kind == "综合" {
		kind = "综合笔试"
	}
	if company == "" {
		company = "未指定"
	}
	if issuer == "" {
		issuer = "未指定"
	}
	return company, issuer, kind
}

func (s *Store) renderExport(session *Session, base, name string, opts ExportOptions, temp string) (string, string, int, error) {
	company, issuer, kind := exportMetadata(session, opts)
	var raw strings.Builder
	fmt.Fprintf(&raw, "# %s：原始记录\n\n- 分段：%s\n- 出题方：%s\n- 单位：%s\n- 题型：%s\n- 开始时间：%s\n- 提交次数：%d\n\n> 原答案未经校验，不是官方标准答案。一次提交可能包含多道题，失败或取消记录也完整保留。\n\n", heading(name), session.ID, heading(issuer), heading(company), heading(kind), session.StartedAt.Format("2006-01-02 15:04:05 -07:00"), len(session.Records))

	categories := make([]string, 0)
	typesByCategory := make(map[string][]string)
	groups := make(map[string]map[string][]organizedQuestion)
	unorganized := make([]int, 0)
	questionCount := 0

	for recordIndex, record := range session.Records {
		fmt.Fprintf(&raw, "<a id=\"record-%d\"></a>\n\n## 提交 %d\n\n- 时间：%s\n- 状态：%s\n- 模型：%s\n- 场景：%s\n\n", recordIndex+1, recordIndex+1, record.CreatedAt.Format("2006-01-02 15:04:05"), statusLabel(record.Status), heading(record.Model), heading(record.Scene))
		for imageIndex, imageName := range record.Images {
			source, childErr := safeChild(filepath.Join(base, session.ID), imageName)
			if childErr != nil {
				return "", "", 0, childErr
			}
			data, readErr := readLimited(source, 20*1024*1024)
			if readErr != nil {
				return "", "", 0, readErr
			}
			destinationName := filepath.Base(imageName)
			if writeErr := os.WriteFile(filepath.Join(temp, "assets", destinationName), data, 0600); writeErr != nil {
				return "", "", 0, writeErr
			}
			fmt.Fprintf(&raw, "![提交 %d 截图 %d](assets/%s)\n\n", recordIndex+1, imageIndex+1, destinationName)
		}
		raw.WriteString("### 原答案\n\n")
		if record.Answer == "" {
			raw.WriteString("（无答案）\n\n")
		} else {
			raw.WriteString(rawAnswerMarkdown(record.Answer))
		}
		if record.OrganizeError != "" {
			fmt.Fprintf(&raw, "> 整理状态：%s\n\n", heading(record.OrganizeError))
		}

		if len(record.Questions) == 0 {
			unorganized = append(unorganized, recordIndex+1)
			continue
		}
		for _, question := range record.Questions {
			category := strings.TrimSpace(question.Category)
			if category == "" {
				category = "未分类"
			}
			questionType := strings.TrimSpace(question.Type)
			if questionType == "" {
				questionType = "未分类题型"
			}
			if groups[category] == nil {
				groups[category] = make(map[string][]organizedQuestion)
				categories = append(categories, category)
			}
			if _, exists := groups[category][questionType]; !exists {
				typesByCategory[category] = append(typesByCategory[category], questionType)
			}
			questionCount++
			groups[category][questionType] = append(groups[category][questionType], organizedQuestion{
				question: question, recordIndex: recordIndex + 1, number: questionCount,
			})
		}
	}

	var organized strings.Builder
	fmt.Fprintf(&organized, "# %s\n\n- 出题方：%s\n- 单位：%s\n- 题型：%s\n- 题目数量：%d\n- 整理时间：%s\n\n> 题干、分类、参考答案与解析由模型辅助提炼，均需人工核对。完整截图和未经改写的原答案见 [原始记录](原始记录.md)。\n\n## 试题\n\n", heading(name), heading(issuer), heading(company), heading(kind), questionCount, time.Now().Format("2006-01-02 15:04:05"))
	if len(categories) == 0 {
		organized.WriteString("暂无结构化整理结果。\n\n")
	}
	ordered := make([]organizedQuestion, 0, questionCount)
	for _, category := range categories {
		fmt.Fprintf(&organized, "### %s\n\n", heading(category))
		for _, questionType := range typesByCategory[category] {
			fmt.Fprintf(&organized, "#### %s\n\n", heading(questionType))
			for _, item := range groups[category][questionType] {
				ordered = append(ordered, item)
				question := item.question
				fmt.Fprintf(&organized, "##### 第 %d 题：%s\n\n%s\n\n", item.number, heading(question.Title), modelMarkdown(question.Stem))
				for _, option := range question.Options {
					fmt.Fprintf(&organized, "- %s\n", modelMarkdown(option))
				}
				if len(question.Options) > 0 {
					organized.WriteString("\n")
				}
			}
		}
	}

	organized.WriteString("## 参考答案与解析\n\n")
	for _, item := range ordered {
		question := item.question
		referenceAnswer := strings.TrimSpace(question.ReferenceAnswer)
		if referenceAnswer == "" {
			referenceAnswer = "待核对"
		}
		fmt.Fprintf(&organized, "### 第 %d 题：%s\n\n**参考答案：** %s\n\n", item.number, heading(question.Title), modelMarkdown(referenceAnswer))
		organized.WriteString("**知识点：**\n\n")
		if len(question.KnowledgePoints) == 0 {
			organized.WriteString("- 待核对\n")
		} else {
			for _, point := range question.KnowledgePoints {
				fmt.Fprintf(&organized, "- %s\n", modelMarkdown(point))
			}
		}
		organized.WriteString("\n**解析要点：**\n\n")
		if len(question.KeyPoints) == 0 {
			organized.WriteString("- 待核对\n")
		} else {
			for _, point := range question.KeyPoints {
				fmt.Fprintf(&organized, "- %s\n", modelMarkdown(point))
			}
		}
		fmt.Fprintf(&organized, "\n**来源：** [原始提交 %d](原始记录.md#record-%d)\n\n**核对说明：** %s\n\n", item.recordIndex, item.recordIndex, modelMarkdown(question.ReviewNote))
	}

	if len(unorganized) > 0 {
		organized.WriteString("## 尚未结构化的提交\n\n")
		for _, recordIndex := range unorganized {
			fmt.Fprintf(&organized, "- [提交 %d：截图与原答案](原始记录.md#record-%d)\n", recordIndex, recordIndex)
		}
		organized.WriteString("\n")
	}
	return raw.String(), organized.String(), questionCount, nil
}
