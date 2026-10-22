import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { reactive, ref, computed, watch, nextTick } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/settings.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace('export const useSettingsStore', 'const useSettingsStore')
const fixture = { apiKey: 'fixture-only', baseURL: 'http://localhost/v1', model: 'fixture-model', theme: 'light', windowWidth: 900, interruptThinking: true }
function setup(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const ui = reactive({ availableModels: [], isLoadingModels: false, showSettings: false, showToast: () => {} })
  const api = {
    getSettings: async () => ({ ...fixture }), getDomainCategories: async () => [],
    syncSettings: async (value) => { calls.push(['save', JSON.parse(value)]); return '' },
    getModels: async (...args) => { calls.push(['models', ...args]); return ['draft-model'] },
    testConnection: async (...args) => { calls.push(['test', ...args]); return '' },
    restoreFocus: async () => {}, removeFocus: async () => {}, ...overrides,
  }
  const currentTheme = ref('light')
  const context = vm.createContext({
    defineStore, reactive, ref, computed, watch, api, useUIStore: () => ui, currentTheme,
    setTheme: (value) => { currentTheme.value = value },
    navigator: { platform: 'Win32', userAgent: '' }, window: { addEventListener() {} },
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
assert.deepEqual(h.calls[0], ['models', 'draft-key', 'http://localhost:9999/v1'])
await h.store.testConnection()
assert.deepEqual(h.calls[1], ['test', 'draft-key', 'http://localhost:9999/v1', 'draft-model'])
assert.equal(h.store.settings.baseURL, fixture.baseURL, 'testing must not change saved state')
await h.store.saveSettings({ close: false })
assert.equal(h.calls[2][1].windowWidth, 900, 'unexposed fields must survive save')
assert.equal(h.calls[2][1].interruptThinking, true)
assert.equal(h.ui.showSettings, true)

h = setup({ syncSettings: async () => 'synthetic save failure' })
await h.store.loadSettings()
await h.store.openSettings()
h.store.tempSettings.apiKey = 'not-saved'
assert.equal(await h.store.saveSettings(), false)
assert.equal(h.store.settings.apiKey, fixture.apiKey)

// 迟到模型列表不可覆盖用户随后修改的连接信息。
let resolveModels
h = setup({ getModels: () => new Promise(resolve => { resolveModels = resolve }) })
await h.store.loadSettings()
await h.store.openSettings()
const pending = h.store.refreshModels()
h.store.tempSettings.baseURL = 'http://localhost:9998/v1'
resolveModels(['stale-model'])
await pending
assert.equal(h.ui.availableModels.length, 0)
assert.equal(h.store.tempSettings.model, fixture.model)

const modal = fs.readFileSync(new URL('./src/components/SettingsModal.vue', import.meta.url), 'utf8')
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
