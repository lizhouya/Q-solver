import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { reactive, ref, computed, watch, nextTick } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/settings.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace('export const useSettingsStore', 'const useSettingsStore')

const fixture = {
  apiKey: 'saved-key', baseURL: 'https://api.openai.com/v1', connectionMode: 'direct', apiProtocol: 'openai_responses',
  model: 'manual-model', theme: 'light', windowWidth: 900,
}

function setup(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const ui = reactive({
    availableModels: [], isLoadingModels: false, showSettings: false,
    textProbeLoading: false, visionProbeLoading: false,
    probeStatus: { text: null, vision: null }, showToast() {},
  })
  const api = {
    getSettings: async () => ({ ...fixture }), getDomainCategories: async () => [],
    syncSettings: async value => { calls.push(['save', JSON.parse(value)]); return '' },
    listModelsForDraft: async draft => { calls.push(['models', { ...draft }]); return { models: ['suggested-model'], error: null } },
    probeModelConnection: async request => { calls.push(['probe', JSON.parse(JSON.stringify(request))]); return { kind: request.kind, outcome: 'passed', protocol: request.connection.apiProtocol, message: '通过', requestSent: true } },
    restoreFocus: async () => {}, removeFocus: async () => {}, ...overrides,
  }
  const currentTheme = ref('light')
  const context = vm.createContext({
    defineStore, reactive, ref, computed, watch, URL, api, useUIStore: () => ui, currentTheme,
    setTheme: value => { currentTheme.value = value }, navigator: { platform: 'Win32', userAgent: '' },
    window: { addEventListener() {} }, document: { documentElement: { style: { setProperty() {} }, getAttribute: () => 'light' } },
    getComputedStyle: () => ({ getPropertyValue: () => '' }), requestAnimationFrame: fn => fn(), console: { error() {} },
  })
  vm.runInContext(source + '\nglobalThis.store = useSettingsStore()', context)
  return { store: context.store, calls, ui, api }
}

let h = setup()
await h.store.loadSettings()
assert.equal(h.store.settings.apiProtocol, 'openai_responses', 'protocol must load from config')
await h.store.openSettings()
assert.equal(h.calls.length, 0, 'open must not list models or probe')
assert.equal(h.store.statusText, '已配置 / 未验证')
await h.store.refreshModels()
assert.equal(h.calls[0][1].apiProtocol, 'openai_responses')
assert.equal(h.store.tempSettings.model, 'manual-model', 'model suggestions must not replace manual model')
await h.store.testTextConnection()
await h.store.testVisionConnection()
assert.deepEqual(h.calls.slice(1).map(call => call[1].kind), ['text', 'vision'])
assert.equal(h.ui.probeStatus.text.outcome, 'passed')
assert.equal(h.ui.probeStatus.vision.outcome, 'passed')
await h.store.saveSettings({ close: false })
assert.equal(h.calls[3][1].apiProtocol, 'openai_responses', 'protocol must persist')
assert.equal(Object.hasOwn(h.calls[3][1], 'probeStatus'), false, 'probe state must not persist')

// 连接身份任一字段变化时，文字和图片结果同时失效。
h.store.tempSettings.model = 'changed-model'
await nextTick()
assert.equal(h.ui.probeStatus.text, null)
assert.equal(h.ui.probeStatus.vision, null)

// A -> B -> A 仍必须拒绝旧 A 的迟到结果。
let resolveProbe
h = setup({ probeModelConnection: request => new Promise(resolve => { resolveProbe = () => resolve({ kind: request.kind, outcome: 'passed', protocol: request.connection.apiProtocol, message: 'stale', requestSent: true }) }) })
await h.store.loadSettings()
await h.store.openSettings()
const pending = h.store.testTextConnection()
h.store.tempSettings.baseURL = 'https://b.invalid/v1'
h.store.tempSettings.baseURL = fixture.baseURL
await nextTick()
resolveProbe()
await pending
assert.equal(h.ui.probeStatus.text, null, 'stale A result must not survive A-B-A')

// 关闭再打开后，上个会话迟到结果无效。
let resolveModels
h = setup({ listModelsForDraft: () => new Promise(resolve => { resolveModels = resolve }) })
await h.store.loadSettings()
await h.store.openSettings()
const listPending = h.store.refreshModels()
h.store.closeSettings()
await h.store.openSettings()
resolveModels({ models: ['stale'], error: null })
await listPending
assert.equal(h.ui.availableModels.length, 0)

// 品牌预设只有切换 origin 才清除草稿凭据；协议随后仍可独立修改。
h = setup()
await h.store.loadSettings()
await h.store.openSettings()
h.store.applyProviderPreset({ baseURL: 'https://api.openai.com/v1', connectionMode: 'direct', apiProtocol: 'openai_chat' })
assert.equal(h.store.tempSettings.apiKey, 'saved-key', 'same origin should retain draft key')
h.store.applyProviderPreset({ baseURL: 'https://api.anthropic.com', apiProtocol: 'anthropic_messages' })
assert.equal(h.store.tempSettings.apiKey, '')
assert.equal(h.store.tempSettings.model, '')
h.store.tempSettings.apiProtocol = 'openai_chat'
assert.equal(h.store.tempSettings.apiProtocol, 'openai_chat', 'protocol must remain independently editable')

const provider = fs.readFileSync(new URL('./src/components/ProviderSelect.vue', import.meta.url), 'utf8')
assert.ok(provider.includes('Anthropic Messages'))
assert.ok(provider.includes('Responses API'))
assert.equal(provider.includes('OPENAI-COMPATIBLE'), false, 'provider header must reflect actual protocol')
const model = fs.readFileSync(new URL('./src/components/ModelSelect.vue', import.meta.url), 'utf8')
assert.ok(model.includes('<input'), 'model control must allow manual input')
assert.ok(model.includes('模型目录参考，非当前通道实测'))
const modal = fs.readFileSync(new URL('./src/components/SettingsModal.vue', import.meta.url), 'utf8')
assert.ok(modal.includes('每次点击会调用一次模型，可能产生少量费用'))
assert.ok(modal.includes('testTextConnection'))
assert.ok(modal.includes('testVisionConnection'))
console.log('Model connectivity regression tests passed')
