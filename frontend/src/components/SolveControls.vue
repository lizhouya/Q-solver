<template>
  <div
    v-if="running || cancelled || canRetry || controlError"
    class="solve-controls"
    :class="{ interrupted: cancelled }"
    role="status"
  >
    <div v-if="running" class="solve-state">
      <span class="state-dot" aria-hidden="true"></span>
      <span>{{ mode === 'fast' ? '快速解题中' : '正在解题' }}</span>
    </div>

    <div v-if="cancelled" class="interruption-copy">
      <strong>{{ interruption?.title || '已中断' }}</strong>
      <span>{{ interruption?.desc || '已中断，以上内容可能不完整' }}</span>
    </div>

    <span v-if="controlError" class="control-error">{{ controlError }}</span>

    <div class="control-actions">
      <button
        v-if="running"
        type="button"
        class="control-btn cancel-btn"
        :disabled="busy || cancelling"
        @click="emit('cancel')"
      >
        {{ cancelling ? '正在中断…' : '中断解题' }}
        <kbd v-if="cancelShortcut && !cancelling">{{ cancelShortcut }}</kbd>
      </button>
      <button
        v-if="canRetry"
        type="button"
        class="control-btn retry-btn"
        :disabled="busy"
        @click="emit('retry')"
      >
        {{ busy ? '正在重试…' : '重试本题' }}
      </button>
    </div>
  </div>
</template>

<script setup>
defineProps({
  mode: { type: String, default: 'normal' },
  running: { type: Boolean, default: false },
  cancelling: { type: Boolean, default: false },
  cancelled: { type: Boolean, default: false },
  canRetry: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  interruption: { type: Object, default: null },
  controlError: { type: String, default: '' },
  cancelShortcut: { type: String, default: '' },
})

const emit = defineEmits(['retry', 'cancel'])
</script>

<style scoped>
.solve-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--sp-3);
  margin-top: var(--sp-3);
  padding: var(--sp-3);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-card);
}

.solve-controls.interrupted {
  border-color: var(--warning-border);
  background: var(--warning-bg);
}

.solve-state {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
}

.state-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 8px var(--accent-glow);
  animation: statePulse 1.2s ease-in-out infinite;
}

.interruption-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: var(--sp-1);
  color: var(--text-secondary);
  font-size: var(--text-xs);
  line-height: var(--leading-normal);
}

.interruption-copy strong {
  color: var(--color-warning);
  font-size: var(--text-sm);
}

.control-error {
  color: var(--color-error);
  font-size: var(--text-xs);
}

.control-actions {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-left: auto;
}

.control-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  min-height: 32px;
  padding: var(--sp-1-5) var(--sp-3);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-elevated);
  color: var(--text-primary);
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: border-color var(--duration-fast) ease, color var(--duration-fast) ease, background var(--duration-fast) ease;
}

.control-btn:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--surface-card-hover);
}

.cancel-btn:hover:not(:disabled) {
  border-color: var(--color-warning);
  color: var(--color-warning);
}

.control-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.control-btn kbd {
  padding: 1px 5px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xs);
  background: var(--surface-card);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 10px;
}

@keyframes statePulse {
  0%, 100% { opacity: 0.45; transform: scale(0.85); }
  50% { opacity: 1; transform: scale(1); }
}
</style>
