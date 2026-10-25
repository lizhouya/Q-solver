package questionbank

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacySessionJSONLeavesNewMetadataEmpty(t *testing.T) {
	for _, version := range []int{1, 2} {
		raw := `{"version":` + string(rune('0'+version)) + `,"id":"0123456789abcdef01234567","company":"旧单位","type":"综合","startedAt":"2026-10-23T00:00:00Z","state":"ended","records":[{"id":"abcdef0123456789abcdef01","createdAt":"2026-10-23T00:00:00Z","images":[],"model":"m","scene":"s","answer":"a","status":"completed"}]}`
		var session Session
		if err := json.Unmarshal([]byte(raw), &session); err != nil {
			t.Fatalf("v%d 旧记录读取失败: %v", version, err)
		}
		if session.Name != "" || session.Issuer != "" || session.NameCustomized || session.Records[0].MetadataSuggestion != nil {
			t.Fatalf("v%d 旧记录产生了伪元数据: %+v", version, session)
		}
	}
}

func TestParseOrganizedResultIncludesReviewedMetadataSuggestion(t *testing.T) {
	id := "0123456789abcdef01234567"
	body := `{"sourceRecordId":"` + id + `","metadataSuggestion":{"company":" 示例单位 ","issuer":"测评机构","type":"逻辑判断","reviewNote":"截图标题可见，仍需核对"},"questions":[{"title":"题目","stem":"题干","category":"行测","type":"逻辑判断","options":[],"knowledgePoints":[],"keyPoints":[],"imageIndexes":[1],"reviewNote":"待核对"}]}`
	questions, suggestion, err := parseOrganizedResult(body, id, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 1 || suggestion == nil || suggestion.Company != "示例单位" || suggestion.Issuer != "测评机构" || suggestion.Type != "逻辑判断" {
		t.Fatalf("整理候选丢失: questions=%+v suggestion=%+v", questions, suggestion)
	}
}

func TestParseOrganizedResultRejectsUnknownAndOversizedSuggestion(t *testing.T) {
	id := "0123456789abcdef01234567"
	question := `"questions":[{"title":"题目","stem":"题干","category":"行测","type":"逻辑","options":[],"knowledgePoints":[],"keyPoints":[],"imageIndexes":[1],"reviewNote":"待核对"}]`
	cases := []string{
		`{"sourceRecordId":"` + id + `","metadataSuggestion":{"company":"单位","unexpected":"x"},` + question + `}`,
		`{"sourceRecordId":"` + id + `","metadataSuggestion":{"company":"` + strings.Repeat("甲", 121) + `"},` + question + `}`,
		`{"sourceRecordId":"` + id + `","metadataSuggestion":{"reviewNote":"` + strings.Repeat("核", 2001) + `"},` + question + `}`,
	}
	for _, body := range cases {
		if _, _, err := parseOrganizedResult(body, id, 1); err == nil {
			t.Fatal("非法模型元数据候选被接受")
		}
	}
}

func TestReadRecordImageReturnsValidatedDataURLWithoutExposingPath(t *testing.T) {
	s := automaticBank(t)
	recordID := add(t, s, Completed)
	session, _ := s.Snapshot()
	got, err := s.ReadRecordImage(session.ID, recordID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") || strings.Contains(got, session.ID) || strings.Contains(got, "assets/") {
		t.Fatalf("图片返回值无效或泄露路径: %q", got[:min(len(got), 80)])
	}
	if _, err = s.ReadRecordImage(session.ID, recordID, 3); err == nil {
		t.Fatal("越界图片索引被接受")
	}
	if _, err = s.ReadRecordImage(session.ID, "../record", 0); err == nil {
		t.Fatal("非法记录 ID 被接受")
	}
}

func TestReadRecordImageRejectsExtensionFormatMismatch(t *testing.T) {
	s := automaticBank(t)
	recordID := add(t, s, Completed)
	session, _ := s.Snapshot()
	record := findRecord(session, recordID)
	path := filepath.Join(s.sessionDir(session.ID), filepath.FromSlash(record.Images[0]))
	var data bytes.Buffer
	if err := jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadRecordImage(session.ID, recordID, 0); err == nil || !strings.Contains(err.Error(), "实际格式") {
		t.Fatalf("扩展名与实际格式不一致未被拒绝: %v", err)
	}
}

func TestUpdateSessionMetadataPreservesRecordsAndRejectsDiscarded(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	id, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.SnapshotByID(id)
	if err = s.UpdateSessionMetadata(id, "示例单位_测评机构_逻辑", "示例单位", "测评机构", "逻辑", true); err != nil {
		t.Fatal(err)
	}
	after, _ := s.SnapshotByID(id)
	if after.Name != "示例单位_测评机构_逻辑" || after.Issuer != "测评机构" || !after.NameCustomized || len(after.Records) != len(before.Records) || after.Records[0].Answer != before.Records[0].Answer {
		t.Fatalf("元数据写回改变了记录或丢失字段: %+v", after)
	}

	discarded := automaticBank(t)
	add(t, discarded, Completed)
	discardedID := snapshotID(discarded)
	if err = discarded.DiscardCurrent(); err != nil {
		t.Fatal(err)
	}
	if err = discarded.UpdateSessionMetadata(discardedID, "名称", "单位", "出题方", "题型", true); err == nil {
		t.Fatal("已清空分段允许写入权威元数据")
	}
}

func TestFailedOrganizeDoesNotOverwriteExistingSuggestion(t *testing.T) {
	record := Record{MetadataSuggestion: &SourceMetadataSuggestion{Company: "原建议", Issuer: "原出题方"}, OrganizeError: ""}
	applyOrganizeResult(&record, nil, nil, os.ErrPermission)
	if record.MetadataSuggestion == nil || record.MetadataSuggestion.Company != "原建议" || record.MetadataSuggestion.Issuer != "原出题方" {
		t.Fatalf("失败整理覆盖了旧建议: %+v", record.MetadataSuggestion)
	}
	if record.OrganizeError == "" {
		t.Fatal("失败整理没有记录安全错误")
	}
}

func setRecordImageName(t *testing.T, s *Store, recordID, name string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	record := findRecord(s.current, recordID)
	if record == nil {
		t.Fatal("测试记录不存在")
	}
	record.Images[0] = name
}

func syntheticPNG(width, height uint32) []byte {
	var output bytes.Buffer
	output.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	writeChunk := func(kind string, payload []byte) {
		_ = binary.Write(&output, binary.BigEndian, uint32(len(payload)))
		output.WriteString(kind)
		output.Write(payload)
		checksum := crc32.NewIEEE()
		_, _ = checksum.Write([]byte(kind))
		_, _ = checksum.Write(payload)
		_ = binary.Write(&output, binary.BigEndian, checksum.Sum32())
	}
	header := make([]byte, 13)
	binary.BigEndian.PutUint32(header[0:4], width)
	binary.BigEndian.PutUint32(header[4:8], height)
	header[8] = 8
	header[9] = 2
	writeChunk("IHDR", header)
	writeChunk("IEND", nil)
	return output.Bytes()
}

func TestReadRecordImageRejectsUnsafeManifestPaths(t *testing.T) {
	cases := []string{"../outside.png", "C:/outside.png", `assets\outside.png`, "assets/C:outside.png"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			s := automaticBank(t)
			recordID := add(t, s, Completed)
			setRecordImageName(t, s, recordID, name)
			session, _ := s.Snapshot()
			if _, err := s.ReadRecordImage(session.ID, recordID, 0); err == nil {
				t.Fatalf("不安全归档路径被接受: %q", name)
			}
		})
	}
}

func TestReadRecordImageRejectsPersistedManifestTraversal(t *testing.T) {
	s := automaticBank(t)
	recordID := add(t, s, Completed)
	id, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Begin([]string{shot(t, false)}, "model", "scene"); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(s.sessionDir(id), "session.json")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var persisted Session
	if err = json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	persisted.Records[0].Images[0] = "../outside.png"
	if err = atomicJSON(manifest, &persisted); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadRecordImage(id, recordID, 0); err == nil {
		t.Fatal("持久化清单中的路径穿越被接受")
	}
}

func TestReadRecordImageRejectsSymlinkAndNonRegularFile(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		s := automaticBank(t)
		recordID := add(t, s, Completed)
		session, _ := s.Snapshot()
		record := findRecord(session, recordID)
		root := filepath.Join(s.sessionDir(session.ID), "assets")
		link := filepath.Join(root, "linked.png")
		target := filepath.Join(s.sessionDir(session.ID), filepath.FromSlash(record.Images[0]))
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("当前 Windows 环境不能创建符号链接: %v", err)
		}
		setRecordImageName(t, s, recordID, "assets/linked.png")
		if _, err := s.ReadRecordImage(session.ID, recordID, 0); err == nil || !strings.Contains(err.Error(), "符号链接") {
			t.Fatalf("符号链接图片未被拒绝: %v", err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		s := automaticBank(t)
		recordID := add(t, s, Completed)
		session, _ := s.Snapshot()
		path := filepath.Join(s.sessionDir(session.ID), "assets", "directory.png")
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		setRecordImageName(t, s, recordID, "assets/directory.png")
		if _, err := s.ReadRecordImage(session.ID, recordID, 0); err == nil || !strings.Contains(err.Error(), "不是普通文件") {
			t.Fatalf("非普通图片未被拒绝: %v", err)
		}
	})
}

func TestReadRecordImageRejectsFileAndPixelLimits(t *testing.T) {
	t.Run("over 20 MiB", func(t *testing.T) {
		s := automaticBank(t)
		recordID := add(t, s, Completed)
		session, _ := s.Snapshot()
		record := findRecord(session, recordID)
		path := filepath.Join(s.sessionDir(session.ID), filepath.FromSlash(record.Images[0]))
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if err = file.Truncate(20*1024*1024 + 1); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err = s.ReadRecordImage(session.ID, recordID, 0); err == nil || !strings.Contains(err.Error(), "过大") {
			t.Fatalf("超限图片未被拒绝: %v", err)
		}
	})

	t.Run("over 80 million pixels", func(t *testing.T) {
		s := automaticBank(t)
		recordID := add(t, s, Completed)
		session, _ := s.Snapshot()
		record := findRecord(session, recordID)
		path := filepath.Join(s.sessionDir(session.ID), filepath.FromSlash(record.Images[0]))
		if err := os.WriteFile(path, syntheticPNG(10000, 8001), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ReadRecordImage(session.ID, recordID, 0); err == nil || !strings.Contains(err.Error(), "尺寸无效") {
			t.Fatalf("超像素图片未被拒绝: %v", err)
		}
	})
}
