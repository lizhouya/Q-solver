<template>
  <Teleport to="body">
    <Transition name="overlay-fade">
      <div
        v-if="questionBank.nextExamPending"
        class="next-exam-overlay"
        role="presentation"
        @click.self="cancel"
      >
        <section
          class="next-exam-dialog"
          role="alertdialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          aria-describedby="next-exam-confirm-description"
        >
          <div class="next-exam-icon" aria-hidden="true">
            <Icon name="alert-triangle" :size="28" />
          </div>
          <h2 :id="titleId">确认开始下一套？</h2>
          <p id="next-exam-confirm-description" class="next-exam-description">
            成功后将封存当前段，并清空桌面历史、未发送截图和模型上下文。
          </p>
          <div class="next-exam-preserved">
            题库原始记录、归档截图、旧导出和同步端当前显示都会保留，不会被清除。
          </div>
          <p v-if="questionBank.nextExamError" class="next-exam-error" role="alert" aria-live="assertive">
            {{ questionBank.nextExamError }}
          </p>
          <div class="next-exam-actions">
            <button
              class="btn-cancel"
              :disabled="questionBank.nextExamSubmitting"
              @click="cancel"
            >
              取消
            </button>
            <button
              class="btn-confirm"
              :disabled="questionBank.nextExamSubmitting"
              @click="questionBank.confirmStartNextExam"
            >
              {{ questionBank.nextExamSubmitting ? '处理中…' : '确认开始下一套' }}
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import Icon from './Icon.vue'
import { useQuestionBankStore } from '../stores/questionbank'

const questionBank = useQuestionBankStore()
const titleId = 'next-exam-confirm-title'

function cancel() {
  questionBank.cancelStartNextExam()
}
</script>

<style scoped>
.next-exam-overlay {
  position: fixed;
  z-index: calc(var(--z-modal) + 20);
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-5);
  background: var(--surface-overlay);
  backdrop-filter: blur(7px);
  pointer-events: auto;
}

.next-exam-dialog {
  width: min(440px, 100%);
  padding: var(--sp-7);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-xl);
  text-align: center;
}

.next-exam-icon {
  width: 52px;
  height: 52px;
  margin: 0 auto var(--sp-4);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-lg);
  background: var(--warning-bg);
  color: var(--color-warning);
}

h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: var(--text-lg);
}

.next-exam-description {
  margin: var(--sp-3) 0 0;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.7;
}

.next-exam-preserved {
  margin-top: var(--sp-4);
  padding: var(--sp-3);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-card);
  color: var(--text-muted);
  font-size: var(--text-xs);
  line-height: 1.6;
  text-align: left;
}

.next-exam-error {
  margin: var(--sp-3) 0 0;
  padding: var(--sp-3);
  border: 1px solid var(--error-border);
  border-radius: var(--radius-md);
  background: var(--error-bg);
  color: var(--color-error);
  font-size: var(--text-xs);
  line-height: 1.6;
  text-align: left;
}

.next-exam-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
  margin-top: var(--sp-5);
}

.next-exam-actions button {
  padding: var(--sp-2) var(--sp-5);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  cursor: pointer;
}

.btn-cancel {
  border: 1px solid var(--border-default);
  background: var(--surface-card);
  color: var(--text-primary);
}

.btn-confirm {
  border: 1px solid var(--color-warning);
  background: var(--color-warning);
  color: var(--text-inverse);
}

.next-exam-actions button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
</style>
