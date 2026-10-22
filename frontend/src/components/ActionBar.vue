<template>
  <div class="action-bar">
    <div class="action-left">
      <button class="action-btn" @click="handleCopy" :title="copied ? '已复制' : '复制全文'">
        <Icon :name="copied ? 'check' : 'copy'" :size="14" />
        <span>{{ copied ? '已复制' : '复制' }}</span>
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
    </div>
    <div class="action-right">
      <span class="meta-text" v-if="model">{{ model }}</span>
      <span class="meta-dot" v-if="model && duration">·</span>
      <span class="meta-text" v-if="duration">{{ duration }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import { useQuestionBankStore } from '../stores/questionbank'

const props = defineProps({
  content: { type: String, default: '' },
  model: { type: String, default: '' },
  duration: { type: String, default: '' },
})

const questionBank = useQuestionBankStore()
const copied = ref(false)

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

function handleCopy() {
  if (!props.content) return
  navigator.clipboard.writeText(props.content).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  }).catch(() => {
    const ta = document.createElement('textarea')
    ta.value = props.content
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  })
}
</script>

<style scoped>
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
