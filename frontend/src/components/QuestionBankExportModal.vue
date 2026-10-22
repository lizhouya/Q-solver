<template>
  <Teleport to="body">
    <Transition name="qb-export-fade">
      <div
        v-if="bank.exportVisible"
        class="qb-export-overlay"
        @keydown.esc.stop.prevent="bank.closeExport"
      >
        <section
          ref="dialog"
          class="qb-export-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="qb-export-title"
          tabindex="-1"
        >
          <header class="qb-export-header">
            <div>
              <h2 id="qb-export-title">导出题库</h2>
              <p>确认本次分段的信息和输出方式；取消不会封存分段，也不会调用模型。</p>
            </div>
            <button
              class="qb-icon-button"
              type="button"
              aria-label="关闭导出窗口"
              :disabled="bank.exportSubmitting"
              @click="bank.closeExport"
            >
              <Icon name="x" :size="18" />
            </button>
          </header>

          <div v-if="bank.exportLoading" class="qb-export-loading" role="status">
            <Icon name="loader" :size="20" spinning />
            <span>正在读取分段信息…</span>
          </div>

          <form v-else class="qb-export-form" @submit.prevent="bank.submitExport">
            <div class="qb-export-summary">
              <span><strong>{{ bank.exportDraft.count }}</strong> 次提交</span>
              <span>{{ formatTime(bank.exportDraft.startedAt) }}</span>
              <span class="qb-export-id">{{ bank.exportDraft.sessionId }}</span>
            </div>

            <div class="qb-field-grid">
              <label class="qb-field">
                <span>单位（可选）</span>
                <input
                  :value="bank.exportDraft.company"
                  maxlength="120"
                  placeholder="不明确时可留空"
                  :disabled="bank.exportSubmitting"
                  @input="bank.setExportCompany($event.target.value)"
                />
              </label>
              <label class="qb-field">
                <span>题目类型</span>
                <input
                  :value="bank.exportDraft.type"
                  maxlength="120"
                  placeholder="例如：行测、编程与算法"
                  :disabled="bank.exportSubmitting"
                  @input="bank.setExportType($event.target.value)"
                />
              </label>
            </div>

            <label class="qb-field">
              <span>题库名称</span>
              <input
                :value="bank.exportDraft.name"
                maxlength="120"
                placeholder="例如：招商银行_行测"
                :disabled="bank.exportSubmitting"
                @input="bank.setExportName($event.target.value)"
              />
              <small>默认按单位和题型生成；手动修改后，不再被单位或题型变化覆盖。</small>
            </label>

            <div class="qb-field">
              <span>本次导出位置</span>
              <div class="qb-directory-row">
                <code>{{ bank.exportDraft.directory || '尚未选择' }}</code>
                <button
                  type="button"
                  :disabled="bank.exportDirectorySelecting || bank.exportSubmitting"
                  @click="bank.selectExportDirectory"
                >
                  {{ bank.exportDirectorySelecting ? '选择中…' : '修改位置' }}
                </button>
              </div>
              <small>仅影响本次导出，不会修改设置中的默认目录。</small>
            </div>

            <label class="qb-organize-switch" :class="{ enabled: bank.exportDraft.autoOrganize }">
              <input
                v-model="bank.exportDraft.autoOrganize"
                type="checkbox"
                :disabled="bank.exportSubmitting"
              />
              <span>
                <strong>本次调用模型整理</strong>
                <small v-if="bank.exportDraft.autoOrganize">
                  提交后才会把本段截图和原答案发送给已配置的模型服务商，可能产生额外费用。
                </small>
                <small v-else>
                  只在本地生成原始记录和全部截图，不生成空的整理试卷。
                </small>
              </span>
            </label>

            <div class="qb-output-preview">
              <strong>预计输出</strong>
              <span>{{ bank.exportExpectedFile }}</span>
              <small>每次导出创建新的版本目录，不覆盖旧记录；整理稿不包含图片。</small>
            </div>

            <p v-if="bank.exportError" class="qb-export-error" role="alert">{{ bank.exportError }}</p>

            <footer class="qb-export-actions">
              <button
                type="button"
                class="qb-secondary"
                :disabled="bank.exportSubmitting"
                @click="bank.closeExport"
              >取消</button>
              <button
                type="submit"
                class="qb-primary"
                :disabled="bank.exportSubmitting || bank.exportDirectorySelecting"
              >
                <Icon v-if="bank.exportSubmitting" name="loader" :size="14" spinning />
                {{ submitLabel }}
              </button>
            </footer>
          </form>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { useQuestionBankStore } from '../stores/questionbank'

const bank = useQuestionBankStore()
const dialog = ref(null)

const submitLabel = computed(() => {
  if (bank.exportSubmitting) return '正在提交…'
  return bank.exportDraft.autoOrganize ? '整理并导出' : '仅导出原始记录'
})

function formatTime(value) {
  if (!value) return '开始时间未知'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '开始时间未知' : date.toLocaleString()
}

watch(() => bank.exportVisible, async (visible) => {
  if (!visible) return
  await nextTick()
  dialog.value?.focus()
})
</script>

<style scoped>
.qb-export-overlay {
  position: fixed;
  inset: 0;
  z-index: 3200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  pointer-events: auto;
  background: var(--surface-overlay, rgba(0, 0, 0, 0.55));
  backdrop-filter: blur(7px);
}

.qb-export-dialog {
  width: min(620px, 100%);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  outline: none;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--surface-popover);
  color: var(--text-primary);
  box-shadow: var(--shadow-xl);
}

.qb-export-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.qb-export-header h2 { margin: 0 0 .3rem; font-size: 1.1rem; }
.qb-export-header p { margin: 0; color: var(--text-secondary); font-size: .78rem; line-height: 1.55; }
.qb-icon-button { display: flex; padding: .35rem; flex-shrink: 0; }

.qb-export-loading {
  min-height: 240px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: .6rem;
  color: var(--text-secondary);
  font-size: .85rem;
}

.qb-export-form {
  padding: 1rem 1.25rem 1.2rem;
  overflow-y: auto;
}

.qb-export-summary {
  display: flex;
  align-items: center;
  gap: .55rem;
  flex-wrap: wrap;
  padding: .65rem .75rem;
  margin-bottom: 1rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
  color: var(--text-secondary);
  font-size: .75rem;
}
.qb-export-summary span + span::before { content: '·'; margin-right: .55rem; color: var(--text-muted); }
.qb-export-summary strong { color: var(--accent); }
.qb-export-id { max-width: 15rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.qb-field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: .75rem; }
.qb-field { display: flex; flex-direction: column; gap: .4rem; margin-bottom: .9rem; }
.qb-field > span { font-size: .8rem; font-weight: 650; }
.qb-field small, .qb-organize-switch small, .qb-output-preview small { color: var(--text-muted); font-size: .72rem; line-height: 1.55; }

input {
  width: 100%;
  box-sizing: border-box;
  padding: .58rem .7rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  outline: none;
  background: var(--surface-input);
  color: var(--text-primary);
  font-size: .82rem;
}
input:focus { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-muted); }

.qb-directory-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .55rem; }
.qb-directory-row code {
  min-height: 38px;
  display: flex;
  align-items: center;
  padding: .5rem .65rem;
  overflow-wrap: anywhere;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-input);
  color: var(--text-secondary);
  font-size: .72rem;
}

.qb-organize-switch {
  display: flex;
  align-items: flex-start;
  gap: .65rem;
  padding: .8rem;
  margin-bottom: .9rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
  cursor: pointer;
}
.qb-organize-switch.enabled { border-color: var(--warning-border); background: var(--warning-bg); }
.qb-organize-switch input { width: auto; margin-top: .18rem; accent-color: var(--accent); }
.qb-organize-switch span { display: flex; flex-direction: column; gap: .25rem; }
.qb-organize-switch strong { font-size: .82rem; }
.qb-organize-switch.enabled small { color: var(--color-warning); }

.qb-output-preview {
  display: flex;
  flex-direction: column;
  gap: .25rem;
  padding: .75rem;
  border-left: 3px solid var(--accent);
  border-radius: var(--radius-sm);
  background: var(--accent-muted);
  font-size: .78rem;
}

.qb-export-error {
  padding: .7rem;
  margin: .8rem 0 0;
  border: 1px solid var(--error-border);
  border-radius: var(--radius-sm);
  background: var(--error-bg);
  color: var(--color-error);
  font-size: .78rem;
  overflow-wrap: anywhere;
}

.qb-export-actions { display: flex; justify-content: flex-end; gap: .6rem; margin-top: 1rem; }
button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: .35rem;
  padding: .55rem .8rem;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
  color: var(--text-primary);
  font-size: .78rem;
  cursor: pointer;
}
button:hover:not(:disabled) { background: var(--surface-card-hover); }
button:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
button:disabled { opacity: .5; cursor: not-allowed; }
.qb-primary { min-width: 145px; border-color: var(--accent); background: var(--accent); color: white; }
.qb-primary:hover:not(:disabled) { background: var(--accent-hover); }
.qb-secondary { min-width: 70px; }

.qb-export-fade-enter-active { transition: opacity .2s var(--ease-out); }
.qb-export-fade-leave-active { transition: opacity .15s ease-in; }
.qb-export-fade-enter-from, .qb-export-fade-leave-to { opacity: 0; }
.qb-export-fade-enter-from .qb-export-dialog, .qb-export-fade-leave-to .qb-export-dialog { transform: translateY(8px) scale(.98); }
.qb-export-dialog { transition: transform .2s var(--ease-out); }

@media (max-width: 560px) {
  .qb-export-overlay { align-items: flex-start; padding: .6rem; }
  .qb-export-dialog { max-height: calc(100vh - 1.2rem); }
  .qb-field-grid, .qb-directory-row { grid-template-columns: 1fr; }
  .qb-export-actions { flex-direction: column-reverse; }
  .qb-export-actions button { width: 100%; }
  .qb-export-id { display: none; }
}
</style>
