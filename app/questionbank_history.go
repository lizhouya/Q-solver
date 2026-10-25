package app

import (
	"Q-Solver/pkg/answersummary"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"Q-Solver/pkg/questionbank"
)

const (
	recentQuestionBankSessionLimit = 20
	questionBankDetailMaxBytes     = 8 * 1024 * 1024
)

type QuestionBankSessionSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Company        string `json:"company"`
	Issuer         string `json:"issuer,omitempty"`
	Type           string `json:"type"`
	NameCustomized bool   `json:"nameCustomized,omitempty"`
	State          string `json:"state"`
	Count          int    `json:"count"`
	StartedAt      string `json:"startedAt"`
}

type QuestionBankSessionList struct {
	Sessions []QuestionBankSessionSummary `json:"sessions"`
	Total    int                          `json:"total"`
	Hidden   int                          `json:"hidden"`
	Limit    int                          `json:"limit"`
}

type QuestionBankImageInfo struct {
	Index int    `json:"index"`
	MIME  string `json:"mime"`
}

type QuestionBankRecordDetail struct {
	AnswerItems        []answersummary.Item                   `json:"answerItems,omitempty"`
	AnswerCopyText     string                                 `json:"answerCopyText,omitempty"`
	ID                 string                                 `json:"id"`
	CreatedAt          string                                 `json:"createdAt"`
	Model              string                                 `json:"model"`
	Scene              string                                 `json:"scene"`
	Answer             string                                 `json:"answer"`
	Status             string                                 `json:"status"`
	OrganizeError      string                                 `json:"organizeError,omitempty"`
	MetadataSuggestion *questionbank.SourceMetadataSuggestion `json:"metadataSuggestion,omitempty"`
	Images             []QuestionBankImageInfo                `json:"images"`
	Questions          []questionbank.Question                `json:"questions,omitempty"`
	KeepContext        *bool                                  `json:"keepContext,omitempty"`
}

type QuestionBankSessionDetail struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name,omitempty"`
	Company        string                     `json:"company"`
	Issuer         string                     `json:"issuer,omitempty"`
	Type           string                     `json:"type"`
	NameCustomized bool                       `json:"nameCustomized,omitempty"`
	State          string                     `json:"state"`
	Count          int                        `json:"count"`
	StartedAt      string                     `json:"startedAt"`
	EndedAt        string                     `json:"endedAt,omitempty"`
	ExportedAt     string                     `json:"exportedAt,omitempty"`
	DiscardedAt    string                     `json:"discardedAt,omitempty"`
	Records        []QuestionBankRecordDetail `json:"records"`
}

func (a *App) questionBankHistoryStore() (*questionbank.Store, error) {
	if a.questionBank == nil {
		return nil, fmt.Errorf("题库服务不可用")
	}
	a.questionBank.mu.Lock()
	defer a.questionBank.mu.Unlock()
	if a.questionBank.store == nil {
		return nil, nil
	}
	if a.questionBank.closing {
		return nil, fmt.Errorf("应用正在关闭")
	}
	return a.questionBank.store, nil
}

func recentQuestionBankSessions(store *questionbank.Store) ([]questionbank.SessionInfo, error) {
	all, err := store.List()
	if err != nil {
		return nil, err
	}
	if len(all) > recentQuestionBankSessionLimit {
		all = all[:recentQuestionBankSessionLimit]
	}
	return all, nil
}

func recentQuestionBankContains(store *questionbank.Store, id string) (bool, error) {
	recent, err := recentQuestionBankSessions(store)
	if err != nil {
		return false, err
	}
	for _, session := range recent {
		if session.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func (a *App) ListRecentQuestionBankSessions() (QuestionBankSessionList, error) {
	result := QuestionBankSessionList{Sessions: []QuestionBankSessionSummary{}, Limit: recentQuestionBankSessionLimit}
	store, err := a.questionBankHistoryStore()
	if err != nil || store == nil {
		return result, err
	}
	all, err := store.List()
	if err != nil {
		return result, err
	}
	result.Total = len(all)
	if result.Total > result.Limit {
		result.Hidden = result.Total - result.Limit
		all = all[:result.Limit]
	}
	for _, session := range all {
		result.Sessions = append(result.Sessions, QuestionBankSessionSummary{
			ID:             session.ID,
			Name:           session.Name,
			Company:        session.Company,
			Issuer:         session.Issuer,
			Type:           session.Type,
			NameCustomized: session.NameCustomized,
			State:          session.State,
			Count:          session.Count,
			StartedAt:      session.StartedAt,
		})
	}
	return result, nil
}

func detailTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func imageMIME(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return ""
	}
}

func buildQuestionBankSessionDetail(session *questionbank.Session) QuestionBankSessionDetail {
	detail := QuestionBankSessionDetail{
		ID:             session.ID,
		Name:           session.Name,
		Company:        session.Company,
		Issuer:         session.Issuer,
		Type:           session.Type,
		NameCustomized: session.NameCustomized,
		State:          session.State,
		Count:          len(session.Records),
		StartedAt:      session.StartedAt.Format(time.RFC3339Nano),
		EndedAt:        detailTime(session.EndedAt),
		ExportedAt:     detailTime(session.ExportedAt),
		DiscardedAt:    detailTime(session.DiscardedAt),
		Records:        make([]QuestionBankRecordDetail, 0, len(session.Records)),
	}
	for _, record := range session.Records {
		recordDetail := QuestionBankRecordDetail{
			ID:                 record.ID,
			CreatedAt:          record.CreatedAt.Format(time.RFC3339Nano),
			Model:              record.Model,
			Scene:              record.Scene,
			Answer:             record.Answer,
			Status:             record.Status,
			OrganizeError:      record.OrganizeError,
			MetadataSuggestion: record.MetadataSuggestion,
			Images:             make([]QuestionBankImageInfo, 0, len(record.Images)),
			Questions:          record.Questions,
			KeepContext:        record.KeepContext,
		}
		if record.Status == questionbank.Completed {
			if len(record.AnswerItems) > 0 {
				recordDetail.AnswerItems = append([]answersummary.Item(nil), record.AnswerItems...)
				recordDetail.AnswerCopyText, _ = answersummary.BuildCopyText(record.AnswerItems)
			} else if result, err := answersummary.Parse(record.Answer); err == nil {
				recordDetail.AnswerItems = result.Items
				recordDetail.AnswerCopyText = result.CopyText
			}
		}
		for index, name := range record.Images {
			recordDetail.Images = append(recordDetail.Images, QuestionBankImageInfo{Index: index, MIME: imageMIME(name)})
		}
		detail.Records = append(detail.Records, recordDetail)
	}
	return detail
}

func (a *App) GetQuestionBankSessionDetail(id string) (QuestionBankSessionDetail, error) {
	id = strings.TrimSpace(id)
	store, err := a.questionBankHistoryStore()
	if err != nil {
		return QuestionBankSessionDetail{}, err
	}
	if store == nil {
		return QuestionBankSessionDetail{}, fmt.Errorf("请先在设置中选择题库目录")
	}
	allowed, err := recentQuestionBankContains(store, id)
	if err != nil {
		return QuestionBankSessionDetail{}, err
	}
	if !allowed {
		return QuestionBankSessionDetail{}, fmt.Errorf("只能在界面查看最近 20 个分段；更早记录请从题库文件夹查看")
	}
	session, err := store.SnapshotByID(id)
	if err != nil {
		return QuestionBankSessionDetail{}, err
	}
	detail := buildQuestionBankSessionDetail(session)
	encoded, err := json.Marshal(detail)
	if err != nil {
		return QuestionBankSessionDetail{}, err
	}
	if len(encoded) > questionBankDetailMaxBytes {
		return QuestionBankSessionDetail{}, fmt.Errorf("该分段详情超过 8 MiB，请从题库文件夹查看完整记录")
	}
	return detail, nil
}

func (a *App) GetQuestionBankRecordImage(sessionID, recordID string, imageIndex int) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	recordID = strings.TrimSpace(recordID)
	store, err := a.questionBankHistoryStore()
	if err != nil {
		return "", err
	}
	if store == nil {
		return "", fmt.Errorf("请先在设置中选择题库目录")
	}
	allowed, err := recentQuestionBankContains(store, sessionID)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", fmt.Errorf("只能在界面查看最近 20 个分段；更早截图请从题库文件夹查看")
	}
	dataURL, err := store.ReadRecordImage(sessionID, recordID, imageIndex)
	if err != nil {
		return "", err
	}
	allowed, err = recentQuestionBankContains(store, sessionID)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", fmt.Errorf("该分段已移出最近 20 个记录")
	}
	return dataURL, nil
}
