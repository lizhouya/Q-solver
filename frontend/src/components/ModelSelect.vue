<template>
  <div class="model-select-container">
    <div class="model-input-shell" :class="{ open: isOpen, disabled }" ref="selectRef">
      <input
        :value="modelValue"
        class="model-input"
        type="text"
        autocomplete="off"
        :disabled="disabled"
        :placeholder="placeholder"
        @input="updateValue"
        @focus="openSuggestions"
      />
      <button
        class="suggestion-toggle"
        type="button"
        :disabled="disabled || loading"
        aria-label="显示模型建议"
        @click="toggle"
      >
        <span v-if="loading" class="loading-icon">⏳</span>
        <svg v-else width="12" height="12" viewBox="0 0 12 12" fill="none" :class="{ rotated: isOpen }">
          <path d="M2.5 4.5L6 8L9.5 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
    </div>

    <div v-if="modelValue" class="model-reference">
      <p v-if="connectionMode === 'ccr'">模型由 CCR 目录提供；Q-Solver 固定通过 Chat Completions 入站调用。</p>
      <div v-if="connectionMode !== 'ccr'" class="model-reference-main">
        <div class="provider-logo">
          <div v-if="providerLogo && providerLogo.trim().startsWith('<svg')" v-html="providerLogo"></div>
          <img v-else-if="providerLogo" :src="providerLogo" alt="" />
        </div>
        <span>{{ getProviderName(modelValue) }}</span>
        <span v-if="capabilities.image" class="cap-tag" title="模型目录标记：图片">图片</span>
        <span v-if="capabilities.file" class="cap-tag" title="模型目录标记：文件">文件</span>
        <span v-if="capabilities.contextLength" class="cap-tag">{{ formatContextLength(capabilities.contextLength) }}</span>
      </div>
      <p v-if="connectionMode !== 'ccr'">模型目录参考，非当前通道实测</p>
    </div>

    <Teleport to="body">
      <Transition name="dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="model-dropdown-portal"
          :style="dropdownStyle"
          @click.stop
          @wheel.stop
          @touchmove.stop
        >
          <div v-if="loading" class="loading-state">加载中...</div>
          <template v-else-if="filteredModels.length">
            <button
              v-for="model in filteredModels"
              :key="model"
              type="button"
              class="dropdown-item"
              :class="{ selected: modelValue === model }"
              @click="selectModel(model)"
            >
              <span class="model-name">{{ model }}</span>
              <span v-if="connectionMode !== 'ccr'" class="provider-name">{{ getProviderName(model) }}</span>
              <span v-if="modelValue === model" class="check-icon">✓</span>
            </button>
          </template>
          <div v-else class="empty-state">没有模型建议，仍可直接输入模型名称</div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { formatContextLength, getModelCapabilities, getProviderLogo, getProviderName } from '../utils/modelCapabilities'

const props = defineProps({
  connectionMode: { type: String, default: 'direct' },
  modelValue: { type: String, default: '' },
  models: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  placeholder: { type: String, default: '输入模型名称或从建议中选择' },
  active: { type: Boolean, default: true },
})

const emit = defineEmits(['update:modelValue', 'open-change'])
const isOpen = ref(false)
const searchQuery = ref('')
const selectRef = ref(null)
const dropdownRef = ref(null)
const dropdownStyle = reactive({ position: 'fixed', top: '0px', left: '0px', width: '0px', zIndex: 9999 })
const providerLogo = computed(() => getProviderLogo(props.modelValue))
const capabilities = computed(() => getModelCapabilities(props.modelValue))
const filteredModels = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return props.models
  return props.models.filter(model => String(model).toLowerCase().includes(query))
})

function updateDropdownPosition() {
  if (!selectRef.value) return
  const rect = selectRef.value.getBoundingClientRect()
  dropdownStyle.top = `${rect.bottom + 4}px`
  dropdownStyle.left = `${rect.left}px`
  dropdownStyle.width = `${rect.width}px`
}

function setOpen(value) {
  value = Boolean(value)
  if (isOpen.value === value) return
  isOpen.value = value
  emit('open-change', value)
}

function openSuggestions() {
  if (props.disabled || !props.active) return
  searchQuery.value = ''
  setOpen(true)
  updateDropdownPosition()
}

function toggle() {
  if (props.disabled || props.loading) return
  setOpen(!isOpen.value)
  searchQuery.value = ''
  if (isOpen.value) nextTick(updateDropdownPosition)
}

function updateValue(event) {
  const value = event.target.value
  searchQuery.value = value
  emit('update:modelValue', value)
  setOpen(true)
  nextTick(updateDropdownPosition)
}

function selectModel(model) {
  searchQuery.value = ''
  emit('update:modelValue', model)
  setOpen(false)
}

function handleClickOutside(event) {
  if (selectRef.value?.contains(event.target) || dropdownRef.value?.contains(event.target)) return
  searchQuery.value = ''
  setOpen(false)
}

function handleWindowChange() {
  if (isOpen.value) updateDropdownPosition()
}

watch(() => props.active, active => {
  if (!active) {
    searchQuery.value = ''
    setOpen(false)
  }
})

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('resize', handleWindowChange)
  window.addEventListener('scroll', handleWindowChange, true)
})

onUnmounted(() => {
  setOpen(false)
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('resize', handleWindowChange)
  window.removeEventListener('scroll', handleWindowChange, true)
})
</script>

<style scoped>
.model-select-container { width: 100%; }
.model-input-shell {
  display: flex;
  min-height: 46px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-elevated);
  transition: all var(--duration-fast) ease;
}
.model-input-shell:hover { border-color: var(--border-default); }
.model-input-shell.open,
.model-input-shell:focus-within { border-color: var(--accent); box-shadow: var(--shadow-glow); }
.model-input-shell.disabled { opacity: 0.6; }
.model-input {
  flex: 1;
  min-width: 0;
  padding: 0 var(--sp-4);
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-sm);
}
.model-input::placeholder { color: var(--text-tertiary); }
.suggestion-toggle {
  width: 42px;
  border: 0;
  border-left: 1px solid var(--border-subtle);
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
}
.suggestion-toggle:disabled { cursor: not-allowed; opacity: 0.5; }
.suggestion-toggle svg { transition: transform var(--duration-fast) ease; }
.suggestion-toggle svg.rotated { transform: rotate(180deg); }
.loading-icon { display: inline-block; animation: selectSpin 1s linear infinite; }
@keyframes selectSpin { to { transform: rotate(360deg); } }
.model-reference { margin-top: var(--sp-2); color: var(--text-tertiary); font-size: var(--text-xs); }
.model-reference-main { display: flex; align-items: center; flex-wrap: wrap; gap: var(--sp-2); }
.model-reference p { margin: 5px 0 0; color: var(--text-muted); }
.provider-logo { width: 18px; height: 18px; display: flex; }
.provider-logo :deep(svg), .provider-logo img { width: 100%; height: 100%; object-fit: contain; }
.cap-tag { padding: 2px 6px; border-radius: var(--radius-xs); background: var(--accent-muted); color: var(--accent); font-family: var(--font-mono); }
.dropdown-enter-active, .dropdown-leave-active { transition: all var(--duration-fast) ease; }
.dropdown-enter-from, .dropdown-leave-to { opacity: 0; transform: translateY(-8px); }
</style>

<style>
.model-dropdown-portal {
  position: fixed;
  max-height: min(240px, 35vh);
  overflow-y: auto;
  overscroll-behavior: contain;
  touch-action: pan-y;
  background: var(--surface-popover);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-xl);
  backdrop-filter: blur(20px);
}
.model-dropdown-portal .dropdown-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-4);
  border: 0;
  border-bottom: 1px solid var(--border-subtle);
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}
.model-dropdown-portal .dropdown-item:hover,
.model-dropdown-portal .dropdown-item.selected { background: var(--accent-muted); }
.model-dropdown-portal .model-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.model-dropdown-portal .provider-name { color: var(--text-tertiary); font-size: var(--text-xs); }
.model-dropdown-portal .check-icon { color: var(--accent); font-weight: 700; }
.model-dropdown-portal .loading-state,
.model-dropdown-portal .empty-state { padding: var(--sp-4); color: var(--text-tertiary); font-size: var(--text-xs); text-align: center; }
</style>
