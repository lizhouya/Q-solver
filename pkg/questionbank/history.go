package questionbank

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

func (s *Store) ReadRecordImage(sessionID, recordID string, imageIndex int) (string, error) {
	if !validID(sessionID) || !validID(recordID) {
		return "", fmt.Errorf("分段或记录 ID 无效")
	}

	s.mu.Lock()
	var session *Session
	var base string
	var err error
	if s.current != nil && s.current.ID == sessionID {
		session = cloneSession(s.current)
		base = s.currentBase
		if base == "" {
			base = s.recordsRoot
		}
	} else {
		session, base, err = s.readSession(sessionID)
	}
	s.mu.Unlock()
	if err != nil {
		return "", err
	}

	record := findRecord(session, recordID)
	if record == nil {
		return "", fmt.Errorf("题目记录不存在")
	}
	if imageIndex < 0 || imageIndex >= len(record.Images) {
		return "", fmt.Errorf("截图序号无效")
	}
	name := record.Images[imageIndex]
	ext := strings.ToLower(filepath.Ext(name))
	var mime string
	switch ext {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	default:
		return "", fmt.Errorf("仅支持 PNG/JPEG 归档截图")
	}
	path, err := safeChild(filepath.Join(base, sessionID), name)
	if err != nil {
		return "", err
	}
	data, err := readLimited(path, 20*1024*1024)
	if err != nil {
		return "", err
	}
	dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	if _, actualExt, decodeErr := decodeImage(dataURL); decodeErr != nil {
		return "", decodeErr
	} else if actualExt != ext && !(actualExt == ".jpg" && ext == ".jpeg") {
		return "", fmt.Errorf("截图实际格式与扩展名不一致")
	}
	return dataURL, nil
}

func (s *Store) UpdateSessionMetadata(id, name, company, issuer, kind string, nameCustomized bool) error {
	if !validID(id) {
		return fmt.Errorf("分段 ID 无效")
	}
	name = strings.TrimSpace(name)
	company = strings.TrimSpace(company)
	issuer = strings.TrimSpace(issuer)
	kind = strings.TrimSpace(kind)
	if name == "" {
		return fmt.Errorf("题库名称不能为空")
	}
	if len([]rune(name)) > 120 || len([]rune(company)) > 120 || len([]rune(issuer)) > 120 || len([]rune(kind)) > 120 {
		return fmt.Errorf("题库名称、单位、出题方和题型最多 120 字")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureWriteAllowedLocked(); err != nil {
		return err
	}
	var session *Session
	var base string
	var err error
	isCurrent := s.current != nil && s.current.ID == id
	if isCurrent {
		if err = s.saveLocked(); err != nil {
			return err
		}
		session = cloneSession(s.current)
		base = s.currentBase
	} else {
		session, base, err = s.readSession(id)
		if err != nil {
			return err
		}
	}
	if session.State != Ended && session.State != Exported {
		return fmt.Errorf("只有已结束或已导出的分段可以更新题库元数据")
	}

	session.Name = name
	session.Company = company
	session.Issuer = issuer
	session.Type = kind
	session.NameCustomized = nameCustomized
	if err = s.saveSessionLocked(session, base, nil); err != nil {
		return fmt.Errorf("题库元数据未能保存: %w", err)
	}
	if isCurrent {
		s.current.Name = session.Name
		s.current.Company = session.Company
		s.current.Issuer = session.Issuer
		s.current.Type = session.Type
		s.current.NameCustomized = session.NameCustomized
	}
	return nil
}
