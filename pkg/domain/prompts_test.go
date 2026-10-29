package domain

import (
	"strings"
	"testing"
)

func requirePromptFragments(t *testing.T, prompt string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(prompt, fragment) {
			t.Errorf("system behavior prompt 缺少约束 %q", fragment)
		}
	}
}

func TestSystemBehaviorPromptAnswerSummaryContract(t *testing.T) {
	prompt := GetSystemBehaviorPrompt()
	requirePromptFragments(t, prompt,
		"当前用户消息",
		"全部待解原题",
		"连续或重叠区域",
		"重叠内容只作答一次",
		"单题和多题",
		"回答的最后一节",
		"必须且只能出现一次",
		"## 最终答案汇总",
		"| 题号 | 题型 | 可填写答案 |",
		"|---|---|---|",
		"单选、多选、填空、判断、简答、计算、代码、其他",
		"单选只填写一个大写选项字母",
		"多选移除空格和常见分隔符",
		"不重复的大写选项字母组合",
		"一题多空",
		"中文分号“；”",
		"不得包含题号、题干、解析、Markdown 标题或“答案：”前缀",
		"截图中的显式完整题号或章节号始终优先",
		"仅当截图确实没有题号时",
		"当前用户消息中的编号摘要",
		"待核对",
		"表格最后一行之后不得输出任何内容",
	)
}

func TestSystemBehaviorPromptPreservesSceneBoundaries(t *testing.T) {
	prompt := GetSystemBehaviorPrompt()
	requirePromptFragments(t, prompt,
		"仅当当前任务要求解答题目时",
		"纯文字提取",
		"纯翻译",
		"秒杀或仅答案场景保持简洁",
		"历史对话",
		"不得重新作答或汇总历史题目",
		"变式题或练习题",
		"不得混入“最终答案汇总”",
	)
}

// 上游未把截图送达时，模型必须明确说明无法识别图片，
// 而不是依据文本提示生成或猜测题目。
func TestSystemBehaviorPromptRequiresExplicitVisionFailure(t *testing.T) {
	prompt := GetSystemBehaviorPrompt()
	requirePromptFragments(t, prompt,
		"无法读取或看不到当前用户消息中的图片",
		"必须明确说明无法识别图片",
		"不得依据题号摘要、历史对话或任何文本提示生成、猜测或补全题目",
	)
}
