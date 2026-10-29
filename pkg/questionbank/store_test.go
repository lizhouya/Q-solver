package questionbank

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Q-Solver/pkg/llm"
)

func shot(t *testing.T, jpegImage bool) string {
	t.Helper()
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	mime := "image/png"
	var err error
	if jpegImage {
		mime = "image/jpeg"
		err = jpeg.Encode(&b, img, nil)
	} else {
		err = png.Encode(&b, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func bank(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Start("示例单位", "综合"); err != nil {
		t.Fatal(err)
	}
	return s
}
func add(t *testing.T, s *Store, status string) string {
	t.Helper()
	id, err := s.Begin([]string{shot(t, false), shot(t, true), shot(t, false)}, "model", "gen-solver")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Append(id, "不完整片段"); err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(id, "原答案 **A**", status); err != nil {
		t.Fatal(err)
	}
	return id
}
func setQuestionsForTest(t *testing.T, s *Store, recordID string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	record := findRecord(s.current, recordID)
	if record == nil {
		t.Fatal("测试记录不存在")
	}
	record.Questions = []Question{{Title: "测试题", Stem: "测试题干", Category: "测试", Type: "单选", ImageIndexes: []int{1}, ReviewNote: "待核对"}}
	if err := s.saveLocked(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteArchiveAndPortableExport(t *testing.T) {
	s := bank(t)
	for i := 0; i < 12; i++ {
		add(t, s, Completed)
	}
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	result, err := s.Export(ExportOptions{GroupBy: "company", Name: "示例单位", RawOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(result.Directory, "原始记录.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "原答案 **A**") != 12 || strings.Contains(string(raw), "不完整片段") {
		t.Fatal("final answer lost or overwritten")
	}
	files, _ := os.ReadDir(filepath.Join(result.Directory, "assets"))
	if len(files) != 36 {
		t.Fatalf("want all 36 images, got %d", len(files))
	}
	next, err := s.Export(ExportOptions{GroupBy: "company", Name: "示例单位", RawOnly: true})
	if err != nil || next.Directory == result.Directory {
		t.Fatal("export overwrote an earlier version")
	}
	moved := filepath.Join(t.TempDir(), "portable")
	if err = os.Rename(result.Directory, moved); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if _, err = os.Stat(filepath.Join(moved, "assets", f.Name())); err != nil {
			t.Fatal(err)
		}
	}
}
func TestLifecycleRecoveryAndLateCallbacks(t *testing.T) {
	s := bank(t)
	id, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if s.End() == nil {
		t.Fatal("closed a running request")
	}
	if err = s.Append(id, "部分答案"); err != nil {
		t.Fatal(err)
	}
	if err = s.Flush(); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot()
	recovered, err := NewStore(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err = recovered.Open(snap.ID); err != nil {
		t.Fatal(err)
	}
	rs, _ := recovered.Snapshot()
	if rs.State != Paused || rs.Records[0].Status != Interrupted || rs.Records[0].Answer != "部分答案" {
		t.Fatalf("recovery failed: %+v", rs)
	}
	if err = recovered.Resume(); err != nil {
		t.Fatal(err)
	}
	next := add(t, recovered, Cancelled)
	if err = recovered.Finish(id, "迟到答案", Completed); err != nil {
		t.Fatal(err)
	}
	rs, _ = recovered.Snapshot()
	if rs.Records[0].Answer != "部分答案" || rs.Records[1].ID != next || rs.Records[1].Status != Cancelled {
		t.Fatal("late callback changed another record")
	}
}
func TestDecodeImageRejectsPayloadAboveExportLimit(t *testing.T) {
	payload := make([]byte, 20*1024*1024+1)
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload)
	if _, _, err := decodeImage(dataURL); err == nil || !strings.Contains(err.Error(), "超过 20 MiB") {
		t.Fatalf("写入端接受了导出端无法读取的截图: %v", err)
	}
}

func TestInvalidPathsAndImages(t *testing.T) {
	s := bank(t)
	if _, err := s.Begin([]string{"data:image/png;base64,bm90IGFuIGltYWdl"}, "", ""); err == nil {
		t.Fatal("accepted invalid image")
	}
	if _, err := NewStore(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("silently created missing root")
	}
	if err := s.Open("../outside"); err == nil {
		t.Fatal("accepted traversal")
	}
	for _, name := range []string{"CON", "..", "A/B:C*?", "示例单位. ", ""} {
		clean := safeName(name)
		if clean == "" || strings.ContainsAny(clean, `/\:*?"<>|`) || strings.HasSuffix(clean, ".") || strings.EqualFold(clean, "CON") {
			t.Fatalf("unsafe name %q -> %q", name, clean)
		}
	}
}
func TestPersistenceFailureKeepsRecoverableRecord(t *testing.T) {
	s := bank(t)
	snap, _ := s.Snapshot()
	manifest := filepath.Join(s.sessionDir(snap.ID), "session.json")
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(manifest, 0700); err != nil {
		t.Fatal(err)
	}
	id, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err == nil || id == "" {
		t.Fatal("write failure must return error and stable record id")
	}
	if err = s.Finish(id, "可恢复答案", Completed); err == nil {
		t.Fatal("write failure hidden")
	}
	snap, dirty := s.Snapshot()
	if !dirty || snap.Records[0].Answer != "可恢复答案" {
		t.Fatal("lost unsaved result")
	}
	if err = os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if err = s.Flush(); err != nil {
		t.Fatal(err)
	}
	_, dirty = s.Snapshot()
	if dirty {
		t.Fatal("retry did not clear dirty state")
	}
}

type fakeProvider struct {
	calls    int
	body     string
	err      error
	messages []llm.Message
}

func (f *fakeProvider) GenerateContent(ctx context.Context, model string, ms []llm.Message) (llm.Message, error) {
	f.calls++
	f.messages = ms
	if ctx.Err() != nil {
		return llm.Message{}, ctx.Err()
	}
	return llm.NewAssistantMessage(f.body), f.err
}
func (f *fakeProvider) GenerateContentStream(context.Context, []llm.Message, llm.StreamCallback) (llm.Message, error) {
	return llm.Message{}, nil
}
func (f *fakeProvider) GetModels(context.Context) ([]string, error) { return nil, nil }
func (f *fakeProvider) TestChat(context.Context) error              { return nil }
func TestOrganizePreservesOriginalAndRetriesFailedOnly(t *testing.T) {
	s := bank(t)
	id := add(t, s, Completed)
	add(t, s, Failed)
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{body: `{"sourceRecordId":"` + id + `","metadataSuggestion":{"company":"示例单位","issuer":"测评机构","type":"逻辑判断","reviewNote":"截图可见，待核对"},"questions":[{"title":"题目","stem":"题干","category":"职业能力","type":"逻辑","options":["A"],"knowledgePoints":["推理"],"keyPoints":["核对条件"],"imageIndexes":[1,2],"reviewNote":"待核对"}]}`}
	if err := s.Organize(context.Background(), p, "model", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Organize(context.Background(), p, "model", nil); err != nil {
		t.Fatal(err)
	}
	if p.calls != 1 {
		t.Fatalf("successful record was retried: %d", p.calls)
	}
	snap, _ := s.Snapshot()
	if snap.Records[0].Answer != "原答案 **A**" || len(snap.Records[0].Questions) != 1 {
		t.Fatal("original answer changed")
	}
	if suggestion := snap.Records[0].MetadataSuggestion; suggestion == nil || suggestion.Company != "示例单位" || suggestion.Issuer != "测评机构" || suggestion.Type != "逻辑判断" {
		t.Fatalf("metadata suggestion was not persisted: %+v", suggestion)
	}
	data, _ := json.Marshal(p.messages)
	for _, v := range []string{"CandidateProfile", "ScenePrompt", "apiKey", "resumeContent"} {
		if strings.Contains(string(data), v) {
			t.Fatal("unrelated config in request")
		}
	}
}
func TestOrganizeRejectsBadOutputAndCancellation(t *testing.T) {
	for _, body := range []string{"not json", `{"sourceRecordId":"wrong","questions":[]}`} {
		s := bank(t)
		add(t, s, Completed)
		if err := s.End(); err != nil {
			t.Fatal(err)
		}
		p := &fakeProvider{body: body}
		if s.Organize(context.Background(), p, "model", nil) == nil {
			t.Fatal("invalid model output accepted")
		}
		snap, _ := s.Snapshot()
		if snap.Records[0].Answer != "原答案 **A**" {
			t.Fatal("original corrupted")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if !errors.Is(s.Organize(ctx, p, "model", nil), context.Canceled) {
			t.Fatal("cancellation ignored")
		}
		if _, err := s.Export(ExportOptions{GroupBy: "type", Name: "综合", RawOnly: true}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMissingRootIsNotRecreated(t *testing.T) {
	s := bank(t)
	add(t, s, Completed)
	root := s.Root()
	moved := root + "-moved"
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Rename(moved, root) })
	if err := s.End(); err == nil {
		t.Fatal("missing output root silently recreated")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("old output root recreated")
	}
}
func TestStructuredSchemaValidation(t *testing.T) {
	id := "0123456789abcdef01234567"
	q := Question{Title: "标题", Stem: "题干", Category: "编程与算法", Type: "算法", ImageIndexes: []int{1}, KnowledgePoints: []string{"排序"}, KeyPoints: []string{"核对复杂度"}}
	encode := func(qs []Question) string {
		b, _ := json.Marshal(struct {
			ID string     `json:"sourceRecordId"`
			Q  []Question `json:"questions"`
		}{id, qs})
		return string(b)
	}
	if got, err := parseQuestions(encode([]Question{q, q}), id, 3); err != nil || len(got) != 2 {
		t.Fatal("multi-question input lost")
	}
	for _, indices := range [][]int{{0}, {4}, {1, 1}, {}} {
		q.ImageIndexes = indices
		if _, err := parseQuestions(encode([]Question{q}), id, 3); err == nil {
			t.Fatal("invalid source indices accepted")
		}
	}
	q.ImageIndexes = []int{1}
	q.Stem = ""
	if _, err := parseQuestions(encode([]Question{q}), id, 3); err == nil {
		t.Fatal("missing stem accepted")
	}
}
func TestSessionsRemainIsolated(t *testing.T) {
	s := bank(t)
	add(t, s, Completed)
	first, _ := s.Snapshot()
	if err := s.Start("其他单位", "算法"); err == nil {
		t.Fatal("replaced active session")
	}
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	if err := s.Start("其他单位", "算法"); err != nil {
		t.Fatal(err)
	}
	add(t, s, Empty)
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	if err := s.Open(first.ID); err != nil {
		t.Fatal(err)
	}
	restored, _ := s.Snapshot()
	if len(restored.Records) != 1 || restored.Company != "示例单位" {
		t.Fatal("mixed sessions")
	}
	sessions, err := s.List()
	if err != nil || len(sessions) != 2 {
		t.Fatal("session listing lost history")
	}
}

func TestConcurrentCaptureAndSnapshot(t *testing.T) {
	s := bank(t)
	id, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() { err := s.Append(id, "x"); s.Snapshot(); done <- err }()
	}
	for i := 0; i < 20; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Finish(id, "完整答案", Completed); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot()
	if snap.Records[0].Answer != "完整答案" {
		t.Fatal("concurrent state lost final answer")
	}
}

func TestOutputRootMustBeExplicitAndOutsideGit(t *testing.T) {
	for _, path := range []string{"", ".", "relative/output"} {
		if _, err := NewStore(path); err == nil {
			t.Fatalf("accepted implicit root %q", path)
		}
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "输出")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(dir); err == nil {
		t.Fatal("accepted archive inside repository")
	}
}
