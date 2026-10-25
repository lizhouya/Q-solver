<template>
  <div class="action-bar">
    <div class="action-left">
      <button class="action-btn" :disabled="!numberedCopyText || copying" @click="handleCopyNumberedAnswers"
        title="复制本轮实际题号与答案，每题一行，不重新编号">
        <Icon :name="numberedAnswersCopied ? 'check' : 'copy'" :size="14" />
        <span>{{ numberedAnswersCopied ? '已复制题号＋答案' : '复制题号＋答案' }}</span>
      </button>
      <button class="action-btn" :disabled="!answerCopyText || copying" @click="handleCopyAnswers"
        :title="answerCopyText ? '仅复制本轮可填写答案，每题一行，不含题号与解析' : (answerItems.length ? '含待核对项，暂不可复制答案' : '本轮未生成可复制的答案汇总')">
        <Icon :name="answersCopied ? 'check' : 'copy'" :size="14" />
        <span>{{ answersCopied ? '已复制答案' : '复制答案' }}</span>
      </button>
      <span v-if="copyError" class="copy-error" role="status">{{ copyError }}</span>
      <button class="action-btn" @click="handleCopy" :title="copied ? '已复制' : '复制全文'">
        <Icon :name="copied ? 'check' : 'copy'" :size="14" />
        <span>{{ copied ? '已复制全文' : '复制全文' }}</span>
      </button>
      <button
        class="action-btn"
        :class="{ busy: questionBank.busy }"
        :disabled="!questionBank.canExportCurrent || questionBank.busy"
        @click="questionBank.exportCurrent"
        :title="exportTitle"
      >
        <Icon :name="questionBank.busy ? 'loader' : 'download'" :size="14" :spinning="questionBank.busy" />
        <span>{{ exportLabel }}</span>
      </button>
      <button
        v-if="questionBank.canEndCurrent"
        class="action-btn"
        :disabled="questionBank.busy"
        title="封存当前段；下一次解题会自动建立新段"
        @click="questionBank.splitCurrent"
      >
        <Icon name="check-circle" :size="14" />
        <span>结束本段</span>
      </button>
      <button
        class="action-btn"
        :class="{ busy: questionBank.nextExamSubmitting }"
        :disabled="questionBank.nextExamPending || questionBank.nextExamSubmitting"
        title="封存当前段并清理本机运行态，准备开始下一套笔试"
        @click="questionBank.requestStartNextExam"
      >
        <Icon name="refresh" :size="14" :spinning="questionBank.nextExamSubmitting" />
        <span>{{ questionBank.nextExamSubmitting ? '切换中…' : '开始下一套' }}</span>
      </button>
    </div>
    <div class="action-right">
      <span class="meta-text" v-if="model">{{ model }}</span>
      <span class="meta-dot" v-if="model && duration">·</span>
      <span class="meta-text" v-if="duration">{{ duration }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { copyText, formatNumberedAnswers } from '../utils/clipboard'
import Icon from './Icon.vue'
import { useQuestionBankStore } from '../stores/questionbank'

const props = defineProps({
  content: { type: String, default: '' },
  answerItems: { type: Array, default: () => [] },
  answerCopyText: { type: String, default: '' },
  model: { type: String, default: '' },
  duration: { type: String, default: '' },
})

const questionBank = useQuestionBankStore()
const numberedCopyText = computed(() => formatNumberedAnswers(props.answerItems, props.answerCopyText))
const numberedAnswersCopied = ref(false)
const copied = ref(false)
const answersCopied = ref(false)
const copying = ref(false)
const copyError = ref('')
let feedbackTimer
let copyVersion = 0
function resetCopyFeedback() {
  copyVersion++
  clearTimeout(feedbackTimer)
  numberedAnswersCopied.value = false
  copied.value = false
  answersCopied.value = false
  copying.value = false
  copyError.value = ''
}
watch(() => [props.content, props.answerCopyText, numberedCopyText.value], resetCopyFeedback)
onBeforeUnmount(resetCopyFeedback)

const exportLabel = computed(() => {
  if (questionBank.status.organizing) {
    if (questionBank.status.operation === 'exporting') return '导出中…'
    const done = questionBank.status.progress?.done || 0
    const total = questionBank.status.progress?.total || 0
    return total > 0 ? `整理中 ${done}/${total}` : '整理中…'
  }
  if (questionBank.exportLoading) return '准备导出…'
  return '导出题库'
})

const exportTitle = computed(() => {
  if (questionBank.busy) return exportLabel.value
  if (!questionBank.canExportCurrent) return '当前没有可导出的题库分段'
  return '确认名称、位置和本次是否调用模型后导出当前分段'
})

async function performCopy(text, mode) {
  if (!text || copying.value) return
  const version = ++copyVersion
  copying.value = true
  copyError.value = ''
  const success = await copyText(text)
  if (version !== copyVersion) return
  copying.value = false
  copied.value = success && mode === 'full'
  answersCopied.value = success && mode === 'answers'
  numberedAnswersCopied.value = success && mode === 'numbered'
  if (!success) copyError.value = '复制失败，请手动选择文本复制'
  clearTimeout(feedbackTimer)
  feedbackTimer = setTimeout(resetCopyFeedback, 2500)
}
function handleCopy() { return performCopy(props.content, 'full') }
function handleCopyAnswers() { return performCopy(props.answerCopyText, 'answers') }
function handleCopyNumberedAnswers() { return performCopy(numberedCopyText.value, 'numbered') }
</script>

<style scoped>
.copy-error { color: var(--color-error); font-size: var(--text-xs); }
.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-2) 0;
  margin-top: var(--sp-3);
  border-top: 1px solid var(--border-subtle);
}
.action-left {
  display: flex;
  gap: var(--sp-1);
  flex-wrap: wrap;
}
.action-btn {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
  padding: var(--sp-1) var(--sp-2);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.action-btn:hover:not(:disabled) {
  background: var(--surface-card);
  color: var(--text-secondary);
}
.action-btn.busy,
.action-btn:disabled {
  color: var(--text-muted);
  cursor: not-allowed;
  opacity: 0.55;
}
.action-btn.busy { color: var(--accent); cursor: wait; opacity: 0.8; }
.action-right {
  display: flex;
  align-items: center;
  gap: var(--sp-1-5);
}
.meta-text {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-weight: var(--weight-medium);
}
.meta-dot {
  color: var(--text-muted);
  opacity: 0.4;
  font-size: 10px;
}
</style>
