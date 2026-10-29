<template>
  <div v-if="ui.showSettings" class="modal" id="settings-modal" style="display: flex">
    <div class="modal-content">
      <div class="modal-warning-banner">
        <Icon name="alert-triangle" :size="13" class="banner-icon" />
        <span>当前窗口已获得焦点，关闭设置后将自动恢复低打扰模式</span>
      </div>

      <div class="modal-header">
        <div class="tabs">
          <div class="tab" :class="{ active: ui.activeTab === 'general' }" @click="ui.activeTab = 'general'">常规</div>
          <div class="tab" :class="{ active: ui.activeTab === 'model' }" @click="ui.activeTab = 'model'">模型</div>
          <div class="tab" :class="{ active: ui.activeTab === 'screenshot' }" @click="ui.activeTab = 'screenshot'">截图</div>
          <div class="tab" :class="{ active: ui.activeTab === 'output' }" @click="ui.activeTab = 'output'">输出</div>
          <div class="tab" :class="{ active: ui.activeTab === 'questionbank' }" @click="ui.activeTab = 'questionbank'">题库</div>
          <div class="tab" :class="{ active: ui.activeTab === 'resume' }" @click="ui.activeTab = 'resume'">简历</div>
          <div class="tab" :class="{ active: ui.activeTab === 'api' }" @click="ui.activeTab = 'api'">API</div>
        </div>
        <button class="close-btn" @click="settingsStore.closeSettings">
          <Icon name="x" :size="16" />
        </button>
      </div>

      <div class="modal-body">
        <div v-show="ui.activeTab === 'general'" class="tab-pane">
          <div class="form-group">
            <label>快捷键配置 {{ settingsStore.isMacOS ? '(macOS 使用固定快捷键)' : '(点击录制)' }}</label>
            <div class="shortcut-list">
              <div class="shortcut-item" v-for="key in settingsStore.shortcutActions" :key="key.action">
                <span>{{ key.label }}</span>
                <button
                  class="btn-record"
                  :class="{ recording: settingsStore.recordingAction === key.action, disabled: settingsStore.isMacOS }"
                  @click="!settingsStore.isMacOS && settingsStore.recordKey(key.action)"
                >
                  {{ settingsStore.shortcutDisplayName(key) }}
                </button>
              </div>
            </div>
          </div>

          <div class="form-group">
            <label for="opacity-slider">窗口透明度 <span>{{ Math.round(settingsStore.tempSettings.transparency * 100) }}%</span></label>
            <input
              id="opacity-slider"
              v-model.number="settingsStore.tempSettings.transparency"
              type="range"
              min="0.0"
              max="1.0"
              step="0.05"
            />
          </div>
        </div>

        <div v-show="ui.activeTab === 'model'" class="tab-pane model-tab">
          <div class="form-group model-select-group">
            <div class="model-header">
              <label>模型名称</label>
              <button
                class="btn-secondary model-refresh"
                @click="settingsStore.refreshModels"
                :disabled="ui.isLoadingModels || !settingsStore.tempSettings.apiKey || !settingsStore.tempSettings.baseURL"
              >
                {{ ui.isLoadingModels ? '刷新中…' : '刷新模型建议' }}
              </button>
            </div>

            <ModelSelect
              v-model="settingsStore.tempSettings.model"
              :models="ui.availableModels"
              :loading="ui.isLoadingModels"
              :connection-mode="settingsStore.tempSettings.connectionMode"
            />
            <p v-if="!settingsStore.tempSettings.apiKey || !settingsStore.tempSettings.baseURL" class="hint-text warning-hint">请先填写 API Key 和接口地址</p>
          </div>

          <div v-if="settingsStore.tempSettings.connectionMode === 'ccr'" class="form-group ccr-model-notice">
            <div class="probe-header">
              <label>CCR 模型目录</label>
              <span>{{ ui.modelCatalog.status }}</span>
            </div>
            <p>这里只读取 CCR 的模型目录，不执行文字或图片实测。Q-Solver 固定通过 Chat Completions 入站调用，上游路由与协议转换由 CCR 管理。</p>
            <p class="catalog-status" :class="`catalog-${ui.modelCatalog.status}`">{{ ui.modelCatalog.message || '进入模型页后将自动同步一次；也可以手动刷新。' }}</p>
          </div>

          <div v-if="settingsStore.tempSettings.connectionMode !== 'ccr'" class="form-group probe-panel">
            <div class="probe-header">
              <label>当前通道实测</label>
              <span>{{ settingsStore.statusText }}</span>
            </div>
            <p class="probe-cost-notice">每次点击会调用一次模型，可能产生少量费用</p>
            <div class="probe-grid">
              <div class="probe-card">
                <div class="probe-card-header">
                  <strong>文字实测</strong>
                  <button
                    class="btn-secondary"
                    @click="settingsStore.testTextConnection"
                    :disabled="ui.textProbeLoading || !settingsStore.tempSettings.model"
                  >{{ ui.textProbeLoading ? '测试中…' : '开始测试' }}</button>
                </div>
                <p :class="probeOutcomeClass(ui.probeStatus.text)">{{ probeStatusText(ui.probeStatus.text, '文字') }}</p>
              </div>
              <div class="probe-card">
                <div class="probe-card-header">
                  <strong>图片实测</strong>
                  <button
                    class="btn-secondary"
                    @click="settingsStore.testVisionConnection"
                    :disabled="ui.visionProbeLoading || !settingsStore.tempSettings.model"
                  >{{ ui.visionProbeLoading ? '测试中…' : '开始测试' }}</button>
                </div>
                <p :class="probeOutcomeClass(ui.probeStatus.vision)">{{ probeStatusText(ui.probeStatus.vision, '图片') }}</p>
              </div>
            </div>
          </div>

          <div class="form-group domain-group">
            <DomainSelector v-model="settingsStore.tempSettings.domainId" :categories="settingsStore.domainCategories" />
          </div>
        </div>

        <div v-show="ui.activeTab === 'screenshot'" class="tab-pane">
          <ScreenshotSettings v-model="screenshotConfig" />
        </div>

        <div v-show="ui.activeTab === 'output'" class="tab-pane">
          <div class="form-group">
            <label>输出模式（谁看解题结果）</label>
            <div class="output-mode-grid">
              <label class="mode-option" :class="{ active: settingsStore.tempSettings.outputMode === 'desktop' }">
                <input type="radio" value="desktop" v-model="settingsStore.tempSettings.outputMode" />
                <span class="mode-title">💻 desktop</span>
                <span class="mode-desc">仅电脑端显示</span>
              </label>
              <label class="mode-option" :class="{ active: settingsStore.tempSettings.outputMode === 'mobile' }">
                <input type="radio" value="mobile" v-model="settingsStore.tempSettings.outputMode" />
                <span class="mode-title">📱 mobile</span>
                <span class="mode-desc">仅手机显示，电脑端隐藏答案</span>
              </label>
              <label class="mode-option" :class="{ active: settingsStore.tempSettings.outputMode === 'dual' }">
                <input type="radio" value="dual" v-model="settingsStore.tempSettings.outputMode" />
                <span class="mode-title">🖥️📱 dual</span>
                <span class="mode-desc">电脑和手机同时显示</span>
              </label>
            </div>
            <p class="hint-text">mobile 模式下，解题答案只推送到同一局域网内的手机浏览器，电脑不会渲染。</p>
            <p class="hint-text output-mode-state">当前生效：<b>{{ settingsStore.settings.outputMode }}</b></p>
            <p v-if="mobileModeDirty" class="hint-text warning-hint">本次选择 <b>{{ settingsStore.tempSettings.outputMode }}</b> 尚未保存；保存后才改变答案与滚动路由。</p>
          </div>

          <div class="form-group">
            <label>同步端连接</label>
            <div v-if="mobile.loading" class="empty-hint">连接信息加载中…</div>
            <div v-else-if="mobile.readError" class="mobile-unavailable">
              <p class="hint-text warning-hint">⚠️ 无法读取 Mobile 服务状态。这不代表端口冲突，可重新读取。</p>
              <button class="btn-secondary mobile-retry" :disabled="mobile.busy" @click="mobile.refresh">重新读取</button>
            </div>
            <div v-else-if="!mobile.info.available" class="mobile-unavailable">
              <p class="hint-text">⚠️ {{ mobileStatusMessage }}</p>
              <button v-if="mobile.info.retryable" class="btn-secondary mobile-retry" :disabled="mobile.busy" @click="mobile.retry">{{ mobile.mutating ? '重试中…' : '重试 Mobile 服务' }}</button>
            </div>
            <div v-else class="mobile-info">
              <div class="mobile-url-row">
                <input readonly :value="mobile.info.url" class="mobile-url" @click="$event.target.select()" />
                <button class="btn-copy" :disabled="mobile.busy" @click="copyMobileUrl">复制</button>
              </div>
              <ol class="mobile-steps">
                <li>在能访问本机地址的设备浏览器中打开上方链接</li>
                <li>等待解题后实时查看答案与分析</li>
                <li>网络 IP 变化时，Token 固定也可能导致完整 URL 变化</li>
              </ol>
              <p class="hint-text">此 HTTP 链接包含访问凭据；可达网络中的链接持有者可以查看同步内容。当前端口 <b>{{ mobile.info.port }}</b>。</p>
            </div>

            <div v-if="!mobile.loading && !mobile.readError" class="mobile-management">
              <div class="mobile-management-row">
                <div>
                  <strong>{{ mobile.info.stableLinkEnabled ? '固定链接已开启' : '固定链接已关闭' }}</strong>
                  <p>{{ mobile.info.stableLinkEnabled ? '完整重启后复用 Token；网络 IP 仍可能变化。' : '每次完整启动生成新临时 Token；同进程重试保持不变。' }}</p>
                </div>
                <button class="btn-secondary" :disabled="mobile.busy || !mobile.info.canResetLink" @click="mobile.requestStableMode(!mobile.info.stableLinkEnabled)">
                  {{ mobile.info.stableLinkEnabled ? '关闭固定链接' : '启用固定链接' }}
                </button>
              </div>
              <div class="mobile-action-row">
                <button class="btn-secondary" :disabled="mobile.busy || !mobile.info.canClear" @click="mobile.requestClear">清空同步端显示</button>
                <button class="btn-secondary danger-action" :disabled="mobile.busy || !mobile.info.canResetLink" @click="mobile.requestReset">重置同步链接</button>
              </div>
              <p class="hint-text">清屏不删除桌面答案、题库、截图、导出文件或模型上下文；重置链接会让旧链接和旧连接失效。</p>
            </div>
          </div>
        </div>

        <div v-show="ui.activeTab === 'questionbank'" class="tab-pane questionbank-tab">
          <div class="form-group">
            <label>默认题库目录</label>
            <div class="questionbank-path-row">
              <code class="questionbank-path">{{ questionBank.tempPreferences.archiveDir || '尚未选择' }}</code>
              <button
                class="btn-secondary"
                :disabled="questionBank.preferenceWorking || savingAll"
                @click="questionBank.selectOutput"
              >选择目录</button>
            </div>
            <p class="hint-text">每次打开导出窗口时默认使用这里；可在窗口中临时修改，且不会写回长期偏好。请选择已存在、可写且位于源码仓库外的目录。</p>
          </div>

          <div class="form-group questionbank-organize-card">
            <label class="questionbank-switch">
              <input
                v-model="questionBank.tempPreferences.autoOrganize"
                type="checkbox"
                :disabled="questionBank.preferenceWorking || savingAll"
              />
              <span>
                <strong>默认启用模型整理</strong>
                <small>仅作为导出窗口的默认值；每次都可以单独开关。</small>
              </span>
            </label>
            <div class="questionbank-data-notice" :class="{ enabled: questionBank.tempPreferences.autoOrganize }">
              <Icon name="alert-triangle" :size="15" />
              <p>
                只有在导出窗口中保持开启并确认提交后，才会把该分段的截图和原答案发送给已配置的模型服务商，可能产生额外调用费用；打开或取消窗口不会上传内容，也不会携带简历或其他历史会话。
                当前模型：<strong>{{ settingsStore.tempSettings.model || '未配置' }}</strong>
              </p>
            </div>
          </div>

          <p v-if="questionBank.error" class="questionbank-error" role="alert">{{ questionBank.error }}</p>
          <p v-else-if="questionBank.preferencesDirty" class="hint-text" role="status">题库偏好尚未保存；点击下方“保存”后与其他设置一起提交。</p>
        </div>

        <div v-if="ui.activeTab === 'resume'" class="tab-pane">
          <ResumeImport
            :resumePath="settingsStore.tempSettings.resumePath"
            :rawContent="settingsStore.resumeRawContent"
            :isParsing="settingsStore.isResumeParsing"
            @update:rawContent="val => settingsStore.resumeRawContent = val"
            @select-resume="settingsStore.selectResume"
            @clear-resume="settingsStore.clearResume"
            @parse-resume="settingsStore.parseResume"
          />
        </div>

        <div v-show="ui.activeTab === 'api'" class="tab-pane api-tab-pane">
          <ProviderSelect
            v-model:connectionMode="settingsStore.tempSettings.connectionMode"
            v-model:apiKey="settingsStore.tempSettings.apiKey"
            v-model:baseURL="settingsStore.tempSettings.baseURL"
            v-model:apiProtocol="settingsStore.tempSettings.apiProtocol"
            @select-provider="settingsStore.applyProviderPreset"
            @continue-to-model="continueToCCRModels"
          />
        </div>
      </div>

      <div v-if="settingsStore.settingsLoadError" class="form-group" role="alert">
        <p class="hint-text warning-hint">{{ settingsStore.settingsLoadError }}</p>
        <button class="btn-secondary" :disabled="settingsStore.settingsLoading" @click="settingsStore.loadSettings">重新读取配置</button>
      </div>
      <div class="modal-footer">
        <button class="btn-primary" :disabled="savingAll || !settingsStore.settingsLoaded || settingsStore.settingsLoading" @click="saveAllSettings">{{ savingAll ? '保存中…' : '保存' }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import { useQuestionBankStore } from '../stores/questionbank'
import { useMobileStore } from '../stores/mobile'
import ResumeImport from './ResumeImport.vue'
import ProviderSelect from './ProviderSelect.vue'
import ModelSelect from './ModelSelect.vue'
import DomainSelector from './DomainSelector.vue'
import ScreenshotSettings from './ScreenshotSettings.vue'
import Icon from './Icon.vue'

const ui = useUIStore()
const settingsStore = useSettingsStore()
const questionBank = useQuestionBankStore()
const mobile = useMobileStore()
const savingAll = ref(false)

watch(() => ui.showSettings, (visible) => {
  if (!visible) return
  questionBank.beginPreferenceEdit()
  mobile.refresh()
})

watch(() => ui.activeTab, (tab) => {
  if (ui.showSettings && tab === 'output') mobile.refresh()
})

watch(
  () => [ui.showSettings, ui.activeTab, settingsStore.tempSettings.connectionMode],
  ([visible, tab, mode]) => {
    if (visible && tab === 'model' && mode === 'ccr') settingsStore.ensureCCRModelCatalog()
  }
)

function continueToCCRModels() {
  ui.activeTab = 'model'
}

function probeStatusText(result, label) {
  if (!result) return `${label}：未测试`
  if (result.outcome === 'passed') return `${label}：通过`
  if (result.outcome === 'inconclusive') return `${label}：不确定 — ${result.message || '无法确认结果'}`
  return `${label}：失败 — ${result.message || '请检查连接配置'}`
}

function probeOutcomeClass(result) {
  return ['probe-result', result?.outcome || 'untested']
}

async function saveAllSettings() {
  if (savingAll.value) return
  savingAll.value = true
  try {
    let preferencesSaved = true
    if (questionBank.preferencesDirty) {
      try { preferencesSaved = await questionBank.savePreferences() } catch (_) { preferencesSaved = false }
    }
    const settingsSaved = await settingsStore.saveSettings({ close: false })
    if (!preferencesSaved) {
      ui.showToast(settingsSaved ? 'API 等常规设置已保存，但题库偏好保存失败，请检查题库页' : '常规设置与题库偏好均未保存，请分别检查', 'warning')
    } else if (settingsSaved) {
      settingsStore.closeSettings()
    }
  } finally {
    savingAll.value = false
  }
}

// Mobile Output 连接信息：服务状态与尚未保存的 outputMode 草稿彼此独立。
const mobileModeDirty = computed(() => (
  settingsStore.tempSettings.outputMode !== settingsStore.settings.outputMode
))

const mobileStatusMessage = computed(() => {
  switch (mobile.info.status) {
    case 'mobile_state_unavailable':
      return 'Mobile 专用状态不可用；为避免覆盖固定链接，服务未启动。'
    case 'port_unavailable':
      return 'Mobile 服务端口当前不可用；答案与滚动会安全回退到电脑端。'
    case 'initialization_failed':
      return 'Mobile 服务初始化失败；桌面功能不受影响。'
    case 'stopped':
      return 'Mobile 服务已停止，请重启应用后再试。'
    case 'not_started':
      return 'Mobile 服务尚未启动；可尝试重新启动服务。'
    default:
      return 'Mobile 服务当前不可用；答案与滚动会安全回退到电脑端。'
  }
})

async function copyMobileUrl() {
  if (!mobile.info.url) return
  try {
    if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(mobile.info.url)
    ui.showToast('连接地址已复制', 'success')
  } catch (_) {
    document.querySelector('.mobile-url')?.select()
    ui.showToast('复制失败，请手动复制已选中的连接地址', 'error')
  }
}

const screenshotConfig = computed({
  get: () => ({
    compressionQuality: settingsStore.tempSettings.compressionQuality,
    sharpening: settingsStore.tempSettings.sharpening,
    grayscale: settingsStore.tempSettings.grayscale,
    noCompression: settingsStore.tempSettings.noCompression,
    screenshotMode: settingsStore.tempSettings.screenshotMode,
  }),
  set: (val) => {
    settingsStore.tempSettings.compressionQuality = val.compressionQuality
    settingsStore.tempSettings.sharpening = val.sharpening
    settingsStore.tempSettings.grayscale = val.grayscale
    settingsStore.tempSettings.noCompression = val.noCompression
    settingsStore.tempSettings.screenshotMode = val.screenshotMode
  }
})
</script>

<style scoped>
.output-mode-state { margin-top: var(--sp-2); }
.mobile-retry { margin-top: var(--sp-2); }
.modal {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  pointer-events: auto;
}

.modal-content {
  width: 520px;
  max-width: 92vw;
  height: 580px;
  max-height: 85vh;
  background: var(--surface-popover);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: modalIn 0.25s var(--ease-out);
}

@keyframes modalIn {
  from { opacity: 0; transform: scale(0.96) translateY(8px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.modal-warning-banner {
  background: var(--warning-bg);
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-full);
  padding: 6px 16px;
  color: var(--color-warning);
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-1-5);
  margin: 12px auto 4px auto;
  width: fit-content;
}

.banner-icon { flex-shrink: 0; }

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-3) var(--sp-5);
  border-bottom: 1px solid var(--border-subtle);
}

.tabs {
  display: flex;
  gap: var(--sp-1);
  min-width: 0;
  overflow-x: auto;
}

.tab {
  padding: var(--sp-2) var(--sp-4);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.tab:hover {
  color: var(--text-primary);
  background: var(--surface-card);
}

.tab.active {
  color: var(--accent);
  background: var(--accent-muted);
}

.close-btn {
  color: var(--text-muted);
  cursor: pointer;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  transition: all var(--duration-fast) ease;
}

.close-btn:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}

.modal-body {
  flex: 1;
  overflow: hidden;
  padding: 0;
  min-height: 0;
  position: relative;
}

.tab-pane {
  height: 100%;
  overflow-y: auto;
  padding: var(--sp-5);
  box-sizing: border-box;
}

.api-tab-pane {
  overflow-y: auto;
  display: block;
}

.modal-footer {
  padding: var(--sp-4) var(--sp-5);
  border-top: 1px solid var(--border-subtle);
  display: flex;
  justify-content: flex-end;
}

.btn-primary {
  padding: var(--sp-2) var(--sp-6);
  border-radius: var(--radius-md);
  background: var(--accent);
  color: var(--text-inverse);
  font-size: var(--text-sm);
  font-weight: 700;
  border: none;
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.btn-primary:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
}

.form-group { margin-bottom: var(--sp-5); }

.form-group label {
  display: block;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--sp-2);
}

.model-tab {
  display: block;
  height: 100%;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
}

.model-select-group { flex-shrink: 0; }
.model-refresh { min-height: 30px; padding: 0 var(--sp-3); }
.ccr-model-notice {
  padding: var(--sp-3);
  border: 1px solid var(--accent-muted);
  border-radius: var(--radius-md);
  background: var(--accent-muted);
}
.ccr-model-notice p { margin: var(--sp-2) 0 0; color: var(--text-secondary); font-size: var(--text-xs); line-height: 1.6; }
.ccr-model-notice .catalog-status { color: var(--text-primary); overflow-wrap: anywhere; }
.ccr-model-notice .catalog-failed,
.ccr-model-notice .catalog-ui_timeout { color: var(--color-error); }
.ccr-model-notice .catalog-empty,
.ccr-model-notice .catalog-needs_config { color: var(--color-warning); }
.ccr-model-notice .catalog-synced { color: var(--color-success); }

.probe-panel {
  padding: var(--sp-3);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
}
.probe-header,
.probe-card-header { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3); }
.probe-header label { margin: 0; }
.probe-header span { color: var(--text-tertiary); font-size: var(--text-xs); }
.probe-cost-notice { margin: var(--sp-2) 0 var(--sp-3); color: var(--color-warning); font-size: var(--text-xs); }
.probe-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--sp-2); }
.probe-card { padding: var(--sp-3); border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); background: var(--surface-input); }
.probe-card strong { color: var(--text-primary); font-size: var(--text-sm); }
.probe-card .btn-secondary { min-height: 30px; padding: 0 var(--sp-3); }
.probe-result { margin: var(--sp-2) 0 0; font-size: var(--text-xs); line-height: 1.5; overflow-wrap: anywhere; }
.probe-result.untested { color: var(--text-muted); }
.probe-result.passed { color: var(--color-success); }
.probe-result.failed { color: var(--color-error); }
.probe-result.inconclusive { color: var(--color-warning); }

.domain-group {
  display: block;
  flex: none;
  min-height: auto;
}

:deep(.domain-selector) {
  display: block;
  height: auto;
  min-height: 0;
}

:deep(.domain-list) {
  flex: none;
  max-height: none;
  overflow: visible;
}

.model-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-2);
}

.model-actions {
  display: flex;
  gap: var(--sp-2);
}

.btn-icon {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.btn-icon:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
  border-color: var(--border-hover);
}

.btn-icon:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.action-icon {
  width: 14px;
  height: 14px;
}

.action-icon.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.connection-status {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  margin-top: var(--sp-2);
}

.connection-status.success {
  background: var(--success-bg);
  border: 1px solid var(--success-border);
  color: var(--color-success);
}

.connection-status.error {
  background: var(--error-bg);
  border: 1px solid var(--error-border);
  color: var(--color-error);
}

.cs-icon {
  font-size: 14px;
  font-weight: 700;
}

.cs-text { font-weight: 600; }

.hint-text {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: var(--sp-2);
}

.warning-hint { color: var(--color-warning); }

.questionbank-tab { display: block; }
.questionbank-path-row {
  display: flex;
  align-items: stretch;
  gap: var(--sp-2);
}
.questionbank-path {
  flex: 1;
  min-width: 0;
  min-height: 36px;
  display: flex;
  align-items: center;
  padding: var(--sp-2) var(--sp-3);
  overflow-wrap: anywhere;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-input);
  color: var(--text-secondary);
  font-size: var(--text-xs);
}
.btn-secondary {
  flex-shrink: 0;
  padding: 0 var(--sp-4);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-card);
  color: var(--text-primary);
  cursor: pointer;
}
.btn-secondary:hover:not(:disabled) { background: var(--surface-card-hover); }
.btn-secondary:disabled,
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }
.settings-text-input {
  width: 100%;
  box-sizing: border-box;
  padding: var(--sp-2) var(--sp-3);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-input);
  color: var(--text-primary);
  font-size: var(--text-sm);
}
.settings-text-input:focus,
.btn-secondary:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.questionbank-organize-card {
  padding: var(--sp-4);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
}
.form-group .questionbank-switch {
  display: flex;
  align-items: flex-start;
  gap: var(--sp-3);
  margin: 0;
  cursor: pointer;
}
.questionbank-switch input { margin-top: 3px; accent-color: var(--accent); }
.questionbank-switch span { display: flex; flex-direction: column; gap: 4px; }
.questionbank-switch strong { color: var(--text-primary); font-size: var(--text-sm); }
.questionbank-switch small { color: var(--text-muted); font-size: var(--text-xs); font-weight: 400; line-height: 1.5; }
.questionbank-data-notice {
  display: flex;
  align-items: flex-start;
  gap: var(--sp-2);
  margin-top: var(--sp-3);
  padding: var(--sp-3);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  background: var(--surface-input);
}
.questionbank-data-notice.enabled {
  border-color: var(--warning-border);
  color: var(--color-warning);
  background: var(--warning-bg);
}
.questionbank-data-notice p { margin: 0; font-size: var(--text-xs); line-height: 1.65; }
.questionbank-error {
  padding: var(--sp-3);
  border: 1px solid var(--error-border);
  border-radius: var(--radius-sm);
  background: var(--error-bg);
  color: var(--color-error);
  font-size: var(--text-xs);
  overflow-wrap: anywhere;
}

.mobile-management {
  margin-top: var(--sp-3);
  padding: var(--sp-3);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: var(--surface-card);
}
.mobile-management-row { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3); }
.mobile-management-row strong { color: var(--text-primary); font-size: var(--text-sm); }
.mobile-management-row p { margin: 4px 0 0; color: var(--text-muted); font-size: var(--text-xs); line-height: 1.5; }
.mobile-action-row { display: flex; gap: var(--sp-2); margin-top: var(--sp-3); }
.danger-action { color: var(--color-warning); border-color: var(--warning-border); }

/* ---- 001 · 输出模式 + 手机连接 ---- */
.output-mode-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-2);
}
.mode-option {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  padding: var(--sp-3);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
  transition: all var(--duration-fast) ease;
  background: var(--surface-card);
}
.mode-option input { display: none; }
.mode-option.active {
  border-color: var(--accent);
  background: var(--accent-muted);
  box-shadow: 0 0 0 1px var(--accent);
}
.mode-title {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}
.mode-desc {
  font-size: var(--text-xs);
  color: var(--text-muted);
  line-height: 1.4;
}
.mobile-info {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  padding: var(--sp-3);
  background: var(--surface-card);
}
.mobile-url-row { display: flex; gap: var(--sp-2); }
.mobile-url {
  flex: 1;
  font-family: var(--font-mono);
  font-size: 12px;
  padding: var(--sp-2) var(--sp-3);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
  background: var(--surface-input);
  color: var(--text-secondary);
  min-width: 0;
}
.btn-copy {
  padding: 0 var(--sp-4);
  border-radius: var(--radius-sm);
  background: var(--accent);
  color: white;
  font-size: var(--text-xs);
  font-weight: 700;
  border: none;
  cursor: pointer;
  white-space: nowrap;
}
.btn-copy:hover { background: var(--accent-hover); }
.mobile-steps {
  margin: var(--sp-3) 0 0;
  padding-left: 1.2em;
  font-size: var(--text-xs);
  color: var(--text-secondary);
  line-height: 1.8;
}
.mobile-unavailable .hint-text {
  margin-top: 0;
  padding: var(--sp-3);
  border: 1px dashed var(--border-default);
  border-radius: var(--radius-md);
}
.empty-hint {
  font-size: var(--text-xs);
  color: var(--text-muted);
  padding: var(--sp-3);
}

.shortcut-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--sp-2);
}

.shortcut-item:last-child:nth-child(odd) { grid-column: 1 / -1; }

.shortcut-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--text-primary);
  gap: var(--sp-2);
}

.btn-record {
  padding: var(--sp-1) var(--sp-3);
  font-size: var(--text-xs);
  font-family: var(--font-mono);
  font-weight: 600;
  background: var(--surface-input);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xs);
  color: var(--text-secondary);
  cursor: pointer;
  min-width: 70px;
  text-align: center;
  transition: all var(--duration-fast) ease;
}

.btn-record:hover:not(.disabled) {
  border-color: var(--accent-border);
  color: var(--accent);
}

.btn-record.recording {
  border-color: var(--accent);
  background: var(--accent-muted);
  color: var(--accent);
  animation: pulseRecord 1s ease-in-out infinite;
}

.btn-record.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@keyframes pulseRecord {
  0%, 100% { box-shadow: 0 0 0 0 var(--accent-glow); }
  50% { box-shadow: 0 0 0 4px var(--accent-glow); }
}

input[type="range"] {
  width: 100%;
  height: 4px;
  appearance: none;
  -webkit-appearance: none;
  background: var(--border-default);
  border-radius: var(--radius-full);
  outline: none;
  cursor: pointer;
}

input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--accent);
  border: 2px solid var(--surface-elevated);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
}
</style>
