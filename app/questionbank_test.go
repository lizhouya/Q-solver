package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/questionbank"
	"Q-Solver/pkg/task"
)

func archiveApp(t *testing.T) (*App, *questionbank.Store, string) {
	t.Helper()
	store, err := questionbank.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = store.SetDefaultCompany("单位"); err != nil {
		t.Fatal(err)
	}
	var imageBytes bytes.Buffer
	if err = png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	imageURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes.Bytes())
	a := &App{questionBank: &questionBankService{store: store}, taskManager: task.NewTaskCoordinator()}
	return a, store, imageURL
}
func TestQuestionBankCapturePrecedesOutputRouting(t *testing.T) {
	for _, mode := range []string{"desktop", "mobile", "dual"} {
		t.Run(mode, func(t *testing.T) {
			a, store, img := archiveApp(t)
			emit, finish := a.captureQuestion(context.Background(), config.Config{Model: "test", DomainId: "gen-solver", OutputMode: mode}, []string{img, img, img}, func(event string, data ...interface{}) {
				if event == "solution" {
					snap, dirty := store.Snapshot()
					if dirty || snap.Records[0].Answer != "complete answer" {
						t.Fatal("routing happened before archival")
					}
					route := decideRoute(mode, true)
					if mode == "mobile" && route.needDesktop {
						t.Fatal("mobile route unexpectedly needs desktop")
					}
				}
			})
			emit("solution-stream-chunk", "partial")
			emit("solution", "complete answer")
			finish()
			s, dirty := store.Snapshot()
			if dirty || len(s.Records) != 1 || len(s.Records[0].Images) != 3 || s.Records[0].Answer != "complete answer" {
				t.Fatal("archive depends on desktop/mobile events")
			}
			if a.GetQuestionBankStatus().Busy {
				t.Fatal("capture still busy")
			}
		})
	}
}
func TestQuestionBankCancelledRecordCannotOverwriteNext(t *testing.T) {
	a, store, img := archiveApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	emit, finish := a.captureQuestion(ctx, config.Config{}, []string{img}, a.EmitEvent)
	emit("solution-stream-chunk", "partial")
	cancel()
	emit("solution-error", "sensitive upstream error must not be saved")
	finish()
	next, done := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "late old result")
	next("solution", "new result")
	done()
	s, _ := store.Snapshot()
	if s.Records[0].Status != questionbank.Cancelled || s.Records[0].Answer != "partial" || s.Records[1].Answer != "new result" {
		t.Fatal("record association lost")
	}
}
func TestQuestionBankShutdownPersistsActiveSession(t *testing.T) {
	a, store, img := archiveApp(t)
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution-stream-chunk", "partial")
	finish()
	a.shutdownQuestionBank()
	s, dirty := store.Snapshot()
	if dirty || s.Records[0].Status != questionbank.Interrupted {
		t.Fatal("shutdown did not preserve interrupted record")
	}
}

func TestQuestionBankPreferencesIgnoreLegacyCompany(t *testing.T) {
	got := preferencesForUI(questionBankPreferencesFile{
		Version:      2,
		ArchiveDir:   filepath.Join("C:", "output"),
		Company:      "旧单位不应继续作为默认值",
		AutoOrganize: true,
	})
	if got.Company != "" || !got.AutoOrganize {
		t.Fatalf("长期题库偏好仍暴露旧单位: %+v", got)
	}
}

func TestNormalizeLegacyQuestionBankPreferences(t *testing.T) {
	root := filepath.Join("C:", "output")
	got := normalizeLegacyPreferences(questionBankPreferencesFile{Version: 1, Root: root})
	if got.Version != 2 || got.ArchiveDir != filepath.Join(root, "求职笔试", "题库") {
		t.Fatalf("v1 偏好迁移错误: %+v", got)
	}
	if len(got.LegacyRoots) != 1 || got.LegacyRoots[0] != filepath.Join(root, "求职笔试", "会话记录") {
		t.Fatalf("legacy 记录目录丢失: %+v", got.LegacyRoots)
	}

	archive := filepath.Join(root, "求职笔试", "题库")
	direct := normalizeLegacyPreferences(questionBankPreferencesFile{Version: 1, Root: archive})
	if direct.ArchiveDir != archive {
		t.Fatalf("已是题库目录时被重复追加: %+v", direct)
	}
}

func TestExportCurrentQuestionBankSegmentSealsAndExportsOnce(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root(), Company: "测试单位"}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()

	if err := a.ExportCurrentQuestionBankSegment(); err != nil {
		t.Fatal(err)
	}
	a.questionBank.workers.Wait()
	status := a.GetQuestionBankStatus()
	if status.Organizing || status.Session == nil || status.Session.State != questionbank.Exported {
		t.Fatalf("导出后状态错误: %+v", status)
	}
	if status.LastExport.Directory == "" || status.LastExport.Markdown != "" || filepath.Base(status.LastExport.RawMarkdown) != "原始记录.md" {
		t.Fatalf("关闭模型整理时应只导出原始记录: %+v", status.LastExport)
	}
	if err := a.ExportCurrentQuestionBankSegment(); err == nil {
		t.Fatal("重复点击导出了已经封存的当前段")
	}

	next, done := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	next("solution", "next")
	done()
	current, _ := store.Snapshot()
	if current.ID == status.Session.ID || current.State != questionbank.Active || len(current.Records) != 1 {
		t.Fatalf("导出后下一题未进入新段: %+v", current)
	}
}

func TestExportQuestionBankSessionAllowsExplicitReexport(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root(), Company: "测试单位"}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()

	if err := a.ExportCurrentQuestionBankSegment(); err != nil {
		t.Fatal(err)
	}
	a.questionBank.workers.Wait()
	first := a.GetQuestionBankStatus()
	if first.Session == nil || first.Session.State != questionbank.Exported || first.LastExport.Directory == "" {
		t.Fatalf("首次导出状态错误: %+v", first)
	}
	if err := a.ExportQuestionBankSession(first.Session.ID); err != nil {
		t.Fatalf("显式历史重导出不可达: %v", err)
	}
	a.questionBank.workers.Wait()
	second := a.GetQuestionBankStatus()
	if second.LastExport.Directory == "" || second.LastExport.Directory == first.LastExport.Directory {
		t.Fatalf("显式重导出未生成新版本: first=%+v second=%+v", first.LastExport, second.LastExport)
	}
	if err := a.ExportCurrentQuestionBankSegment(); err == nil {
		t.Fatal("普通一键按钮不应重复导出已导出段")
	}
}

func TestAsyncExportKeepsResultWhenStatePersistenceFails(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root(), Company: "测试单位"}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()
	targetID, err := store.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(store.Root(), "_记录", targetID, "session.json")
	backup := manifest + ".bak"
	if err = os.Rename(manifest, backup); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(manifest, 0700); err != nil {
		t.Fatal(err)
	}

	a.questionBank.mu.Lock()
	preferences, cfg, configErr := a.questionBankExportConfigLocked()
	if configErr != nil {
		a.questionBank.mu.Unlock()
		t.Fatal(configErr)
	}
	a.startQuestionBankExportLocked(targetID, preferences, cfg)
	a.questionBank.mu.Unlock()
	a.questionBank.workers.Wait()

	status := a.GetQuestionBankStatus()
	if status.LastExport.Directory == "" {
		t.Fatalf("文件已经生成时丢失了导出结果: %+v", status)
	}
	if _, err = os.Stat(status.LastExport.RawMarkdown); err != nil {
		t.Fatalf("报告的原始记录不存在: %v", err)
	}
	if !strings.Contains(status.Message, "状态保存失败") {
		t.Fatalf("未准确区分文件生成与状态保存失败: %q", status.Message)
	}
}

func TestQuestionBankExportDraftUsesLocalMetadataWithoutModel(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root(), AutoOrganize: true}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()

	draft, err := a.GetQuestionBankExportDraft("")
	if err != nil {
		t.Fatal(err)
	}
	if draft.SessionID == "" || draft.Count != 1 || draft.Directory != store.Root() || !draft.AutoOrganize {
		t.Fatalf("导出草稿错误: %+v", draft)
	}
	if draft.Company != "单位" || draft.Type != "综合笔试" || !strings.Contains(draft.Name, "单位_综合笔试") {
		t.Fatalf("本地建议名称错误: %+v", draft)
	}
}

func TestQuestionBankExportOperationTracksLifecycle(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()
	targetID, err := store.SealCurrent()
	if err != nil {
		t.Fatal(err)
	}

	a.questionBank.mu.Lock()
	a.startQuestionBankExportWithOptionsLocked(targetID, false, questionbank.ExportOptions{RawOnly: true}, config.Config{})
	if !a.questionBank.organizing || a.questionBank.operation != "exporting" {
		a.questionBank.mu.Unlock()
		t.Fatalf("本地导出未暴露 exporting 状态: organizing=%v operation=%q", a.questionBank.organizing, a.questionBank.operation)
	}
	a.questionBank.mu.Unlock()
	a.questionBank.workers.Wait()

	status := a.GetQuestionBankStatus()
	if status.Organizing || status.Operation != "" {
		t.Fatalf("本地导出结束后状态未清理: %+v", status)
	}
}

func TestQuestionBankExportRequestUsesOneOffOptions(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root(), AutoOrganize: true}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()
	current, _ := store.Snapshot()
	destination := t.TempDir()

	err := a.ExportQuestionBankSegment(QuestionBankExportRequest{
		SessionID:    current.ID,
		Company:      "本次单位",
		Type:         "行测",
		Name:         "本次单位_行测",
		Directory:    destination,
		AutoOrganize: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	a.questionBank.workers.Wait()
	status := a.GetQuestionBankStatus()
	if filepath.Dir(status.LastExport.Directory) != destination {
		t.Fatalf("本次目录未生效: %+v", status.LastExport)
	}
	if status.LastExport.Markdown != "" {
		t.Fatalf("显式关闭整理后仍生成组织稿: %+v", status.LastExport)
	}
	if !strings.Contains(filepath.Base(status.LastExport.Directory), "本次单位_行测") {
		t.Fatalf("本次名称未生效: %s", status.LastExport.Directory)
	}
}

func TestQuestionBankExportRequestValidationDoesNotSealCurrent(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root()}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()
	before, _ := store.Snapshot()

	err := a.ExportQuestionBankSegment(QuestionBankExportRequest{
		SessionID: before.ID,
		Type:      "综合笔试",
		Name:      "无效目录",
		Directory: "relative-output",
	})
	if err == nil {
		t.Fatal("相对导出目录未被拒绝")
	}
	after, _ := store.Snapshot()
	if after.ID != before.ID || after.State != questionbank.Active {
		t.Fatalf("无效请求提前封存了当前段: before=%+v after=%+v", before, after)
	}
}

func TestQuestionBankExportDraftUsesStableNumberWithoutCompany(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	if err := store.SetDefaultCompany(""); err != nil {
		t.Fatal(err)
	}
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root()}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "first")
	finish()
	first, err := a.GetQuestionBankExportDraft("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.Name, "_01_综合笔试") || first.Company != "" {
		t.Fatalf("首个无单位编号错误: %+v", first)
	}
	if first.NumberedPrefix == "" || first.Name != first.NumberedPrefix+"_综合笔试" {
		t.Fatalf("首个无单位草稿未返回稳定编号前缀: %+v", first)
	}
	if err = a.SplitQuestionBankSegment(); err != nil {
		t.Fatal(err)
	}
	emit, finish = a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "second")
	finish()
	second, err := a.GetQuestionBankExportDraft("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second.Name, "_02_综合笔试") {
		t.Fatalf("第二个无单位编号错误: %+v", second)
	}
	if second.NumberedPrefix == "" || second.NumberedPrefix == first.NumberedPrefix || second.Name != second.NumberedPrefix+"_综合笔试" {
		t.Fatalf("第二个无单位草稿编号前缀错误: first=%+v second=%+v", first, second)
	}
	again, err := a.GetQuestionBankExportDraft(first.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Name != first.Name {
		t.Fatalf("同一分段默认名称不稳定: first=%q again=%q", first.Name, again.Name)
	}
}

func TestQuestionBankExportRequestDoesNotPersistOneOffDirectory(t *testing.T) {
	a, store, img := archiveApp(t)
	a.configManager = config.NewConfigManager()
	a.questionBank.preferences = QuestionBankPreferences{Version: 2, ArchiveDir: store.Root()}
	emit, finish := a.captureQuestion(context.Background(), config.Config{}, []string{img}, a.EmitEvent)
	emit("solution", "answer")
	finish()
	current, _ := store.Snapshot()
	destination := t.TempDir()

	if err := a.ExportQuestionBankSegment(QuestionBankExportRequest{
		SessionID: current.ID,
		Type:      "综合笔试",
		Name:      "一次性目录",
		Directory: destination,
	}); err != nil {
		t.Fatal(err)
	}
	a.questionBank.workers.Wait()
	preferences := a.GetQuestionBankPreferences()
	if preferences.ArchiveDir != store.Root() {
		t.Fatalf("本次目录污染了长期偏好: %+v", preferences)
	}
}
