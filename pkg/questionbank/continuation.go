package questionbank

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"Q-Solver/pkg/answersummary"
)

const maxResolvedConversationBytes = 80 * 1024 * 1024

type ActivationResult struct {
	Previous *Session
	Current  *Session
}

type ResolvedTurn struct {
	Ref    ConversationTurnRef
	Images []string
	Answer string
}

func cloneConversationSnapshot(snapshot *ConversationSnapshot) *ConversationSnapshot {
	if snapshot == nil {
		return nil
	}
	cloned := *snapshot
	cloned.Turns = append([]ConversationTurnRef(nil), snapshot.Turns...)
	return &cloned
}

func validateConversationSnapshot(snapshot *ConversationSnapshot) error {
	if snapshot == nil {
		return nil
	}
	if snapshot.Version != 1 {
		return fmt.Errorf("对话快照版本无效")
	}
	if snapshot.Quality != ConversationQualityPersisted && snapshot.Quality != ConversationQualityReconstructed {
		return fmt.Errorf("对话快照质量标记无效")
	}
	if len(snapshot.Turns) > MaxConversationTurns {
		return fmt.Errorf("对话快照最多保留 %d 轮", MaxConversationTurns)
	}
	seen := make(map[ConversationTurnRef]bool, len(snapshot.Turns))
	for _, ref := range snapshot.Turns {
		if !validID(ref.SessionID) || !validID(ref.RecordID) {
			return fmt.Errorf("对话快照引用无效")
		}
		if seen[ref] {
			return fmt.Errorf("对话快照包含重复引用")
		}
		seen[ref] = true
	}
	return nil
}

func (s *Store) ConversationSnapshotByID(id string) (*ConversationSnapshot, error) {
	session, err := s.SnapshotByID(id)
	if err != nil {
		return nil, err
	}
	if session.Conversation != nil {
		if err = validateConversationSnapshot(session.Conversation); err != nil {
			return nil, err
		}
		return cloneConversationSnapshot(session.Conversation), nil
	}

	turns := make([]ConversationTurnRef, 0, MaxConversationTurns)
	for _, record := range session.Records {
		if record.Status != Completed || strings.TrimSpace(record.Answer) == "" {
			continue
		}
		turns = append(turns, ConversationTurnRef{SessionID: session.ID, RecordID: record.ID})
		if len(turns) > MaxConversationTurns {
			turns = turns[len(turns)-MaxConversationTurns:]
		}
	}
	return &ConversationSnapshot{
		Version:   1,
		Quality:   ConversationQualityReconstructed,
		Turns:     turns,
		UpdatedAt: time.Now(),
	}, nil
}

func (s *Store) ResolveConversation(snapshot ConversationSnapshot) ([]ResolvedTurn, error) {
	if err := validateConversationSnapshot(&snapshot); err != nil {
		return nil, err
	}
	resolved := make([]ResolvedTurn, 0, len(snapshot.Turns))
	totalBytes := 0
	for _, ref := range snapshot.Turns {
		session, err := s.SnapshotByID(ref.SessionID)
		if err != nil {
			return nil, fmt.Errorf("无法读取对话引用分段: %w", err)
		}
		record := findRecord(session, ref.RecordID)
		if record == nil || record.Status != Completed || strings.TrimSpace(record.Answer) == "" {
			return nil, fmt.Errorf("对话引用不是已完成的有效记录")
		}
		turn := ResolvedTurn{Ref: ref, Answer: record.Answer, Images: make([]string, 0, len(record.Images))}
		totalBytes += len(record.Answer)
		for imageIndex := range record.Images {
			dataURL, imageErr := s.ReadRecordImage(ref.SessionID, ref.RecordID, imageIndex)
			if imageErr != nil {
				return nil, fmt.Errorf("无法恢复对话截图: %w", imageErr)
			}
			totalBytes += len(dataURL)
			if totalBytes > maxResolvedConversationBytes {
				return nil, fmt.Errorf("恢复的模型上下文超过 80 MiB，请先开始新的分段")
			}
			turn.Images = append(turn.Images, dataURL)
		}
		resolved = append(resolved, turn)
	}
	return resolved, nil
}

func hasRunningRecord(session *Session) bool {
	if session == nil {
		return false
	}
	for _, record := range session.Records {
		if record.Status == Running {
			return true
		}
	}
	return false
}

func (s *Store) ActivateHistoricalSession(id string) (ActivationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureWriteAllowedLocked(); err != nil {
		return ActivationResult{}, err
	}
	if !validID(id) {
		return ActivationResult{}, fmt.Errorf("分段 ID 无效")
	}
	if s.dirty || len(s.pending) > 0 {
		return ActivationResult{}, fmt.Errorf("当前分段仍有未落盘内容，不能切换")
	}

	previous := cloneSession(s.current)
	previousBase := s.currentBase
	if previousBase == "" {
		previousBase = s.recordsRoot
	}
	if hasRunningRecord(previous) {
		return ActivationResult{}, fmt.Errorf("当前分段还有题目正在处理，不能切换")
	}

	var target *Session
	var targetBase string
	var err error
	if s.current != nil && s.current.ID == id {
		target = cloneSession(s.current)
		targetBase = previousBase
	} else {
		target, targetBase, err = s.readSession(id)
		if err != nil {
			return ActivationResult{}, err
		}
	}
	if hasRunningRecord(target) {
		return ActivationResult{}, fmt.Errorf("目标分段存在未完成记录，不能继续")
	}
	if target.State != Ended && target.State != Exported && target.State != Paused {
		if target.State == Discarded {
			return ActivationResult{}, fmt.Errorf("已清空分段永久只读，不能继续")
		}
		return ActivationResult{}, fmt.Errorf("只有已结束、已导出或已暂停的分段可以继续")
	}

	if previous != nil && previous.ID == id {
		target.State = Active
		if err = s.saveSessionLocked(target, targetBase, nil); err != nil {
			return ActivationResult{}, err
		}
		s.current = target
		s.currentBase = targetBase
		s.pending = make(map[string][]byte)
		s.dirty = false
		return ActivationResult{Previous: previous, Current: cloneSession(target)}, nil
	}

	var pausedSource *Session
	if previous != nil && previous.State == Active {
		pausedSource = cloneSession(previous)
		pausedSource.State = Paused
		if err = s.saveSessionLocked(pausedSource, previousBase, nil); err != nil {
			return ActivationResult{}, fmt.Errorf("暂停当前分段失败: %w", err)
		}
	}

	activatedTarget := cloneSession(target)
	activatedTarget.State = Active
	if err = s.saveSessionLocked(activatedTarget, targetBase, nil); err != nil {
		if pausedSource != nil {
			if rollbackErr := s.saveSessionLocked(previous, previousBase, nil); rollbackErr != nil {
				failure := fmt.Errorf("激活目标分段失败: %v；恢复原分段也失败: %w", err, rollbackErr)
				s.current = pausedSource
				s.currentBase = previousBase
				s.pending = make(map[string][]byte)
				s.pendingCompletion = nil
				s.dirty = false
				s.writeBlocked = failure
				return ActivationResult{}, failure
			}
		}
		return ActivationResult{}, fmt.Errorf("激活目标分段失败: %w", err)
	}

	s.current = activatedTarget
	s.currentBase = targetBase
	s.pending = make(map[string][]byte)
	s.dirty = false
	return ActivationResult{Previous: previous, Current: cloneSession(activatedTarget)}, nil
}

func (s *Store) FinishWithConversation(recordID, answer, status string, snapshot *ConversationSnapshot) error {
	return s.FinishWithConversationResult(recordID, answer, nil, status, snapshot)
}

func (s *Store) FinishWithConversationResult(recordID, answerText string, items []answersummary.Item, status string, snapshot *ConversationSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureWriteAllowedLocked(); err != nil {
		return err
	}
	if status != Completed && status != Failed && status != Cancelled && status != Interrupted && status != Empty {
		return fmt.Errorf("无效记录状态")
	}
	if status == Completed && strings.TrimSpace(answerText) == "" {
		status = Empty
	}
	if status == Completed {
		if snapshot == nil {
			return fmt.Errorf("完成记录必须同时保存对话快照")
		}
		if err := validateConversationSnapshot(snapshot); err != nil {
			return err
		}
	}
	if s.current == nil {
		return fmt.Errorf("当前没有可写分段")
	}
	working := cloneSession(s.current)
	record := findRecord(working, recordID)
	if record == nil || record.Status != Running {
		return fmt.Errorf("未找到正在处理的题目记录")
	}
	record.Answer = answerText
	record.Status = status
	if status == Completed {
		record.AnswerItems = cloneAnswerItems(items)
		working.Conversation = cloneConversationSnapshot(snapshot)
	} else {
		record.AnswerItems = nil
	}
	base := s.currentBase
	if base == "" {
		base = s.recordsRoot
	}
	if err := s.saveSessionLocked(working, base, s.pending); err != nil {
		s.pendingCompletion = working
		s.dirty = true
		return err
	}
	s.current = working
	s.pendingCompletion = nil
	s.currentBase = base
	clear(s.pending)
	s.dirty = false
	s.lastSave = time.Now()
	return nil
}

func (s *Store) CurrentNumberingContext(excludeRecordID string) answersummary.NumberingContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return answersummary.NumberingContext{}
	}
	limit := len(s.current.Records)
	if excludeRecordID != "" {
		for i := range s.current.Records {
			if s.current.Records[i].ID == excludeRecordID {
				limit = i
				break
			}
		}
	}

	labels := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		record := s.current.Records[i]
		if record.Status != Completed {
			continue
		}
		if len(record.AnswerItems) > 0 {
			if _, err := answersummary.BuildCopyText(record.AnswerItems); err == nil {
				for _, item := range record.AnswerItems {
					if item.Label == "待核对" || strings.Contains(item.Label, "答案") || strings.ContainsRune(item.Label, '|') {
						continue
					}
					labels = append(labels, item.Label)
				}
				continue
			}
		}
		labels = append(labels, answersummary.ExtractLegacyLabels(record.Answer)...)
	}

	labels = recentUniqueNumberingLabels(labels)
	ctx := answersummary.NumberingContext{PreviousLabels: labels}
	if len(labels) > 0 {
		ctx.SuggestedNextLabel = nextDecimalLabel(labels[len(labels)-1])
	}
	return ctx
}

func recentUniqueNumberingLabels(labels []string) []string {
	const maxLabels = 64
	seen := make(map[string]struct{}, len(labels))
	reversed := make([]string, 0, min(len(labels), maxLabels))
	for i := len(labels) - 1; i >= 0 && len(reversed) < maxLabels; i-- {
		if _, exists := seen[labels[i]]; exists {
			continue
		}
		seen[labels[i]] = struct{}{}
		reversed = append(reversed, labels[i])
	}
	result := make([]string, len(reversed))
	for i := range reversed {
		result[len(reversed)-1-i] = reversed[i]
	}
	return result
}

func nextDecimalLabel(label string) string {
	if label == "" {
		return ""
	}
	for i := 0; i < len(label); i++ {
		if label[i] < '0' || label[i] > '9' {
			return ""
		}
	}
	value, err := strconv.ParseUint(label, 10, 64)
	if err != nil || value == ^uint64(0) {
		return ""
	}
	return strconv.FormatUint(value+1, 10)
}

func appendExportSnapshot(session *Session, exportedAt time.Time, directory string) {
	if session == nil || strings.TrimSpace(directory) == "" {
		return
	}
	if session.ExportedAt != nil && strings.TrimSpace(session.ExportDirectory) != "" {
		legacy := ExportSnapshot{ExportedAt: *session.ExportedAt, Directory: session.ExportDirectory}
		if !hasExportSnapshot(session.Exports, legacy) {
			session.Exports = append(session.Exports, legacy)
		}
	}
	current := ExportSnapshot{ExportedAt: exportedAt, Directory: directory}
	if !hasExportSnapshot(session.Exports, current) {
		session.Exports = append(session.Exports, current)
	}
}

func hasExportSnapshot(exports []ExportSnapshot, candidate ExportSnapshot) bool {
	for _, existing := range exports {
		if existing.Directory == candidate.Directory && existing.ExportedAt.Equal(candidate.ExportedAt) {
			return true
		}
	}
	return false
}

// ClearCurrentConversation persists an explicit empty snapshot: nil would trigger
// legacy reconstruction and resurrect the conversations being cleared.
func (s *Store) ClearCurrentConversation() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureStorageWriteAllowedLocked(); err != nil {
		return err
	}
	if s.current == nil || (s.current.State != Active && s.current.State != Paused) {
		return nil
	}
	candidate := cloneSession(s.current)
	if s.pendingCompletion != nil {
		candidate = cloneSession(s.pendingCompletion)
	}
	candidate.Conversation = &ConversationSnapshot{Version: 1, Quality: ConversationQualityPersisted, UpdatedAt: time.Now()}
	base := s.currentBase
	if base == "" {
		base = s.recordsRoot
	}
	if err := s.saveSessionLocked(candidate, base, s.pending); err != nil {
		s.pendingCompletion = candidate
		s.dirty = true
		return err
	}
	s.current = candidate
	s.currentBase = base
	s.pendingCompletion = nil
	clear(s.pending)
	s.dirty = false
	s.lastSave = time.Now()
	return nil
}
