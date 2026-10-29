package questionbank

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func validOrganizedBody(id string) string {
	return `{"sourceRecordId":"` + id + `","questions":[{"title":"题目","stem":"题干","category":"行测","type":"逻辑判断","options":[],"knowledgePoints":[],"keyPoints":[],"imageIndexes":[1],"reviewNote":"待核对"}]}`
}

func TestParseOrganizedResultNormalizesOnlyWholeJSONFence(t *testing.T) {
	id := "0123456789abcdef01234567"
	body := validOrganizedBody(id)
	accepted := []string{
		body,
		"  \ufeff" + body + "\n",
		"```json\n" + body + "\n```",
		"```\n" + body + "\n```",
	}
	for _, input := range accepted {
		questions, _, err := parseOrganizedResult(input, id, 1)
		if err != nil || len(questions) != 1 {
			t.Fatalf("完整 JSON 或单一围栏未被接受: %v", err)
		}
	}
	for _, input := range []string{
		"说明：" + body,
		"<think>分析</think>" + body,
		"```json\n" + body + "\n```\n补充",
		body + body,
	} {
		if _, _, err := parseOrganizedResult(input, id, 1); err == nil {
			t.Fatal("带解释、思考或多个对象的响应被错误接受")
		}
	}
}

func TestOrganizeFailureClassification(t *testing.T) {
	id := "0123456789abcdef01234567"
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "empty", body: " \n", code: organizeCodeEmptyResponse},
		{name: "too_large", body: strings.Repeat("x", 512*1024+1), code: organizeCodeResponseLarge},
		{name: "invalid_json", body: "not json", code: organizeCodeInvalidJSON},
		{name: "extra", body: validOrganizedBody(id) + " {}", code: organizeCodeExtraContent},
		{name: "source", body: validOrganizedBody("abcdef0123456789abcdef01"), code: organizeCodeSourceMismatch},
		{name: "schema", body: `{"sourceRecordId":"` + id + `","unexpected":true,"questions":[]}`, code: organizeCodeSchemaInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parseOrganizedResult(test.body, id, 1)
			var failure *organizeFailureError
			if !errors.As(err, &failure) || failure.code != test.code {
				t.Fatalf("失败分类错误: err=%v code=%v want=%s", err, failure, test.code)
			}
		})
	}
}

func TestSummarizeOrganizeState(t *testing.T) {
	question := Question{Title: "题目"}
	failure := &OrganizeFailure{Code: organizeCodeInvalidJSON}
	tests := []struct {
		name                                            string
		records                                         []Record
		outcome                                         string
		eligible, organized, questions, pending, failed int
	}{
		{name: "no_eligible", records: []Record{{Status: Failed}}, outcome: OrganizeOutcomeNoEligible},
		{name: "pending", records: []Record{{Status: Completed}}, outcome: OrganizeOutcomePending, eligible: 1, pending: 1},
		{name: "complete", records: []Record{{Status: Completed, Questions: []Question{question, question}}}, outcome: OrganizeOutcomeComplete, eligible: 1, organized: 1, questions: 2},
		{name: "partial", records: []Record{{Status: Completed, Questions: []Question{question}}, {Status: Completed, OrganizeFailure: failure}}, outcome: OrganizeOutcomePartial, eligible: 2, organized: 1, questions: 1, failed: 1},
		{name: "all_failed", records: []Record{{Status: Completed, OrganizeFailure: failure}, {Status: Completed, OrganizeError: "旧失败"}}, outcome: OrganizeOutcomeAllFailed, eligible: 2, failed: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SummarizeOrganizeState(&Session{Records: test.records})
			if got.Outcome != test.outcome || got.EligibleRecords != test.eligible || got.OrganizedRecords != test.organized || got.StructuredQuestions != test.questions || got.PendingRecords != test.pending || got.FailedRecords != test.failed {
				t.Fatalf("汇总错误: %+v", got)
			}
		})
	}
}

func TestRetryFailedSessionTouchesOnlyFailures(t *testing.T) {
	s := bank(t)
	successID := add(t, s, Completed)
	failedID := add(t, s, Completed)
	pendingID := add(t, s, Completed)
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	findRecord(s.current, successID).Questions = []Question{{Title: "既有结果"}}
	findRecord(s.current, failedID).OrganizeError = "旧版失败"
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		t.Fatal(err)
	}
	s.mu.Unlock()

	provider := &fakeProvider{body: validOrganizedBody(failedID)}
	if err := s.RetryFailedSession(context.Background(), s.current.ID, provider, "safe-model", nil); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("失败重试调用次数错误: %d", provider.calls)
	}
	snapshot, _ := s.Snapshot()
	if len(findRecord(snapshot, successID).Questions) != 1 || findRecord(snapshot, successID).Questions[0].Title != "既有结果" {
		t.Fatal("既有成功结果被重试或覆盖")
	}
	retried := findRecord(snapshot, failedID)
	if len(retried.Questions) != 1 || retried.OrganizeError != "" || retried.OrganizeFailure != nil {
		t.Fatalf("失败项未恢复或错误未清理: %+v", retried)
	}
	if pending := findRecord(snapshot, pendingID); len(pending.Questions) != 0 || pending.OrganizeError != "" || pending.OrganizeFailure != nil {
		t.Fatal("纯 pending 项被失败重试顺带处理")
	}
	if err := s.RetryFailedSession(context.Background(), snapshot.ID, provider, "safe-model", nil); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatal("已恢复项被再次重试")
	}
}

func TestOrganizePersistsSafeFailureWithoutProviderDetails(t *testing.T) {
	s := bank(t)
	add(t, s, Completed)
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{err: errors.New("token=secret https://private.invalid C:\\private\\response.json raw-body")}
	if err := s.Organize(context.Background(), provider, "model-name", nil); err == nil {
		t.Fatal("模型请求失败未上报")
	}
	snapshot, _ := s.Snapshot()
	record := snapshot.Records[0]
	if record.OrganizeFailure == nil || record.OrganizeFailure.Code != organizeCodeRequestFailed || record.OrganizeFailure.Stage != "request" || !record.OrganizeFailure.ProviderCalled || record.OrganizeFailure.Model != "model-name" {
		t.Fatalf("安全失败分类未持久化: %+v", record.OrganizeFailure)
	}
	serialized := record.OrganizeError
	if strings.Contains(serialized, "secret") || strings.Contains(serialized, "private.invalid") || strings.Contains(serialized, "private\\response") || strings.Contains(serialized, "raw-body") {
		t.Fatalf("失败文案泄露 provider 细节: %q", serialized)
	}
}

func TestExportRejectsZeroQuestionOrganizedDraftBeforeCreatingDirectory(t *testing.T) {
	s := bank(t)
	add(t, s, Completed)
	if err := s.End(); err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{body: "not json"}
	if err := s.Organize(context.Background(), provider, "model", nil); err == nil {
		t.Fatal("测试前置整理失败未上报")
	}
	before, err := os.ReadDir(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Export(ExportOptions{Name: "零题整理稿"}); err == nil {
		t.Fatal("零题整理稿被导出")
	}
	after, err := os.ReadDir(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("拒绝导出后仍创建了目录: before=%d after=%d", len(before), len(after))
	}
	result, err := s.Export(ExportOptions{Name: "仅原稿", RawOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Markdown != "" || result.RawMarkdown == "" {
		t.Fatalf("仅原稿导出结果错误: %+v", result)
	}
}
