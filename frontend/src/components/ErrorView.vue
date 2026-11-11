<template>
  <div class="error-container">
    <div class="error-icon-wrap">
      <Icon :name="errorState.icon || 'x-circle'" :size="32" class="error-icon" />
    </div>
    <div class="error-title">{{ errorState.title }}</div>
    <div class="error-desc">{{ errorState.desc }}</div>
    <ErrorDiagnostics :detail="errorState.detail" />
    <button
      v-if="canRetry"
      type="button"
      class="retry-btn"
      :disabled="busy"
      @click="emit('retry')"
    >{{ busy ? '正在重试…' : '重试本题' }}</button>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'
import ErrorDiagnostics from './ErrorDiagnostics.vue'

defineProps({
  errorState: Object,
  canRetry: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
})

const emit = defineEmits(['retry'])
</script>

<style scoped>
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: var(--sp-5);
  animation: fadeIn 0.3s var(--ease-out);
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.error-icon-wrap {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--error-bg);
  border: 1px solid var(--error-border);
  border-radius: var(--radius-xl);
  margin-bottom: var(--sp-5);
}
.error-icon {
  color: var(--color-error);
}
.error-title {
  font-size: var(--text-xl);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  margin-bottom: var(--sp-2);
}
.error-desc {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin-bottom: var(--sp-6);
  max-width: 80%;
  line-height: var(--leading-relaxed);
}
.retry-btn {
  margin-top: var(--sp-4);
  padding: var(--sp-2) var(--sp-4);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-sm);
  background: var(--accent-muted);
  color: var(--accent);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: background var(--duration-fast) ease, border-color var(--duration-fast) ease;
}
.retry-btn:hover:not(:disabled) {
  background: var(--surface-card-hover);
  border-color: var(--accent);
}
.retry-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
</style>
