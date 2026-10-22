package questionbank

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func automaticBank(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetDefaultCompany("示例单位"); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBeginAutomaticallyCreatesSegmentAndSplitIsolatesIt(t *testing.T) {
	s := automaticBank(t)
	firstRecord, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil || firstRecord == "" {
		t.Fatalf("自动建段失败: id=%q err=%v", firstRecord, err)
	}
	if err = s.Finish(firstRecord, "first", Completed); err != nil {
		t.Fatal(err)
	}
	first, _ := s.Snapshot()
	if first == nil || first.State != Active || first.Company != "示例单位" {
		t.Fatalf("自动段状态错误: %+v", first)
	}
	if err = s.Split(); err != nil {
		t.Fatal(err)
	}
	sealed, err := s.SnapshotByID(first.ID)
	if err != nil || sealed.State != Ended {
		t.Fatalf("旧段未封存: %+v err=%v", sealed, err)
	}
	secondRecord, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil || secondRecord == "" {
		t.Fatal(err)
	}
	second, _ := s.Snapshot()
	if second.ID == first.ID || len(second.Records) != 1 {
		t.Fatalf("新题串入旧段: first=%s second=%+v", first.ID, second)
	}
}

func TestDiscardCurrentIsSoftAndNextBeginStartsFresh(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	first, _ := s.Snapshot()
	manifest := filepath.Join(s.sessionDir(first.ID), "session.json")
	if err := s.DiscardCurrent(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("软清空删除了记录: %v", err)
	}
	discarded, err := s.SnapshotByID(first.ID)
	if err != nil || discarded.State != Discarded || discarded.DiscardedAt == nil {
		t.Fatalf("丢弃元数据错误: %+v err=%v", discarded, err)
	}
	id, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil || id == "" {
		t.Fatal(err)
	}
	current, _ := s.Snapshot()
	if current.ID == first.ID || len(current.Records) != 1 {
		t.Fatal("软清空后的记录仍进入旧段")
	}
}

func TestExportUsesArchiveDirectoryDirectlyAndSequentialSuffix(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	target, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.ExportSession(target, ExportOptions{Name: ""})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(first.Directory) != s.Root() {
		t.Fatalf("导出仍追加重复层级: %s", first.Directory)
	}
	if filepath.Base(first.Markdown) != "题库.md" {
		t.Fatalf("题库文件名错误: %s", first.Markdown)
	}
	base := filepath.Base(first.Directory)
	if !strings.Contains(base, "_示例单位") || strings.Contains(first.Directory, "按单位") {
		t.Fatalf("目录命名错误: %s", first.Directory)
	}
	second, err := s.ExportSession(target, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(second.Directory) != base+"_02" {
		t.Fatalf("冲突后缀错误: first=%s second=%s", base, filepath.Base(second.Directory))
	}
}

func TestExportDefaultsToQuestionBankNameAndGroupsCategoryThenType(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := add(t, s, Completed)
	target, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{body: `{"sourceRecordId":"` + id + `","questions":[{"title":"题目","stem":"题干","category":"计算机基础","type":"操作系统","options":[],"knowledgePoints":["进程"],"keyPoints":["核对状态"],"imageIndexes":[1],"reviewNote":"待核对"}]}`}
	if err = s.OrganizeSession(context.Background(), target, p, "model", nil); err != nil {
		t.Fatal(err)
	}
	result, err := s.ExportSession(target, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.Base(result.Directory), "_笔试题库") {
		t.Fatalf("默认名称错误: %s", result.Directory)
	}
	body, err := os.ReadFile(result.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Index(text, "## 计算机基础") < 0 || strings.Index(text, "### 操作系统") < strings.Index(text, "## 计算机基础") {
		t.Fatalf("未按大类和题型组织:\n%s", text)
	}
}

func TestOrganizeAndExportByTargetIDWhileNewSegmentIsCurrent(t *testing.T) {
	s := automaticBank(t)
	id := add(t, s, Completed)
	old, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	newRecord, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(newRecord, "new", Completed); err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{body: `{"sourceRecordId":"` + id + `","questions":[{"title":"旧题","stem":"旧题干","category":"职业能力","type":"逻辑判断","options":[],"knowledgePoints":[],"keyPoints":["核对"],"imageIndexes":[1],"reviewNote":"待核对"}]}`}
	if err = s.OrganizeSession(context.Background(), old, p, "model", nil); err != nil {
		t.Fatal(err)
	}
	oldSnapshot, err := s.SnapshotByID(old)
	if err != nil || len(oldSnapshot.Records[0].Questions) != 1 {
		t.Fatalf("目标段未更新: %+v err=%v", oldSnapshot, err)
	}
	current, _ := s.Snapshot()
	if current.ID == old || len(current.Records[0].Questions) != 0 {
		t.Fatal("整理结果串入当前新段")
	}
}

func TestLegacySessionCanBeListedAndOpened(t *testing.T) {
	archive := t.TempDir()
	legacyRoot := t.TempDir()
	legacy, err := NewStore(legacyRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err = legacy.Start("旧单位", "综合"); err != nil {
		t.Fatal(err)
	}
	add(t, legacy, Completed)
	if err = legacy.End(); err != nil {
		t.Fatal(err)
	}
	old, _ := legacy.Snapshot()

	migrated, err := NewStoreWithLegacy(archive, []string{legacy.sessionsDir()})
	if err != nil {
		t.Fatal(err)
	}
	list, err := migrated.List()
	if err != nil || len(list) != 1 || list[0].ID != old.ID {
		t.Fatalf("legacy 列表丢失: %+v err=%v", list, err)
	}
	if err = migrated.Open(old.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := migrated.Snapshot()
	encoded, _ := json.Marshal(snapshot)
	if snapshot.Company != "旧单位" || strings.Contains(string(encoded), "apiKey") {
		t.Fatalf("legacy 读取错误: %+v", snapshot)
	}
}

func TestFinalizedSegmentsRejectSoftDiscardAndDiscardedRejectOrganize(t *testing.T) {
	exported := automaticBank(t)
	add(t, exported, Completed)
	id, err := exported.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = exported.ExportSession(id, ExportOptions{}); err != nil {
		t.Fatal(err)
	}
	if err = exported.DiscardCurrent(); err == nil {
		t.Fatal("已导出的分段被软清空改写")
	}
	snapshot, _ := exported.Snapshot()
	if snapshot.State != Exported {
		t.Fatalf("已导出状态被改写: %s", snapshot.State)
	}

	discarded := automaticBank(t)
	add(t, discarded, Completed)
	if err = discarded.DiscardCurrent(); err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{body: `{}`}
	id = snapshotID(discarded)
	if err = discarded.OrganizeSession(context.Background(), id, p, "model", nil); err == nil {
		t.Fatal("已清空分段进入模型整理")
	}
	if p.calls != 0 {
		t.Fatalf("已清空分段不应调用模型，实际 %d 次", p.calls)
	}
}

func TestStructuredMarkdownNeutralizesModelLinksAndHTML(t *testing.T) {
	s := automaticBank(t)
	recordID := add(t, s, Completed)
	target, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{body: `{"sourceRecordId":"` + recordID + `","questions":[{"title":"外链题","stem":"<img src=x onerror=alert(1)> [外链](https://example.invalid) http://example.invalid","category":"计算机基础","type":"安全","options":["[选项](https://example.invalid)"],"knowledgePoints":["<script>alert(1)</script>"],"keyPoints":["查看 https://example.invalid"],"imageIndexes":[1],"reviewNote":"<b>待核对</b>"}]}`}
	if err = s.OrganizeSession(context.Background(), target, p, "model", nil); err != nil {
		t.Fatal(err)
	}
	result, err := s.ExportSession(target, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(result.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{"<img", "<script", "](https://", "http://example.invalid", "https://example.invalid"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("结构化模型内容保留了可执行 HTML 或外链 %q:\n%s", forbidden, text)
		}
	}
}

func TestBeginRetainsRecordWhenLazySegmentInitialSaveFails(t *testing.T) {
	root := t.TempDir()
	s, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(root, "_记录")
	if err = os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}

	recordID, beginErr := s.Begin([]string{shot(t, false)}, "model", "scene")
	if beginErr == nil {
		t.Fatal("懒建段首次落盘失败未返回错误")
	}
	if recordID == "" {
		t.Fatal("懒建段首次落盘失败后丢失了本题记录 ID")
	}
	snapshot, dirty := s.Snapshot()
	if !dirty || snapshot == nil || snapshot.State != Active || len(snapshot.Records) != 1 || snapshot.Records[0].ID != recordID {
		t.Fatalf("落盘失败后内存记录未保留: dirty=%v snapshot=%+v", dirty, snapshot)
	}
	if err = s.Finish(recordID, "answer", Completed); err == nil {
		t.Fatal("阻塞目录仍存在时 Finish 应报告保存失败")
	}
	if err = os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	if err = s.Flush(); err != nil {
		t.Fatalf("恢复目录后重试保存失败: %v", err)
	}
	persisted, err := s.SnapshotByID(snapshot.ID)
	if err != nil || len(persisted.Records) != 1 || persisted.Records[0].Answer != "answer" || persisted.Records[0].Status != Completed {
		t.Fatalf("重试后题目未完整落盘: %+v err=%v", persisted, err)
	}
	if _, err = os.Stat(filepath.Join(s.sessionDir(snapshot.ID), persisted.Records[0].Images[0])); err != nil {
		t.Fatalf("重试后截图缺失: %v", err)
	}
}

func TestDirtyFinalizedSegmentCannotBeReplacedByNextBegin(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	before, _ := s.Snapshot()
	manifest := filepath.Join(s.sessionDir(before.ID), "session.json")
	backup := manifest + ".bak"
	if err := os.Rename(manifest, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(manifest, 0700); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SealCurrent(); err == nil {
		t.Fatal("manifest 被目录占用时封存应失败")
	}
	afterSeal, dirty := s.Snapshot()
	if !dirty || afterSeal.ID != before.ID || afterSeal.State != Ended {
		t.Fatalf("封存失败后的内存状态错误: dirty=%v snapshot=%+v", dirty, afterSeal)
	}
	if recordID, err := s.Begin([]string{shot(t, false)}, "model", "scene"); err == nil || recordID != "" {
		t.Fatalf("未落盘封存段被下一题替换: id=%q err=%v", recordID, err)
	}
	preserved, stillDirty := s.Snapshot()
	if !stillDirty || preserved.ID != before.ID || preserved.State != Ended || len(preserved.Records) != 1 {
		t.Fatalf("下一题覆盖了未落盘封存段: dirty=%v snapshot=%+v", stillDirty, preserved)
	}

	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, manifest); err != nil {
		t.Fatal(err)
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("恢复 manifest 后重试保存失败: %v", err)
	}
	newRecord, err := s.Begin([]string{shot(t, false)}, "model", "scene")
	if err != nil || newRecord == "" {
		t.Fatalf("封存段保存成功后未创建新段: id=%q err=%v", newRecord, err)
	}
	current, _ := s.Snapshot()
	if current.ID == before.ID || current.State != Active || len(current.Records) != 1 {
		t.Fatalf("新段状态错误: %+v", current)
	}
}

func snapshotID(s *Store) string {
	snapshot, _ := s.Snapshot()
	return snapshot.ID
}

func TestNamedPaperUsesExamLayoutWithoutImages(t *testing.T) {
	s := automaticBank(t)
	recordID := add(t, s, Completed)
	target, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{body: `{"sourceRecordId":"` + recordID + `","questions":[{"title":"进程题","stem":"下列关于进程的说法正确的是？","category":"计算机基础","type":"操作系统","options":["A. 选项一","B. 选项二"],"referenceAnswer":"B","knowledgePoints":["进程管理"],"keyPoints":["比较进程与线程"],"imageIndexes":[1],"reviewNote":"待核对"}]}`}
	if err = s.OrganizeSession(context.Background(), target, p, "model", nil); err != nil {
		t.Fatal(err)
	}
	result, err := s.ExportSession(target, ExportOptions{
		Name:    "示例单位_操作系统",
		Company: "示例单位",
		Type:    "操作系统",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(result.Markdown) != "示例单位_操作系统.md" {
		t.Fatalf("组织稿文件名错误: %s", result.Markdown)
	}
	body, err := os.ReadFile(result.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, expected := range []string{"## 试题", "## 参考答案与解析", "第 1 题", "**参考答案：** B", "**来源：** [原始提交 1]"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("试卷缺少 %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "![") || strings.Contains(text, "assets/") {
		t.Fatalf("组织稿不应包含图片:\n%s", text)
	}
	raw, err := os.ReadFile(filepath.Join(result.Directory, "原始记录.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "![提交 1 截图 1](assets/") {
		t.Fatal("原始记录丢失截图")
	}
}

func TestRawOnlyExportOmitsOrganizedPaper(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	target, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.ExportSession(target, ExportOptions{Name: "仅原稿", RawOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Markdown != "" {
		t.Fatalf("仅原稿导出不应报告组织稿: %s", result.Markdown)
	}
	if _, err = os.Stat(filepath.Join(result.Directory, "仅原稿.md")); !os.IsNotExist(err) {
		t.Fatalf("仅原稿导出生成了误导性试卷: %v", err)
	}
	if _, err = os.Stat(filepath.Join(result.Directory, "原始记录.md")); err != nil {
		t.Fatalf("仅原稿导出缺少原始记录: %v", err)
	}
}

func TestOneOffExportDirectoryDoesNotMoveRecoveryRecords(t *testing.T) {
	s := automaticBank(t)
	add(t, s, Completed)
	target, err := s.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	recoveryManifest := filepath.Join(s.Root(), "_记录", target, "session.json")
	destination := t.TempDir()
	result, err := s.ExportSession(target, ExportOptions{
		Name:      "自定义位置",
		Directory: destination,
		RawOnly:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(result.Directory) != destination {
		t.Fatalf("未使用本次目录: %s", result.Directory)
	}
	if _, err = os.Stat(recoveryManifest); err != nil {
		t.Fatalf("一次性目录移动了恢复记录: %v", err)
	}
	if _, err = os.Stat(filepath.Join(destination, "_记录")); !os.IsNotExist(err) {
		t.Fatalf("一次性目录不应创建恢复记录根: %v", err)
	}
}

func TestRawAnswerMarkdownUsesAdaptiveFence(t *testing.T) {
	answer := "原答案 **A**\n```\n<img src=x onerror=alert(1)>\nhttps://example.com"
	got := rawAnswerMarkdown(answer)
	if !strings.HasPrefix(got, "````\n") || !strings.HasSuffix(got, "\n````\n\n") {
		t.Fatalf("原答案没有使用安全的自适应围栏: %q", got)
	}
	if !strings.Contains(got, answer) {
		t.Fatal("安全展示改变了原答案内容")
	}
}
