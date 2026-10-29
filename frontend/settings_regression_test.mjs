import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { reactive, ref, computed, watch, nextTick } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/settings.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace('export const useSettingsStore', 'const useSettingsStore')
const fixture = { apiKey: 'fixture-only', baseURL: 'http://localhost/v1', connectionMode: 'direct', apiProtocol: 'openai_chat', model: 'fixture-model', theme: 'light', windowWidth: 900, interruptThinking: true }
function setup(overrides = {}, platform = 'Win32') {
  setActivePinia(createPinia())
  const calls = []
  const ui = reactive({ availableModels: [], isLoadingModels: false, showSettings: false, textProbeLoading: false, visionProbeLoading: false, probeStatus: { text: null, vision: null }, showToast: () => {} })
  const api = {
    getSettings: async () => ({ ...fixture }), getDomainCategories: async () => [],
    syncSettings: async (value) => { calls.push(['save', JSON.parse(value)]); return '' },
    listModelsForDraft: async (draft) => { calls.push(['models', { ...draft }]); return { models: ['draft-model'], error: null } },
    probeModelConnection: async (request) => { calls.push(['probe', JSON.parse(JSON.stringify(request))]); return { kind: request.kind, outcome: 'passed', protocol: request.connection.apiProtocol, message: '通过', requestSent: true } },
    restoreFocus: async () => {}, removeFocus: async () => {}, ...overrides,
  }
  const currentTheme = ref('light')
  const context = vm.createContext({
    defineStore, reactive, ref, computed, watch, URL, api, useUIStore: () => ui, currentTheme,
    setTheme: (value) => { currentTheme.value = value },
    navigator: { platform, userAgent: '' }, window: { addEventListener() {} },
    document: { documentElement: { style: { setProperty() {} }, getAttribute: () => 'light' } },
    getComputedStyle: () => ({ getPropertyValue: () => '' }), requestAnimationFrame: (fn) => fn(),
    console: { error() {} },
  })
  vm.runInContext(source + '\nglobalThis.store = useSettingsStore()', context)
  return { store: context.store, calls, api, ui, currentTheme }
}
let h = setup({ getDomainCategories: async () => { throw Error('synthetic category failure') } })
await h.store.loadSettings()
assert.equal(h.store.settings.apiKey, fixture.apiKey, 'category failure must not discard config')
assert.equal(h.store.settingsLoaded, true)
assert.equal(h.calls.length, 0, 'loading must not call model API')

h = setup({ getSettings: async () => { throw Error('synthetic read failure') } })
await h.store.loadSettings()
assert.equal(h.store.settingsLoaded, false)
assert.ok(h.store.settingsLoadError)
await h.store.saveSettings()
await h.store.saveSettingsSilent()
assert.equal(h.calls.length, 0, 'unloaded defaults must not be saved')
h.api.getSettings = async () => ({ ...fixture })
await h.store.loadSettings()
assert.equal(h.store.settingsLoaded, true)
assert.equal(h.store.settingsLoadError, '')

h = setup()
await h.store.loadSettings()
await h.store.openSettings()
assert.equal(h.calls.length, 0, 'opening settings must not call API')
h.store.tempSettings.baseURL = 'http://localhost:9999/v1'
h.store.tempSettings.apiKey = 'draft-key'
await h.store.refreshModels()
assert.deepEqual(h.calls[0], ['models', { connectionMode: 'direct', apiKey: 'draft-key', baseURL: 'http://localhost:9999/v1', model: 'fixture-model', apiProtocol: 'openai_chat' }])
await h.store.testConnection()
assert.equal(h.calls[1][0], 'probe')
assert.equal(h.calls[1][1].kind, 'text')
assert.equal(h.calls[1][1].connection.model, 'fixture-model')
assert.equal(h.store.settings.baseURL, fixture.baseURL, 'testing must not change saved state')
await h.store.saveSettings({ close: false })
assert.equal(h.calls[2][1].windowWidth, 900, 'unexposed fields must survive save')
assert.equal(h.calls[2][1].interruptThinking, true)
for (const forbidden of ['token', 'url', 'generation', 'sessionId', 'clearSeq', 'stableLinkEnabled']) {
  assert.equal(Object.hasOwn(h.calls[2][1], forbidden), false, `ordinary settings must not contain ${forbidden}`)
}
assert.equal(h.ui.showSettings, true)

h = setup({ syncSettings: async () => 'synthetic save failure' })
await h.store.loadSettings()
await h.store.openSettings()
h.store.tempSettings.apiKey = 'not-saved'
assert.equal(await h.store.saveSettings(), false)
assert.equal(h.store.settings.apiKey, fixture.apiKey)

// 迟到模型列表不可覆盖用户随后修改的连接信息。
let resolveModels
h = setup({ listModelsForDraft: () => new Promise(resolve => { resolveModels = resolve }) })
await h.store.loadSettings()
await h.store.openSettings()
const pending = h.store.refreshModels()
h.store.tempSettings.baseURL = 'http://localhost:9998/v1'
resolveModels({ models: ['stale-model'], error: null })
await pending
assert.equal(h.ui.availableModels.length, 0)
assert.equal(h.store.tempSettings.model, fixture.model)

// Windows-only 快捷键只出现在 Windows，空绑定必须明确显示为未绑定。
h = setup()
assert.ok(h.store.shortcutActions.some(item => item.action === 'clear_mobile_display' && item.default === 'Shift+C'))
assert.ok(h.store.shortcutActions.some(item => item.action === 'start_next_exam' && item.label === '开始下一套' && item.default === 'Ctrl+Shift+N'))
assert.equal(h.store.shortcutDisplayName({ action: 'clear_mobile_display', default: 'Shift+C' }), 'Shift+C')
h.store.tempShortcuts.clear_mobile_display = { keyName: '', vkCode: '' }
assert.equal(h.store.shortcutDisplayName({ action: 'clear_mobile_display', default: 'Shift+C' }), '未绑定（点击录制）')
h.store.tempShortcuts.start_next_exam = { keyName: '', comboID: '' }
assert.equal(h.store.shortcutDisplayName({ action: 'start_next_exam', default: 'Ctrl+Shift+N' }), '未绑定（点击录制）')

h = setup({}, 'MacIntel')
assert.equal(h.store.shortcutActions.some(item => item.action === 'clear_mobile_display'), false, 'macOS 不显示 Windows 清屏 action')
assert.equal(h.store.shortcutActions.some(item => item.action === 'start_next_exam'), false, 'macOS 不显示 Windows 开始下一套 action')

const app = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
assert.ok(app.includes("on('shortcut-saved'"), 'successful recording must leave frontend recording state')
assert.ok(app.includes("on('shortcut-error'"), 'conflicting recording must restore the previous draft binding')
assert.ok(app.includes("on('request-clear-mobile-display'"), 'clear shortcut must request the shared confirmation flow')
assert.equal(app.includes('api.clearMobileDisplay(true)'), false, 'shortcut event must not bypass confirmation')

const modal = fs.readFileSync(new URL('./src/components/SettingsModal.vue', import.meta.url), 'utf8')
assert.ok(modal.includes('useMobileStore'), 'settings must use the shared Mobile store')
assert.equal(modal.includes('const mobileInfo = reactive'), false, 'settings must not keep a second Mobile state')
assert.ok(modal.includes("ui.showToast('复制失败，请手动复制已选中的连接地址', 'error')"), 'clipboard failure must be visible')
const start = modal.indexOf('async function saveAllSettings()')
const end = modal.indexOf('// Mobile Output', start)
for (const dirty of [false, true]) {
  let saves = 0, preferenceSaves = 0
  const context = vm.createContext({
    savingAll: { value: false },
    questionBank: { preferencesDirty: dirty, savePreferences: async () => { preferenceSaves++; return false } },
    settingsStore: { saveSettings: async () => { saves++; return true }, closeSettings() {} },
    ui: { activeTab: 'api', showToast() {} },
  })
  vm.runInContext(modal.slice(start, end), context)
  await vm.runInContext('saveAllSettings()', context)
  assert.equal(saves, 1, 'question bank failure must not block API save')
  assert.equal(preferenceSaves, dirty ? 1 : 0)
}
await nextTick()
console.log('Settings regression tests passed')
