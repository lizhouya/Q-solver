<template>
  <div class="provider-page">
    <section class="overview-card">
      <div class="overview-header">
        <div class="overview-logo">
          <span v-if="connectionMode === 'ccr'" class="ccr-logo">CCR</span>
          <div v-else-if="selectedProviderLogo.trim().startsWith('<svg')" v-html="selectedProviderLogo"></div>
          <img v-else :src="selectedProviderLogo" :alt="selectedProviderName" />
        </div>
        <div class="overview-copy">
          <span class="overview-tag">{{ selectedProtocolLabel }}</span>
          <h3>{{ selectedProviderName }}</h3>
          <p>{{ selectedProviderDescription }}</p>
        </div>
      </div>
    </section>

    <section class="form-card">
      <div class="form-item">
        <label class="field-label"><span>连接模式</span><span class="field-meta">Mode</span></label>
        <div class="input-shell">
          <select class="protocol-select" :value="connectionMode" @change="changeConnectionMode($event.target.value)">
            <option value="direct">Direct 直连</option>
            <option value="ccr">CCR 统一网关</option>
          </select>
        </div>
        <p class="field-hint">Direct 由 Q-Solver 直接连接服务商；CCR 由统一网关负责上游路由与协议转换。</p>
      </div>

      <div v-if="connectionMode === 'direct'" class="form-item">
        <label class="field-label">
          <span>模型提供商</span>
          <span class="field-meta">Provider</span>
        </label>
        <div class="select-shell" :class="{ open: isOpen }" @click="toggle" ref="selectRef">
          <div class="select-leading">
            <div class="provider-logo">
              <div v-if="selectedProviderLogo.trim().startsWith('<svg')" v-html="selectedProviderLogo"></div>
              <img v-else :src="selectedProviderLogo" :alt="selectedProviderName" />
            </div>
          </div>
          <div class="select-body">
            <span class="select-title">{{ selectedProviderName }}</span>
            <span class="select-subtitle">{{ selectedProviderCode === 'custom' ? '手动配置接口地址' : '使用默认兼容地址' }}</span>
          </div>
          <div class="select-arrow" :class="{ rotated: isOpen }">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M3.5 5.5L7 9L10.5 5.5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </div>
        </div>
      </div>

      <div v-if="connectionMode === 'direct'" class="form-item">
        <label class="field-label">
          <span>API 协议</span>
          <span class="field-meta">Protocol</span>
        </label>
        <div class="input-shell">
          <select
            class="protocol-select"
            :value="apiProtocol"
            @change="emit('update:apiProtocol', $event.target.value)"
          >
            <option value="openai_chat">Chat Completions</option>
            <option value="openai_responses">Responses API</option>
            <option value="anthropic_messages">Anthropic Messages</option>
          </select>
        </div>
        <p class="field-hint">协议与品牌独立；请按当前服务端实际支持的协议选择。</p>
      </div>

      <div v-if="connectionMode === 'ccr' || selectedProviderCode === 'custom'" class="form-item">
        <label class="field-label">
          <span>{{ connectionMode === 'ccr' ? 'CCR 网关地址' : '接口地址' }}</span>
          <span class="field-meta">Base URL</span>
        </label>
        <div class="input-shell">
          <span class="input-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M10 14L21 3M21 3H14M21 3V10" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
              <path d="M21 14V19C21 20.1046 20.1046 21 19 21H5C3.89543 21 3 20.1046 3 19V5C3 3.89543 3.89543 3 5 3H10" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
            </svg>
          </span>
          <input
            type="text"
            :value="baseURL"
            @input="emit('update:baseURL', $event.target.value)"
            :placeholder="connectionMode === 'ccr' ? 'http://127.0.0.1:3456/v1' : 'https://api.openai.com/v1'"
            class="text-input"
          />
        </div>
        <p class="field-hint">{{ connectionMode === 'ccr' ? '填写 CCR 根地址或 /v1 API root；不要填写 models 或 chat/completions endpoint。' : '填写 API root，不要填写完整的 messages、responses 或 chat/completions endpoint。' }}</p>
      </div>

      <div class="form-item">
        <label class="field-label">
          <span>{{ connectionMode === 'ccr' ? 'CCR API Key' : `${selectedProviderName} API Key` }}</span>
          <span class="field-meta">Credential</span>
        </label>
        <div class="input-shell">
          <span class="input-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
              <path d="M21 2L12 11M21 2C21.5523 2 22 2.44772 22 3V6.5C22 6.63261 21.9473 6.75979 21.8536 6.85355L19.4393 9.26777C18.596 10.1111 17.2292 10.1111 16.3858 9.26777L14.7322 7.61421C13.8889 6.77088 13.8889 5.40404 14.7322 4.56071L17.1464 2.14645C17.2402 2.05268 17.3674 2 17.5 2H21ZM10 14C11.6569 14 13 12.6569 13 11C13 9.34315 11.6569 8 10 8C8.34315 8 7 9.34315 7 11C7 12.6569 8.34315 14 10 14ZM10 14C6.68629 14 4 16.6863 4 20C4 21.1046 4.89543 22 6 22H10" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </span>
          <input
            type="password"
            :value="apiKey"
            @input="emit('update:apiKey', $event.target.value)"
            :placeholder="connectionMode === 'ccr' ? '请输入 CCR API Key' : (selectedProviderCode === 'custom' ? '请输入你的 API Key' : `请输入 ${selectedProviderName} 的 API Key`)"
            class="text-input"
          />
        </div>
      </div>

      <div v-if="connectionMode === 'ccr'" class="ccr-notice">
        <strong>固定 Chat Completions 入站</strong>
        <p>Q-Solver 固定通过 Chat Completions 入站调用；实际 provider、协议转换和路由由 CCR 管理。模型目录同步不会调用模型推理。</p>
        <button type="button" class="continue-button" @click="emit('continue-to-model')">进入模型页并同步目录</button>
      </div>
    </section>

    <div class="status-bar">
      <span class="status-dot" :class="{ active: isReady }"></span>
      <span class="status-text">{{ isReady ? '接口配置已就绪' : '请选择模型提供商并填写 API Key' }}</span>
    </div>

    <Teleport to="body">
      <Transition name="dropdown">
        <div v-if="isOpen" class="provider-dropdown-portal" ref="dropdownRef" :style="dropdownStyle" @click.stop>
          <div
            v-for="provider in featuredProviders"
            :key="provider.code"
            class="dropdown-item"
            :class="{ selected: selectedProviderCode === provider.code }"
            @click.stop="selectProvider(provider.code)"
          >
            <div class="provider-logo">
              <div v-if="provider.logo.trim().startsWith('<svg')" v-html="provider.logo"></div>
              <img v-else :src="provider.logo" :alt="provider.name" />
            </div>
            <div class="dropdown-copy">
              <span class="dropdown-title">{{ provider.name }}</span>
              <span class="dropdown-subtitle">{{ provider.code === 'custom' ? '手动配置接口地址' : '自动填充默认地址' }}</span>
            </div>
            <span v-if="selectedProviderCode === provider.code" class="check-icon">✓</span>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import {
  FEATURED_PROVIDER_CODES,
  PROVIDERS,
  PROVIDER_BASE_URLS,
  getProviderLogoByCode,
  guessProviderFromBaseURL,
} from '../utils/modelCapabilities'

const props = defineProps({
  connectionMode: { type: String, default: 'direct' },
  apiKey: { type: String, default: '' },
  baseURL: { type: String, default: '' },
  apiProtocol: { type: String, default: 'openai_chat' },
})

const emit = defineEmits(['update:connectionMode', 'update:apiKey', 'update:baseURL', 'update:apiProtocol', 'select-provider', 'continue-to-model'])

const isOpen = ref(false)
const selectRef = ref(null)
const dropdownRef = ref(null)

const dropdownStyle = reactive({
  position: 'fixed',
  top: '0px',
  left: '0px',
  width: '0px',
  zIndex: 9999,
})

const featuredProviders = computed(() => FEATURED_PROVIDER_CODES.map((code) => ({
  code,
  name: PROVIDERS[code]?.name || code,
  logo: getProviderLogoByCode(code),
})))

const selectedProviderCode = ref(guessProviderFromBaseURL(props.baseURL))
const selectedProviderName = computed(() => props.connectionMode === 'ccr' ? 'CCR 统一网关' : (PROVIDERS[selectedProviderCode.value]?.name || '自定义'))
const protocolLabels = {
  openai_chat: 'CHAT COMPLETIONS',
  openai_responses: 'RESPONSES API',
  anthropic_messages: 'ANTHROPIC MESSAGES',
}
const selectedProtocolLabel = computed(() => props.connectionMode === 'ccr' ? 'CHAT COMPLETIONS VIA CCR' : (protocolLabels[props.apiProtocol] || '未识别协议'))
const selectedProviderLogo = computed(() => getProviderLogoByCode(selectedProviderCode.value))
const selectedProviderDescription = computed(() => {
  if (props.connectionMode === 'ccr') return '读取 CCR 模型目录后可直接切换路由；Q-Solver 不根据模型名猜测上游协议。'
  const descriptions = {
    openai: 'OpenAI 官方接口，适合直接接入 GPT 系列模型和标准兼容能力。',
    google: 'Google Gemini 的兼容接口，适合接入 Gemini 系列模型。',
    anthropic: 'Anthropic 官方服务；默认推荐原生 Messages，也可按网关能力独立选择其他协议。',
    deepseek: 'DeepSeek 兼容接口，适合理解、推理和代码类任务。',
    alibaba: '阿里云百炼兼容接口，可接入 Qwen 等模型服务。',
    moonshot: 'Moonshot AI 兼容接口，适合接入 Kimi 系列模型。',
    openrouter: 'OpenRouter 聚合了多家模型服务商，方便统一切换模型来源。',
    custom: '自定义 API root；协议由上方选项显式决定，不根据地址或模型名推断。',
  }
  return descriptions[selectedProviderCode.value] || '选择品牌预设后，仍可独立调整实际 API 协议。'
})
const isReady = computed(() => !!props.apiKey && ((props.connectionMode === 'ccr' || selectedProviderCode.value === 'custom') ? !!props.baseURL : true))

watch(() => props.baseURL, (newBaseURL) => {
  const normalized = (newBaseURL || '').trim()
  if (!normalized && selectedProviderCode.value === 'custom') return
  selectedProviderCode.value = guessProviderFromBaseURL(newBaseURL)
})

function updateDropdownPosition() {
  if (!selectRef.value) return
  const rect = selectRef.value.getBoundingClientRect()
  dropdownStyle.top = `${rect.bottom + 6}px`
  dropdownStyle.left = `${rect.left}px`
  dropdownStyle.width = `${rect.width}px`
}

function toggle() {
  isOpen.value = !isOpen.value
  if (!isOpen.value) return
  updateDropdownPosition()
  nextTick(() => {
    const selected = dropdownRef.value?.querySelector('.dropdown-item.selected')
    if (selected) selected.scrollIntoView({ block: 'nearest' })
  })
}

function changeConnectionMode(mode) {
  emit('update:connectionMode', mode)
  if (mode !== 'ccr') return
  emit('update:apiProtocol', 'openai_chat')
  const current = (props.baseURL || '').trim().replace(/\/$/, '')
  if (!current || current === 'https://api.openai.com/v1') {
    emit('update:baseURL', 'http://127.0.0.1:3456/v1')
  }
}

function selectProvider(providerCode) {
  selectedProviderCode.value = providerCode
  const recommendedProtocol = providerCode === 'anthropic' ? 'anthropic_messages' : 'openai_chat'
  emit('select-provider', {
    baseURL: PROVIDER_BASE_URLS[providerCode] ?? '',
    apiProtocol: recommendedProtocol,
  })
  isOpen.value = false
}

function handleClickOutside(event) {
  if (
    selectRef.value &&
    !selectRef.value.contains(event.target) &&
    dropdownRef.value &&
    !dropdownRef.value.contains(event.target)
  ) {
    isOpen.value = false
  }
}

function handleWindowChange() {
  if (isOpen.value) updateDropdownPosition()
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('resize', handleWindowChange)
  window.addEventListener('scroll', handleWindowChange, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('resize', handleWindowChange)
  window.removeEventListener('scroll', handleWindowChange, true)
})
</script>

<style scoped>
.provider-page {
  height: 100%;
  display: grid;
  grid-template-rows: auto auto auto;
  gap: var(--sp-4);
  align-content: start;
}

.overview-card,
.form-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-sm);
}

.overview-card {
  padding: var(--sp-4);
  display: grid;
  gap: var(--sp-3);
  background: linear-gradient(135deg, var(--surface-card) 0%, var(--surface-elevated) 100%);
}

.overview-header {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
}

.overview-logo {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--surface-elevated);
  box-shadow: var(--shadow-sm);
}

.overview-logo :deep(svg),
.overview-logo img {
  width: 30px;
  height: 30px;
  object-fit: contain;
}

.overview-copy {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.overview-tag {
  font-size: var(--text-xs);
  font-weight: var(--weight-bold);
  letter-spacing: 0.08em;
  color: var(--accent);
}

.overview-copy h3 {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
}

.overview-copy p {
  margin: 0;
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
  color: var(--text-secondary);
}

.provider-logo {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.provider-logo :deep(svg),
.provider-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.form-card {
  padding: var(--sp-4);
  display: grid;
  gap: var(--sp-4);
}

.form-item {
  display: grid;
  gap: var(--sp-2);
}

.field-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
}

.field-meta {
  color: var(--text-tertiary);
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  font-weight: var(--weight-normal);
}

.select-shell,
.input-shell {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  min-height: 46px;
  padding: 0 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: var(--surface-input);
  box-shadow: var(--shadow-sm);
  transition: all var(--duration-fast) ease;
}

.select-shell { cursor: pointer; }

.select-shell:hover,
.input-shell:hover {
  border-color: var(--border-default);
  background: var(--surface-card-hover);
}

.select-shell.open,
.input-shell:focus-within {
  border-color: var(--accent);
  box-shadow: var(--shadow-glow);
  background: var(--surface-card);
}

.select-leading,
.input-icon {
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.select-body,
.dropdown-copy {
  flex: 1;
  min-width: 0;
  display: grid;
  gap: 2px;
}

.select-title,
.dropdown-title {
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-subtitle,
.dropdown-subtitle {
  color: var(--text-tertiary);
  font-size: var(--text-xs);
}

.select-arrow {
  margin-left: auto;
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform var(--duration-fast) ease;
}

.select-arrow.rotated { transform: rotate(180deg); }

.protocol-select {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-sm);
}

.text-input {
  flex: 1;
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-family: var(--font-sans);
}

.text-input::placeholder { color: var(--text-muted); }

.field-hint {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.status-bar {
  min-height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  color: var(--text-tertiary);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  box-shadow: 0 0 8px var(--error-border);
}

.status-dot.active {
  background: var(--color-success);
  box-shadow: 0 0 8px var(--success-border);
}

.status-text { font-size: var(--text-xs); }

.ccr-logo {
  color: var(--accent);
  font-size: var(--text-sm);
  font-weight: var(--weight-bold);
  letter-spacing: 0.06em;
}

.ccr-notice {
  padding: var(--sp-4);
  border: 1px solid var(--accent-muted);
  border-radius: var(--radius-md);
  background: var(--accent-muted);
  color: var(--text-secondary);
}
.ccr-notice strong { color: var(--text-primary); }
.ccr-notice p { margin: var(--sp-2) 0 var(--sp-3); font-size: var(--text-sm); line-height: var(--leading-relaxed); }
.continue-button {
  min-height: 36px;
  padding: 0 var(--sp-4);
  border: 1px solid var(--accent);
  border-radius: var(--radius-sm);
  background: var(--accent);
  color: var(--text-inverse);
  cursor: pointer;
}

@media (max-height: 740px) {
  .provider-page { gap: var(--sp-3); }
  .overview-card,
  .form-card { padding: var(--sp-3); }
}
</style>

<style>
.provider-dropdown-portal {
  position: fixed;
  max-height: min(240px, 35vh);
  overflow-y: auto;
  background: var(--surface-popover);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-xl);
  backdrop-filter: blur(20px);
}

.provider-dropdown-portal::-webkit-scrollbar { width: 6px; }
.provider-dropdown-portal::-webkit-scrollbar-track { background: transparent; }
.provider-dropdown-portal::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
  border-radius: 3px;
}

.provider-dropdown-portal .dropdown-item {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-3) var(--sp-4);
  cursor: pointer;
  transition: background var(--duration-fast) ease;
  border-bottom: 1px solid var(--border-subtle);
}

.provider-dropdown-portal .dropdown-item:last-child { border-bottom: none; }
.provider-dropdown-portal .dropdown-item:hover { background: var(--surface-card-hover); }
.provider-dropdown-portal .dropdown-item.selected { background: var(--accent-muted); }

.provider-dropdown-portal .provider-logo {
  width: 22px;
  height: 22px;
}

.provider-dropdown-portal .provider-logo svg,
.provider-dropdown-portal .provider-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.provider-dropdown-portal .check-icon {
  margin-left: auto;
  color: var(--accent);
  font-weight: var(--weight-bold);
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all var(--duration-fast) ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
