package questionbank

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const currentSessionVersion = 2

type Store struct {
	mu             sync.Mutex
	root           string
	recordsRoot    string
	legacyRoots    []string
	defaultCompany string
	current        *Session
	currentBase    string
	pending        map[string][]byte
	dirty          bool
	lastSave       time.Time
}

func NewStore(root string) (*Store, error) {
	return NewStoreWithLegacy(root, nil)
}

func NewStoreWithLegacy(root string, legacyRoots []string) (*Store, error) {
	path, err := validateArchiveRoot(root)
	if err != nil {
		return nil, err
	}
	recordsRoot := filepath.Join(path, "_记录")
	seen := map[string]bool{strings.ToLower(filepath.Clean(recordsRoot)): true}
	legacy := make([]string, 0, len(legacyRoots))
	for _, candidate := range legacyRoots {
		if strings.TrimSpace(candidate) == "" || !filepath.IsAbs(candidate) {
			continue
		}
		absolute, absErr := filepath.Abs(candidate)
		if absErr != nil {
			continue
		}
		resolved, evalErr := filepath.EvalSymlinks(absolute)
		if evalErr != nil {
			if os.IsNotExist(evalErr) {
				continue
			}
			return nil, fmt.Errorf("旧题库记录目录不可访问: %w", evalErr)
		}
		info, statErr := os.Stat(resolved)
		if statErr != nil || !info.IsDir() {
			continue
		}
		key := strings.ToLower(filepath.Clean(resolved))
		if seen[key] {
			continue
		}
		seen[key] = true
		legacy = append(legacy, resolved)
	}
	return &Store{
		root:        path,
		recordsRoot: recordsRoot,
		legacyRoots: legacy,
		pending:     make(map[string][]byte),
	}, nil
}

func validateArchiveRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) {
		return "", fmt.Errorf("请明确选择绝对路径的题库目录")
	}
	path, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("题库目录不存在或不可访问: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("题库路径不是目录")
	}
	for dir := path; ; dir = filepath.Dir(dir) {
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			return "", fmt.Errorf("题库目录不能位于 Git 仓库内，请选择仓库外目录")
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return path, nil
}

// ValidateExportDirectory checks a one-off export destination without creating
// recovery storage there.
func ValidateExportDirectory(root string) (string, error) {
	path, err := validateArchiveRoot(root)
	if err != nil {
		return "", err
	}
	test, err := os.CreateTemp(path, ".qsolver-export-check-")
	if err != nil {
		return "", fmt.Errorf("题库目录不可写")
	}
	name := test.Name()
	if err = test.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	if err = os.Remove(name); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) Root() string        { return s.root }
func (s *Store) sessionsDir() string { return s.recordsRoot }

// sessionDir is intentionally stable for tests and internal file access. Existing
// legacy sessions continue to use their original directory.
func (s *Store) sessionDir(id string) string {
	if s.current != nil && s.current.ID == id && s.currentBase != "" {
		return filepath.Join(s.currentBase, id)
	}
	if base, ok := s.findSessionBase(id); ok {
		return filepath.Join(base, id)
	}
	return filepath.Join(s.recordsRoot, id)
}

func (s *Store) SetDefaultCompany(company string) error {
	company = strings.TrimSpace(company)
	if len([]rune(company)) > 120 {
		return fmt.Errorf("单位名称最多 120 字")
	}
	s.mu.Lock()
	s.defaultCompany = company
	s.mu.Unlock()
	return nil
}

func (s *Store) blockedLocked() bool {
	return s.dirty || (s.current != nil && (s.current.State == Active || s.current.State == Paused))
}

// Start remains as a compatibility entry point. Normal Change 012 capture uses
// Begin, which creates the writable segment lazily.
func (s *Store) Start(company, kind string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blockedLocked() {
		return fmt.Errorf("请先封存当前分段并保存未落盘记录")
	}
	return s.createCurrentLocked(company, kind)
}

func (s *Store) createCurrentLocked(company, kind string) error {
	company = strings.TrimSpace(company)
	kind = strings.TrimSpace(kind)
	if len([]rune(company)) > 120 || len([]rune(kind)) > 120 {
		return fmt.Errorf("单位和题型最多 120 字")
	}
	id, err := newID()
	if err != nil {
		return err
	}
	if kind == "" {
		kind = "综合"
	}
	s.current = &Session{
		Version:   currentSessionVersion,
		ID:        id,
		Company:   company,
		Type:      kind,
		StartedAt: time.Now(),
		State:     Active,
		Records:   []Record{},
	}
	s.currentBase = s.recordsRoot
	s.pending = make(map[string][]byte)
	s.dirty = true
	return s.saveLocked()
}

func cloneSession(session *Session) *Session {
	if session == nil {
		return nil
	}
	data, _ := json.Marshal(session)
	var result Session
	_ = json.Unmarshal(data, &result)
	return &result
}

func (s *Store) Snapshot() (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSession(s.current), s.dirty
}

func (s *Store) SnapshotByID(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil && s.current.ID == id {
		return cloneSession(s.current), nil
	}
	session, _, err := s.readSession(id)
	if err != nil {
		return nil, err
	}
	return cloneSession(session), nil
}

func (s *Store) ensureWritableLocked() error {
	if s.current != nil {
		switch s.current.State {
		case Active:
			return nil
		case Paused:
			s.current.State = Active
			s.dirty = true
			return s.saveLocked()
		default:
			if s.dirty {
				return fmt.Errorf("当前分段尚未成功落盘，请先重试保存")
			}
		}
	}
	return s.createCurrentLocked(s.defaultCompany, "综合")
}

func (s *Store) Begin(images []string, model, scene string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(images) == 0 || len(images) > 3 {
		return "", fmt.Errorf("每次提交需要 1～3 张截图")
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	r := Record{ID: id, CreatedAt: time.Now(), Model: model, Scene: scene, Status: Running, Images: []string{}}
	pending := make(map[string][]byte)
	for i, img := range images {
		data, ext, decodeErr := decodeImage(img)
		if decodeErr != nil {
			return "", decodeErr
		}
		name := fmt.Sprintf("assets/%s-%d%s", id, i+1, ext)
		r.Images = append(r.Images, name)
		pending[name] = data
	}

	writableErr := s.ensureWritableLocked()
	if writableErr != nil && (s.current == nil || s.current.State != Active) {
		return "", writableErr
	}
	for name, data := range pending {
		s.pending[name] = data
	}
	s.current.Records = append(s.current.Records, r)
	s.dirty = true
	saveErr := s.saveLocked()
	if writableErr != nil {
		return id, writableErr
	}
	return id, saveErr
}

func (s *Store) recordLocked(id string) *Record {
	if s.current == nil {
		return nil
	}
	for i := range s.current.Records {
		if s.current.Records[i].ID == id {
			return &s.current.Records[i]
		}
	}
	return nil
}

func findRecord(session *Session, id string) *Record {
	if session == nil {
		return nil
	}
	for i := range session.Records {
		if session.Records[i].ID == id {
			return &session.Records[i]
		}
	}
	return nil
}

func (s *Store) Append(id, chunk string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recordLocked(id)
	if r == nil || r.Status != Running {
		return nil
	}
	r.Answer += chunk
	s.dirty = true
	if time.Since(s.lastSave) >= time.Second {
		return s.saveLocked()
	}
	return nil
}

func (s *Store) Finish(id, answer, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recordLocked(id)
	if r == nil || r.Status != Running {
		return nil
	}
	switch status {
	case Completed, Failed, Cancelled, Interrupted, Empty:
	default:
		return fmt.Errorf("无效记录状态")
	}
	if status == Completed {
		r.Answer = answer
		if answer == "" {
			status = Empty
		}
	}
	r.Status = status
	s.dirty = true
	return s.saveLocked()
}

func (s *Store) SealCurrent() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sealCurrentLocked()
}

func (s *Store) sealCurrentLocked() (string, error) {
	if s.current == nil || len(s.current.Records) == 0 {
		return "", fmt.Errorf("当前分段暂无记录")
	}
	if s.current.State == Exported {
		return "", fmt.Errorf("当前分段已导出；完成下一题后会自动进入新段")
	}
	if s.current.State == Discarded {
		return "", fmt.Errorf("当前分段已清空；完成下一题后会自动进入新段")
	}
	for _, r := range s.current.Records {
		if r.Status == Running {
			return "", fmt.Errorf("还有题目正在处理，请完成或取消后再分段")
		}
	}
	if s.current.State == Active || s.current.State == Paused {
		now := time.Now()
		s.current.EndedAt = &now
		s.current.State = Ended
		s.dirty = true
	}
	if err := s.saveLocked(); err != nil {
		return "", err
	}
	return s.current.ID, nil
}

func (s *Store) Split() error {
	_, err := s.SealCurrent()
	return err
}

func (s *Store) End() error {
	_, err := s.SealCurrent()
	return err
}

func (s *Store) DiscardCurrent() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || len(s.current.Records) == 0 {
		return fmt.Errorf("当前分段暂无记录")
	}
	switch s.current.State {
	case Exported:
		return fmt.Errorf("当前分段已导出，不能清空历史结果")
	case Discarded:
		return fmt.Errorf("当前分段已清空")
	}
	for _, r := range s.current.Records {
		if r.Status == Running {
			return fmt.Errorf("还有题目正在处理，不能清空当前分段")
		}
	}
	now := time.Now()
	s.current.EndedAt = &now
	s.current.DiscardedAt = &now
	s.current.State = Discarded
	s.dirty = true
	return s.saveLocked()
}

func (s *Store) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || s.current.State != Paused {
		return fmt.Errorf("没有可继续的分段")
	}
	s.current.State = Active
	s.dirty = true
	return s.saveLocked()
}

func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if !s.dirty || s.current == nil {
		return nil
	}
	base := s.currentBase
	if base == "" {
		base = s.recordsRoot
		s.currentBase = base
	}
	if err := s.saveSessionLocked(s.current, base, s.pending); err != nil {
		return err
	}
	clear(s.pending)
	s.dirty = false
	s.lastSave = time.Now()
	return nil
}

func (s *Store) saveSessionLocked(session *Session, base string, pending map[string][]byte) error {
	if session == nil || !validID(session.ID) {
		return fmt.Errorf("分段记录无效")
	}
	if filepath.Clean(base) == filepath.Clean(s.recordsRoot) {
		if err := mkdirWithin(s.root, "_记录"); err != nil {
			return err
		}
	}
	if err := mkdirWithin(base, filepath.ToSlash(filepath.Join(session.ID, "assets"))); err != nil {
		return err
	}
	dir := filepath.Join(base, session.ID)
	for name, data := range pending {
		path, err := safeWriteChild(dir, name)
		if err != nil {
			return err
		}
		if err = atomicWrite(path, data); err != nil {
			return err
		}
	}
	return atomicJSON(filepath.Join(dir, "session.json"), session)
}

func safeWriteChild(root, relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("无效归档相对路径")
	}
	path := filepath.Join(root, clean)
	parent, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if parent != rootAbs && !strings.HasPrefix(strings.ToLower(parent), strings.ToLower(rootAbs+string(filepath.Separator))) {
		return "", fmt.Errorf("无效归档相对路径")
	}
	return path, nil
}

func (s *Store) findSessionBase(id string) (string, bool) {
	if !validID(id) {
		return "", false
	}
	for _, base := range append([]string{s.recordsRoot}, s.legacyRoots...) {
		info, err := os.Lstat(filepath.Join(base, id, "session.json"))
		if err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			return base, true
		}
	}
	return "", false
}

func (s *Store) readSession(id string) (*Session, string, error) {
	if !validID(id) {
		return nil, "", fmt.Errorf("分段 ID 无效")
	}
	base, ok := s.findSessionBase(id)
	if !ok {
		return nil, "", fmt.Errorf("分段记录不存在: %s", id)
	}
	session, err := s.readSessionFromBase(base, id)
	return session, base, err
}

func (s *Store) readSessionFromBase(base, id string) (*Session, error) {
	path, err := safeChild(base, id+"/session.json")
	if err != nil {
		return nil, err
	}
	data, err := readLimited(path, 64*1024*1024)
	if err != nil {
		return nil, err
	}
	var session Session
	if err = json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("分段记录损坏: %s", id)
	}
	if (session.Version != 1 && session.Version != currentSessionVersion) || session.ID != id {
		return nil, fmt.Errorf("分段版本或 ID 不匹配")
	}
	switch session.State {
	case Active, Paused, Ended, Exported, Discarded:
	default:
		return nil, fmt.Errorf("分段状态无效")
	}
	ids := make(map[string]bool)
	for _, r := range session.Records {
		if !validID(r.ID) || ids[r.ID] || len(r.Images) < 1 || len(r.Images) > 3 {
			return nil, fmt.Errorf("题目记录无效")
		}
		ids[r.ID] = true
		switch r.Status {
		case Running, Completed, Failed, Cancelled, Interrupted, Empty:
		default:
			return nil, fmt.Errorf("题目状态无效")
		}
		for _, name := range r.Images {
			if _, childErr := safeChild(filepath.Join(base, id), name); childErr != nil {
				return nil, fmt.Errorf("题目截图缺失或路径无效: %w", childErr)
			}
		}
	}
	return &session, nil
}

func (s *Store) Open(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil && s.current.ID == id {
		return nil
	}
	if s.blockedLocked() {
		return fmt.Errorf("请先封存当前分段并保存未落盘记录")
	}
	session, base, err := s.readSession(id)
	if err != nil {
		return err
	}
	s.current = session
	s.currentBase = base
	s.pending = make(map[string][]byte)
	if session.State == Active {
		session.State = Paused
		s.dirty = true
	}
	for i := range session.Records {
		if session.Records[i].Status == Running {
			session.Records[i].Status = Interrupted
			s.dirty = true
		}
	}
	return s.saveLocked()
}

func (s *Store) List() ([]SessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := []SessionInfo{}
	seen := make(map[string]bool)
	for _, base := range append([]string{s.recordsRoot}, s.legacyRoots...) {
		entries, err := os.ReadDir(base)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			id := entry.Name()
			if !entry.IsDir() || !validID(id) || seen[id] {
				continue
			}
			session, readErr := s.readSessionFromBase(base, id)
			if readErr != nil {
				return nil, readErr
			}
			seen[id] = true
			result = append(result, SessionInfo{
				ID:              session.ID,
				Company:         session.Company,
				Type:            session.Type,
				State:           session.State,
				Count:           len(session.Records),
				StartedAt:       session.StartedAt.Format(time.RFC3339Nano),
				ExportDirectory: session.ExportDirectory,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339Nano, result[i].StartedAt)
		right, _ := time.Parse(time.RFC3339Nano, result[j].StartedAt)
		return left.After(right)
	})
	return result, nil
}
