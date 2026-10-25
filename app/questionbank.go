package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"Q-Solver/pkg/common"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/questionbank"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type questionBankService struct {
	mu          sync.Mutex
	store       *questionbank.Store
	preferences QuestionBankPreferences
	legacyRoots []string
	workers     sync.WaitGroup
	captures    int
	organizing  bool
	operation   string
	closing     bool
	cancel      context.CancelFunc
	progress    questionbank.Progress
	lastExport  questionbank.ExportResult
	message     string
}

type QuestionBankPreferences struct {
	Version      int    `json:"version"`
	ArchiveDir   string `json:"archiveDir"`
	Company      string `json:"company"`
	AutoOrganize bool   `json:"autoOrganize"`
}

type QuestionBankExportDraft struct {
	SessionID          string                                 `json:"sessionId"`
	Company            string                                 `json:"company"`
	Issuer             string                                 `json:"issuer"`
	Type               string                                 `json:"type"`
	Name               string                                 `json:"name"`
	NameCustomized     bool                                   `json:"nameCustomized"`
	Directory          string                                 `json:"directory"`
	AutoOrganize       bool                                   `json:"autoOrganize"`
	NumberedPrefix     string                                 `json:"numberedPrefix"`
	Count              int                                    `json:"count"`
	StartedAt          string                                 `json:"startedAt"`
	MetadataSuggestion *questionbank.SourceMetadataSuggestion `json:"metadataSuggestion,omitempty"`
	SuggestionConflict bool                                   `json:"suggestionConflict,omitempty"`
	OrganizeReady      bool                                   `json:"organizeReady"`
	PendingOrganize    int                                    `json:"pendingOrganize"`
	FailedOrganize     int                                    `json:"failedOrganize"`
}

type QuestionBankExportRequest struct {
	SessionID      string `json:"sessionId"`
	Company        string `json:"company"`
	Issuer         string `json:"issuer"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	NameCustomized bool   `json:"nameCustomized"`
	Directory      string `json:"directory"`
	AutoOrganize   bool   `json:"autoOrganize"`
}

type questionBankPreferencesFile struct {
	Version      int      `json:"version"`
	ArchiveDir   string   `json:"archiveDir,omitempty"`
	Company      string   `json:"company,omitempty"`
	AutoOrganize bool     `json:"autoOrganize,omitempty"`
	Root         string   `json:"root,omitempty"`
	LegacyRoots  []string `json:"legacyRoots,omitempty"`
}

type QuestionBankStatus struct {
	Root        string                    `json:"root"`
	Preferences QuestionBankPreferences   `json:"preferences"`
	Session     *questionbank.SessionInfo `json:"session"`
	Unsaved     bool                      `json:"unsaved"`
	Busy        bool                      `json:"busy"`
	Organizing  bool                      `json:"organizing"`
	Operation   string                    `json:"operation"`
	Progress    questionbank.Progress     `json:"progress"`
	LastExport  questionbank.ExportResult `json:"lastExport"`
	Message     string                    `json:"message"`
}

func bankPreferencesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, common.AppName, "questionbank-settings.json"), nil
}

func normalizeLegacyPreferences(file questionBankPreferencesFile) questionBankPreferencesFile {
	if file.ArchiveDir != "" {
		file.Version = 2
		return file
	}
	root := strings.TrimSpace(file.Root)
	if root == "" {
		return file
	}
	clean := filepath.Clean(root)
	if strings.EqualFold(filepath.Base(clean), "题库") && strings.EqualFold(filepath.Base(filepath.Dir(clean)), "求职笔试") {
		file.ArchiveDir = clean
	} else {
		file.ArchiveDir = filepath.Join(clean, "求职笔试", "题库")
	}
	file.LegacyRoots = append(file.LegacyRoots, filepath.Join(clean, "求职笔试", "会话记录"))
	file.Version = 2
	return file
}

func preferencesForUI(file questionBankPreferencesFile) QuestionBankPreferences {
	return QuestionBankPreferences{
		Version:      2,
		ArchiveDir:   file.ArchiveDir,
		Company:      "",
		AutoOrganize: file.AutoOrganize,
	}
}

func (a *App) loadQuestionBank() {
	b := a.questionBank
	if b == nil {
		return
	}
	path, err := bankPreferencesPath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		b.preferences = QuestionBankPreferences{Version: 2}
		return
	}
	var file questionBankPreferencesFile
	if err != nil || json.Unmarshal(data, &file) != nil {
		b.message = "题库目录偏好无法读取，请重新选择题库目录"
		return
	}
	file = normalizeLegacyPreferences(file)
	b.preferences = preferencesForUI(file)
	b.legacyRoots = append([]string(nil), file.LegacyRoots...)
	if file.ArchiveDir == "" {
		return
	}
	store, err := questionbank.NewStoreWithLegacy(file.ArchiveDir, file.LegacyRoots)
	if err != nil {
		b.message = "原题库目录不可用，请重新选择；原始记录未被删除"
		return
	}
	if err = store.SetDefaultCompany(""); err != nil {
		b.message = err.Error()
		return
	}
	b.store = store
	sessions, err := store.List()
	if err != nil {
		b.message = err.Error()
		return
	}
	for _, session := range sessions {
		if session.State == questionbank.Active || session.State == questionbank.Paused {
			err = store.Open(session.ID)
			break
		}
	}
	if err != nil {
		b.message = err.Error()
	}
}

func (a *App) GetQuestionBankPreferences() QuestionBankPreferences {
	b := a.questionBank
	if b == nil {
		return QuestionBankPreferences{Version: 2}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.preferences
}

func (a *App) GetQuestionBankStatus() QuestionBankStatus {
	b := a.questionBank
	if b == nil {
		return QuestionBankStatus{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	status := QuestionBankStatus{
		Preferences: b.preferences,
		Busy:        b.captures > 0,
		Organizing:  b.organizing,
		Operation:   b.operation,
		Progress:    b.progress,
		LastExport:  b.lastExport,
		Message:     b.message,
	}
	if b.store != nil {
		status.Root = b.store.Root()
		session, dirty := b.store.Snapshot()
		status.Unsaved = dirty
		if session != nil {
			status.Session = &questionbank.SessionInfo{
				ID:              session.ID,
				Name:            session.Name,
				Company:         session.Company,
				Issuer:          session.Issuer,
				Type:            session.Type,
				NameCustomized:  session.NameCustomized,
				State:           session.State,
				Count:           len(session.Records),
				StartedAt:       session.StartedAt.Format(time.RFC3339Nano),
				ExportDirectory: session.ExportDirectory,
			}
		}
	}
	return status
}

func (a *App) notifyQuestionBank() {
	if a.questionBank == nil {
		return
	}
	a.questionBank.mu.Lock()
	closing := a.questionBank.closing
	a.questionBank.mu.Unlock()
	if !closing {
		a.EmitEvent("question-bank-updated")
	}
}

func (a *App) bankAvailableLocked() error {
	b := a.questionBank
	if b == nil || b.store == nil {
		return fmt.Errorf("请先在设置中选择题库目录")
	}
	if b.closing {
		return fmt.Errorf("应用正在关闭")
	}
	if b.organizing || b.captures > 0 {
		return fmt.Errorf("请等待当前题目或整理任务完成")
	}
	if a.taskManager != nil && a.taskManager.HasRunningTask() {
		return fmt.Errorf("请等待解题任务完成")
	}
	return nil
}

func (a *App) SelectQuestionBankOutput() (string, error) {
	b := a.questionBank
	if b == nil {
		return "", fmt.Errorf("题库服务不可用")
	}
	b.mu.Lock()
	if b.organizing || b.captures > 0 {
		b.mu.Unlock()
		return "", fmt.Errorf("正在处理，请稍后更换目录")
	}
	current := b.preferences.ArchiveDir
	b.mu.Unlock()

	defaultDirectory := current
	if defaultDirectory == "" {
		defaultDirectory = suggestQuestionBankOutput()
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择最终题库目录",
		DefaultDirectory: defaultDirectory,
	})
	if err != nil || path == "" {
		return "", err
	}
	if err = ensureWritableDirectory(path); err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func (a *App) SelectQuestionBankExportDirectory(current string) (string, error) {
	b := a.questionBank
	if b == nil {
		return "", fmt.Errorf("题库服务不可用")
	}
	b.mu.Lock()
	if b.organizing || b.captures > 0 {
		b.mu.Unlock()
		return "", fmt.Errorf("正在处理，请稍后选择导出目录")
	}
	defaultDirectory := strings.TrimSpace(current)
	if defaultDirectory == "" {
		defaultDirectory = b.preferences.ArchiveDir
	}
	b.mu.Unlock()
	if defaultDirectory == "" {
		defaultDirectory = suggestQuestionBankOutput()
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择本次题库导出目录",
		DefaultDirectory: defaultDirectory,
	})
	if err != nil || path == "" {
		return "", err
	}
	return questionbank.ValidateExportDirectory(path)
}

func (a *App) UpdateQuestionBankPreferences(preferences QuestionBankPreferences) error {
	b := a.questionBank
	if b == nil {
		return fmt.Errorf("题库服务不可用")
	}
	preferences.Version = 2
	preferences.ArchiveDir = strings.TrimSpace(preferences.ArchiveDir)
	preferences.Company = ""

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closing {
		return fmt.Errorf("应用正在关闭")
	}
	if b.organizing || b.captures > 0 || (a.taskManager != nil && a.taskManager.HasRunningTask()) {
		return fmt.Errorf("正在处理，请稍后保存题库设置")
	}

	var store *questionbank.Store
	legacyRoots := append([]string(nil), b.legacyRoots...)
	rootChanged := b.store != nil && (preferences.ArchiveDir == "" || filepath.Clean(b.store.Root()) != filepath.Clean(preferences.ArchiveDir))
	if rootChanged {
		session, dirty := b.store.Snapshot()
		if dirty || (session != nil && (session.State == questionbank.Active || session.State == questionbank.Paused)) {
			return fmt.Errorf("请先分段或清空当前段，再更换题库目录")
		}
	}
	if preferences.ArchiveDir != "" {
		if err := ensureWritableDirectory(preferences.ArchiveDir); err != nil {
			return err
		}
		var err error
		store, err = questionbank.NewStoreWithLegacy(preferences.ArchiveDir, legacyRoots)
		if err != nil {
			return err
		}
		if err = store.SetDefaultCompany(""); err != nil {
			return err
		}
	}
	if err := saveQuestionBankPreferences(questionBankPreferencesFile{
		Version:      2,
		ArchiveDir:   preferences.ArchiveDir,
		Company:      "",
		AutoOrganize: preferences.AutoOrganize,
		LegacyRoots:  legacyRoots,
	}); err != nil {
		return err
	}
	if b.store != nil && store != nil && filepath.Clean(b.store.Root()) == filepath.Clean(store.Root()) {
		if err := b.store.SetDefaultCompany(""); err != nil {
			return err
		}
		store = b.store
	}
	b.preferences = preferences
	b.legacyRoots = legacyRoots
	b.store = store
	b.message = "题库设置已保存；完成解题后可一键导出当前段"
	return nil
}

func ensureWritableDirectory(path string) error {
	store, err := questionbank.NewStore(path)
	if err != nil {
		return err
	}
	test, err := os.CreateTemp(store.Root(), ".qsolver-write-check-")
	if err != nil {
		return fmt.Errorf("题库目录不可写")
	}
	name := test.Name()
	if err = test.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}

func saveQuestionBankPreferences(file questionBankPreferencesFile) error {
	path, err := bankPreferencesPath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(file)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".questionbank-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func suggestQuestionBankOutput() string {
	cwd, _ := os.Getwd()
	executable, _ := os.Executable()
	for _, start := range []string{cwd, filepath.Dir(executable)} {
		base := start
		for depth := 0; depth < 6; depth++ {
			for _, relative := range []string{filepath.Join("输出", "求职笔试", "题库"), "输出"} {
				candidate := filepath.Join(base, relative)
				if info, err := os.Stat(candidate); err == nil && info.IsDir() {
					return candidate
				}
			}
			parent := filepath.Dir(base)
			if parent == base {
				break
			}
			base = parent
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

// StartQuestionBankSession is retained for old frontends. New capture creates a
// writable segment automatically.
func (a *App) StartQuestionBankSession(company, kind string) error {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return err
	}
	if err := b.store.Start(company, kind); err != nil {
		return err
	}
	b.lastExport = questionbank.ExportResult{}
	b.message = "已新建当前分段"
	return nil
}

func (a *App) ListQuestionBankSessions() ([]questionbank.SessionInfo, error) {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.store == nil {
		return []questionbank.SessionInfo{}, nil
	}
	return b.store.List()
}

func (a *App) OpenQuestionBankSession(id string) error {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return err
	}
	if err := b.store.Open(id); err != nil {
		return err
	}
	b.lastExport = questionbank.ExportResult{}
	b.message = "已打开历史分段"
	return nil
}

func (a *App) ResumeQuestionBankSession() error {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return err
	}
	return b.store.Resume()
}

func (a *App) SaveQuestionBankSession() error {
	a.conversationMu.Lock()
	defer a.conversationMu.Unlock()

	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.store == nil {
		return fmt.Errorf("尚未配置题库目录")
	}
	recoverConversation := b.store.HasPendingCompletion()
	if err := b.store.Flush(); err != nil {
		return err
	}
	b.message = "原始记录已保存"
	if recoverConversation {
		if err := a.restoreCurrentQuestionBankConversationLocked(b); err != nil {
			return fmt.Errorf("原始记录已保存，但模型上下文恢复失败: %w", err)
		}
	}
	return nil
}

func (a *App) SplitQuestionBankSegment() error {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return err
	}
	if err := b.store.Split(); err != nil {
		return err
	}
	b.message = "当前段已封存；下一题会自动进入新段"
	b.lastExport = questionbank.ExportResult{}
	return nil
}

func (a *App) DiscardQuestionBankSegment(confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("请确认清空当前段；原始文件将保留，但不再参与后续导出")
	}
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return err
	}
	if err := b.store.DiscardCurrent(); err != nil {
		return err
	}
	b.message = "当前段已软清空；原始文件仍保留"
	b.lastExport = questionbank.ExportResult{}
	return nil
}

func normalizeQuestionBankType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "综合" {
		return "综合笔试"
	}
	return value
}

func questionBankNamePart(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(value))
	value = strings.TrimRight(value, ". ")
	if value == "" {
		return "未命名"
	}
	runes := []rune(value)
	if len(runes) > 40 {
		value = string(runes[:40])
	}
	return value
}

func inferQuestionBankType(session *questionbank.Session) string {
	counts := make(map[string]int)
	order := make([]string, 0)
	for _, record := range session.Records {
		for _, question := range record.Questions {
			kind := strings.TrimSpace(question.Type)
			if kind == "" {
				continue
			}
			if _, exists := counts[kind]; !exists {
				order = append(order, kind)
			}
			counts[kind]++
		}
	}
	best := ""
	for _, kind := range order {
		if best == "" || counts[kind] > counts[best] {
			best = kind
		}
	}
	if best == "" {
		best = session.Type
	}
	return normalizeQuestionBankType(best)
}

func (a *App) questionBankSessionNumberLocked(session *questionbank.Session) int {
	list, err := a.questionBank.store.List()
	if err != nil {
		return 1
	}
	day := session.StartedAt.Format("20060102")
	type numberedSession struct {
		id      string
		started time.Time
	}
	matches := make([]numberedSession, 0)
	found := false
	for _, item := range list {
		started, parseErr := time.Parse(time.RFC3339Nano, item.StartedAt)
		if parseErr != nil || started.Format("20060102") != day {
			continue
		}
		matches = append(matches, numberedSession{id: item.ID, started: started})
		if item.ID == session.ID {
			found = true
		}
	}
	if !found {
		matches = append(matches, numberedSession{id: session.ID, started: session.StartedAt})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].started.Equal(matches[j].started) {
			return matches[i].id < matches[j].id
		}
		return matches[i].started.Before(matches[j].started)
	})
	for index, item := range matches {
		if item.id == session.ID {
			return index + 1
		}
	}
	return 1
}

func metadataSuggestionValue(records []questionbank.Record, selectValue func(*questionbank.SourceMetadataSuggestion) string) (string, bool) {
	type candidate struct {
		value string
		count int
		order int
	}
	candidates := make(map[string]*candidate)
	order := 0
	for index := range records {
		suggestion := records[index].MetadataSuggestion
		if suggestion == nil {
			continue
		}
		value := strings.TrimSpace(selectValue(suggestion))
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if existing := candidates[key]; existing != nil {
			existing.count++
			continue
		}
		candidates[key] = &candidate{value: value, count: 1, order: order}
		order++
	}
	var best *candidate
	for _, item := range candidates {
		if best == nil || item.count > best.count || item.count == best.count && item.order < best.order {
			best = item
		}
	}
	if best == nil {
		return "", false
	}
	return best.value, len(candidates) > 1
}

func aggregateMetadataSuggestion(records []questionbank.Record) (*questionbank.SourceMetadataSuggestion, bool) {
	company, companyConflict := metadataSuggestionValue(records, func(value *questionbank.SourceMetadataSuggestion) string { return value.Company })
	issuer, issuerConflict := metadataSuggestionValue(records, func(value *questionbank.SourceMetadataSuggestion) string { return value.Issuer })
	kind, typeConflict := metadataSuggestionValue(records, func(value *questionbank.SourceMetadataSuggestion) string { return value.Type })
	notes := make([]string, 0)
	seenNotes := make(map[string]bool)
	for index := range records {
		if suggestion := records[index].MetadataSuggestion; suggestion != nil {
			note := strings.TrimSpace(suggestion.ReviewNote)
			if note != "" && !seenNotes[note] {
				seenNotes[note] = true
				notes = append(notes, note)
			}
		}
	}
	if company == "" && issuer == "" && kind == "" && len(notes) == 0 {
		return nil, false
	}
	return &questionbank.SourceMetadataSuggestion{
		Company: company, Issuer: issuer, Type: kind, ReviewNote: strings.Join(notes, "；"),
	}, companyConflict || issuerConflict || typeConflict
}

func appendQuestionBankNamePart(parts []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return parts
	}
	for _, existing := range parts {
		if strings.EqualFold(existing, value) {
			return parts
		}
	}
	return append(parts, value)
}

func automaticQuestionBankName(numberedPrefix, company, issuer, kind string) string {
	parts := make([]string, 0, 3)
	parts = appendQuestionBankNamePart(parts, company)
	parts = appendQuestionBankNamePart(parts, issuer)
	if len(parts) == 0 {
		parts = append(parts, numberedPrefix)
	}
	parts = appendQuestionBankNamePart(parts, normalizeQuestionBankType(kind))
	for index := range parts {
		parts[index] = questionBankNamePart(parts[index])
	}
	return strings.Join(parts, "_")
}

func questionBankOrganizeState(session *questionbank.Session) (pending, failed int) {
	for _, record := range session.Records {
		if record.Status != questionbank.Completed || len(record.Questions) > 0 {
			continue
		}
		if record.OrganizeError != "" {
			failed++
		} else {
			pending++
		}
	}
	return pending, failed
}

func (a *App) questionBankExportDraftLocked(id string) (QuestionBankExportDraft, error) {
	b := a.questionBank
	if b.store == nil {
		return QuestionBankExportDraft{}, fmt.Errorf("请先在设置中选择题库目录")
	}
	var session *questionbank.Session
	var err error
	id = strings.TrimSpace(id)
	if id == "" {
		session, _ = b.store.Snapshot()
	} else {
		session, err = b.store.SnapshotByID(id)
	}
	if err != nil {
		return QuestionBankExportDraft{}, err
	}
	if session == nil || len(session.Records) == 0 {
		return QuestionBankExportDraft{}, fmt.Errorf("所选分段暂无记录")
	}
	if session.State == questionbank.Discarded {
		return QuestionBankExportDraft{}, fmt.Errorf("已清空的分段不参与导出")
	}
	suggestion, conflict := aggregateMetadataSuggestion(session.Records)
	company := strings.TrimSpace(session.Company)
	issuer := strings.TrimSpace(session.Issuer)
	kind := normalizeQuestionBankType(session.Type)
	if suggestion != nil {
		if company == "" {
			company = suggestion.Company
		}
		if issuer == "" {
			issuer = suggestion.Issuer
		}
		if session.Type == "" || session.Type == "综合" || session.Type == "综合笔试" {
			kind = normalizeQuestionBankType(suggestion.Type)
		}
	}
	if kind == "综合笔试" {
		kind = inferQuestionBankType(session)
	}
	numberedPrefix := fmt.Sprintf("笔试_%s_%02d", session.StartedAt.Format("20060102"), a.questionBankSessionNumberLocked(session))
	name := automaticQuestionBankName(numberedPrefix, company, issuer, kind)
	if session.NameCustomized && strings.TrimSpace(session.Name) != "" {
		name = session.Name
	}
	pending, failed := questionBankOrganizeState(session)
	return QuestionBankExportDraft{
		SessionID:          session.ID,
		Company:            company,
		Issuer:             issuer,
		Type:               kind,
		Name:               name,
		NameCustomized:     session.NameCustomized,
		Directory:          b.store.Root(),
		AutoOrganize:       b.preferences.AutoOrganize,
		NumberedPrefix:     numberedPrefix,
		Count:              len(session.Records),
		StartedAt:          session.StartedAt.Format(time.RFC3339Nano),
		MetadataSuggestion: suggestion,
		SuggestionConflict: conflict,
		OrganizeReady:      pending == 0,
		PendingOrganize:    pending,
		FailedOrganize:     failed,
	}, nil
}

func (a *App) GetQuestionBankExportDraft(id string) (QuestionBankExportDraft, error) {
	b := a.questionBank
	if b == nil {
		return QuestionBankExportDraft{}, fmt.Errorf("题库服务不可用")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return QuestionBankExportDraft{}, err
	}
	return a.questionBankExportDraftLocked(id)
}

func (a *App) questionBankExportConfigForLocked(autoOrganize bool) (config.Config, error) {
	if !autoOrganize {
		return config.Config{}, nil
	}
	if a.configManager == nil {
		return config.Config{}, fmt.Errorf("本次已启用模型整理，请先配置支持图片的模型")
	}
	cfg := a.configManager.Get()
	if cfg.APIKey == "" || cfg.Model == "" {
		return cfg, fmt.Errorf("本次已启用模型整理，请先配置支持图片的模型")
	}
	return cfg, nil
}

func (a *App) questionBankExportConfigLocked() (QuestionBankPreferences, config.Config, error) {
	preferences := a.questionBank.preferences
	cfg, err := a.questionBankExportConfigForLocked(preferences.AutoOrganize)
	return preferences, cfg, err
}

func (a *App) questionBankTargetLocked(id string) (*questionbank.Session, error) {
	b := a.questionBank
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("请选择要处理的分段")
	}
	snapshot, err := b.store.SnapshotByID(id)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || len(snapshot.Records) == 0 {
		return nil, fmt.Errorf("所选分段暂无记录")
	}
	if snapshot.State == questionbank.Discarded {
		return nil, fmt.Errorf("已清空的分段不参与整理或导出")
	}
	if snapshot.State != questionbank.Active && snapshot.State != questionbank.Paused && snapshot.State != questionbank.Ended && snapshot.State != questionbank.Exported {
		return nil, fmt.Errorf("所选分段状态不支持整理或导出")
	}
	for _, record := range snapshot.Records {
		if record.Status == questionbank.Running {
			return nil, fmt.Errorf("所选分段仍有题目正在处理")
		}
	}
	return snapshot, nil
}

func (a *App) sealQuestionBankTargetLocked(snapshot *questionbank.Session) (string, error) {
	if snapshot.State != questionbank.Active && snapshot.State != questionbank.Paused {
		return snapshot.ID, nil
	}
	current, _ := a.questionBank.store.Snapshot()
	if current == nil || current.ID != snapshot.ID {
		return "", fmt.Errorf("只能封存当前活动分段")
	}
	targetID, err := a.questionBank.store.SealCurrent()
	if err != nil {
		return "", err
	}
	if targetID != snapshot.ID {
		return "", fmt.Errorf("分段在封存时发生变化，请重试")
	}
	return targetID, nil
}

func (a *App) startQuestionBankOrganizeLocked(targetID string, cfg config.Config) {
	b := a.questionBank
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.organizing = true
	b.operation = "organizing"
	b.progress = questionbank.Progress{}
	b.message = "正在整理分段并生成待核对建议"
	store := b.store
	provider := llm.NewOpenAIAdapter(&cfg)
	b.workers.Add(1)

	go func() {
		defer b.workers.Done()
		defer cancel()
		organizeErr := store.OrganizeSession(ctx, targetID, provider, cfg.Model, func(progress questionbank.Progress) {
			b.mu.Lock()
			b.progress = progress
			b.mu.Unlock()
			a.notifyQuestionBank()
		})
		b.mu.Lock()
		switch {
		case errors.Is(organizeErr, context.Canceled):
			b.message = "整理已取消；已完成的题目和建议仍保留，可稍后继续"
		case organizeErr != nil:
			b.message = organizeErr.Error() + "；可核对已有结果后导出，或稍后重试"
		default:
			b.message = "整理完成；模型建议尚未成为权威元数据，请核对后再导出"
		}
		b.organizing = false
		b.operation = ""
		b.cancel = nil
		b.mu.Unlock()
		a.notifyQuestionBank()
	}()
}

// PrepareQuestionBankSegment seals the selected active segment when needed and
// runs the existing per-record organizer. It never creates an export directory.
func (a *App) PrepareQuestionBankSegment(sessionID string) error {
	b := a.questionBank
	if b == nil {
		return fmt.Errorf("题库服务不可用")
	}
	b.mu.Lock()
	if err := a.bankAvailableLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	snapshot, err := a.questionBankTargetLocked(sessionID)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	cfg, err := a.questionBankExportConfigForLocked(true)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	targetID, err := a.sealQuestionBankTargetLocked(snapshot)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	a.startQuestionBankOrganizeLocked(targetID, cfg)
	b.mu.Unlock()
	a.notifyQuestionBank()
	return nil
}

func (a *App) startQuestionBankExportWithOptionsLocked(targetID string, autoOrganize bool, options questionbank.ExportOptions, cfg config.Config) {
	b := a.questionBank
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.organizing = true
	b.operation = "exporting"
	b.progress = questionbank.Progress{}
	b.message = "正在导出分段"
	store := b.store
	var provider llm.Provider
	if autoOrganize {
		provider = llm.NewOpenAIAdapter(&cfg)
		b.operation = "organizing"
		b.message = "正在整理并导出分段"
	}
	b.workers.Add(1)

	go func() {
		defer b.workers.Done()
		defer cancel()
		var organizeErr error
		if provider != nil {
			organizeErr = store.OrganizeSession(ctx, targetID, provider, cfg.Model, func(progress questionbank.Progress) {
				b.mu.Lock()
				b.progress = progress
				b.mu.Unlock()
				a.notifyQuestionBank()
			})
		}
		result, exportErr := store.ExportSession(targetID, options)
		b.mu.Lock()
		if result.Directory != "" {
			b.lastExport = result
		}
		switch {
		case exportErr != nil:
			b.message = "导出未完整完成，原始记录仍保留：" + exportErr.Error()
		case errors.Is(organizeErr, context.Canceled):
			b.message = "整理已取消；原稿和已完成的整理结果已导出"
		case organizeErr != nil:
			b.message = organizeErr.Error() + "；原稿和已完成的整理结果已导出"
		case provider != nil:
			b.message = "整理与导出完成；请核对题干和答案"
		default:
			b.message = "分段已导出；未启用模型整理，原始记录完整保留"
		}
		b.organizing = false
		b.operation = ""
		b.cancel = nil
		b.mu.Unlock()
		a.notifyQuestionBank()
	}()
}

func (a *App) startQuestionBankExportLocked(targetID string, preferences QuestionBankPreferences, cfg config.Config) {
	a.startQuestionBankExportWithOptionsLocked(targetID, preferences.AutoOrganize, questionbank.ExportOptions{RawOnly: !preferences.AutoOrganize}, cfg)
}

func (a *App) ExportQuestionBankSegment(request QuestionBankExportRequest) error {
	b := a.questionBank
	if b == nil {
		return fmt.Errorf("题库服务不可用")
	}
	b.mu.Lock()
	if err := a.bankAvailableLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	request.SessionID = strings.TrimSpace(request.SessionID)
	request.Company = strings.TrimSpace(request.Company)
	request.Issuer = strings.TrimSpace(request.Issuer)
	request.Type = normalizeQuestionBankType(request.Type)
	request.Name = strings.TrimSpace(request.Name)
	request.Name = strings.TrimSuffix(strings.TrimSuffix(request.Name, ".md"), ".MD")
	request.Directory = strings.TrimSpace(request.Directory)
	if request.Name == "" {
		b.mu.Unlock()
		return fmt.Errorf("请填写题库名称")
	}
	if len([]rune(request.Company)) > 120 || len([]rune(request.Issuer)) > 120 || len([]rune(request.Type)) > 120 || len([]rune(request.Name)) > 120 {
		b.mu.Unlock()
		return fmt.Errorf("单位、出题方、题型和题库名称最多 120 字")
	}
	snapshot, err := a.questionBankTargetLocked(request.SessionID)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if request.Directory == "" {
		request.Directory = b.store.Root()
	}
	cleanDirectory, err := questionbank.ValidateExportDirectory(request.Directory)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if request.AutoOrganize {
		pending, _ := questionBankOrganizeState(snapshot)
		if pending > 0 {
			b.mu.Unlock()
			return fmt.Errorf("请先整理并生成建议，再核对元数据后导出")
		}
	}
	targetID, err := a.sealQuestionBankTargetLocked(snapshot)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if err = b.store.UpdateSessionMetadata(targetID, request.Name, request.Company, request.Issuer, request.Type, request.NameCustomized); err != nil {
		b.mu.Unlock()
		return err
	}
	options := questionbank.ExportOptions{
		Name:      request.Name,
		Company:   request.Company,
		Issuer:    request.Issuer,
		Type:      request.Type,
		Directory: cleanDirectory,
		RawOnly:   !request.AutoOrganize,
	}
	a.startQuestionBankExportWithOptionsLocked(targetID, false, options, config.Config{})
	b.mu.Unlock()
	a.notifyQuestionBank()
	return nil
}

func (a *App) ExportCurrentQuestionBankSegment() error {
	b := a.questionBank
	b.mu.Lock()
	if err := a.bankAvailableLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	preferences, cfg, err := a.questionBankExportConfigLocked()
	if err != nil {
		b.mu.Unlock()
		return err
	}
	targetID, err := b.store.SealCurrent()
	if err != nil {
		b.mu.Unlock()
		return err
	}
	a.startQuestionBankExportLocked(targetID, preferences, cfg)
	b.mu.Unlock()
	a.notifyQuestionBank()
	return nil
}

// ExportQuestionBankSession is the explicit history action. Unlike the answer-side
// quick export, it may re-export an already exported segment and retries only
// records that still have no structured questions when model organization is enabled.
func (a *App) ExportQuestionBankSession(id string) error {
	b := a.questionBank
	b.mu.Lock()
	if err := a.bankAvailableLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	preferences, cfg, err := a.questionBankExportConfigLocked()
	if err != nil {
		b.mu.Unlock()
		return err
	}
	snapshot, err := b.store.SnapshotByID(strings.TrimSpace(id))
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if snapshot == nil || len(snapshot.Records) == 0 {
		b.mu.Unlock()
		return fmt.Errorf("所选分段暂无记录")
	}
	if snapshot.State != questionbank.Ended && snapshot.State != questionbank.Exported {
		b.mu.Unlock()
		return fmt.Errorf("仅已封存或已导出的历史分段可重新导出")
	}
	for _, record := range snapshot.Records {
		if record.Status == questionbank.Running {
			b.mu.Unlock()
			return fmt.Errorf("所选分段仍有题目正在处理")
		}
	}
	a.startQuestionBankExportLocked(snapshot.ID, preferences, cfg)
	b.mu.Unlock()
	a.notifyQuestionBank()
	return nil
}

// ExportQuestionBank is retained for old frontends and performs local export.
func (a *App) ExportQuestionBank(options questionbank.ExportOptions) (questionbank.ExportResult, error) {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := a.bankAvailableLocked(); err != nil {
		return questionbank.ExportResult{}, err
	}
	targetID, err := b.store.SealCurrent()
	if err != nil {
		return questionbank.ExportResult{}, err
	}
	result, err := b.store.ExportSession(targetID, options)
	if err == nil {
		b.lastExport = result
		b.message = "当前段已导出"
	}
	return result, err
}

// OrganizeQuestionBank is retained for old frontends. It now creates only one
// user-visible export after organizing the locked target segment.
func (a *App) OrganizeQuestionBank(options questionbank.ExportOptions, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("请先确认发送当前段截图和答案进行额外模型整理")
	}
	b := a.questionBank
	b.mu.Lock()
	if err := a.bankAvailableLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	cfg := a.configManager.Get()
	if cfg.APIKey == "" || cfg.Model == "" {
		b.mu.Unlock()
		return fmt.Errorf("请先配置支持图片的模型")
	}
	targetID, err := b.store.SealCurrent()
	if err != nil {
		b.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.organizing = true
	b.operation = "organizing"
	b.progress = questionbank.Progress{}
	b.message = "正在整理当前段"
	store := b.store
	provider := llm.NewOpenAIAdapter(&cfg)
	b.workers.Add(1)
	b.mu.Unlock()

	go func() {
		defer b.workers.Done()
		defer cancel()
		organizeErr := store.OrganizeSession(ctx, targetID, provider, cfg.Model, func(progress questionbank.Progress) {
			b.mu.Lock()
			b.progress = progress
			b.mu.Unlock()
			a.notifyQuestionBank()
		})
		result, exportErr := store.ExportSession(targetID, options)
		b.mu.Lock()
		if result.Directory != "" {
			b.lastExport = result
		}
		switch {
		case exportErr != nil:
			b.message = "整理已停止，但导出未完整完成；原始记录仍保留：" + exportErr.Error()
		case errors.Is(organizeErr, context.Canceled):
			b.message = "整理已取消；原稿与已完成结果均已导出"
		case organizeErr != nil:
			b.message = organizeErr.Error() + "；原稿仍保留"
		default:
			b.message = "整理完成，已导出题库；请核对题干和答案"
		}
		b.organizing = false
		b.operation = ""
		b.cancel = nil
		b.mu.Unlock()
		a.notifyQuestionBank()
	}()
	return nil
}

func (a *App) CancelQuestionBankOrganizing() {
	b := a.questionBank
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		b.cancel()
	}
}
