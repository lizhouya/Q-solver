<template>
  <Teleport to="body">
    <Transition name="overlay-fade">
      <div v-if="mobile.pendingAction" class="mobile-confirm-overlay" role="presentation" @click.self="cancel">
        <section class="mobile-confirm-dialog" role="alertdialog" aria-modal="true" :aria-labelledby="titleId">
          <div class="mobile-confirm-icon" aria-hidden="true">
            <Icon name="alert-triangle" :size="28" />
          </div>
          <h2 :id="titleId">{{ mobile.pendingAction.title }}</h2>
          <p>{{ mobile.pendingAction.description }}</p>
          <div class="mobile-confirm-notice">
            已复制、截图或离线保存的内容无法撤回。操作成功只表示状态已提交并广播，不代表所有终端已经显示新状态。
          </div>
          <div class="mobile-confirm-actions">
            <button class="btn-cancel" :disabled="mobile.mutating" @click="cancel">取消</button>
            <button class="btn-confirm" :disabled="mobile.mutating" @click="mobile.confirmPendingAction">
              {{ mobile.mutating ? '处理中…' : mobile.pendingAction.confirmLabel }}
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import Icon from './Icon.vue'
import { useMobileStore } from '../stores/mobile'

const mobile = useMobileStore()
const titleId = 'mobile-action-confirm-title'

function cancel() {
  mobile.cancelPendingAction()
}
</script>

<style scoped>
.mobile-confirm-overlay {
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
.mobile-confirm-dialog {
  width: min(420px, 100%);
  padding: var(--sp-7);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-xl);
  text-align: center;
}
.mobile-confirm-icon {
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
h2 { margin: 0; color: var(--text-primary); font-size: var(--text-lg); }
p { margin: var(--sp-3) 0 0; color: var(--text-secondary); font-size: var(--text-sm); line-height: 1.7; }
.mobile-confirm-notice {
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
.mobile-confirm-actions { display: flex; justify-content: flex-end; gap: var(--sp-2); margin-top: var(--sp-5); }
.mobile-confirm-actions button {
  padding: var(--sp-2) var(--sp-5);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  cursor: pointer;
}
.btn-cancel { border: 1px solid var(--border-default); background: var(--surface-card); color: var(--text-primary); }
.btn-confirm { border: 1px solid var(--color-warning); background: var(--color-warning); color: var(--text-inverse); }
.mobile-confirm-actions button:disabled { opacity: 0.55; cursor: not-allowed; }
</style>
