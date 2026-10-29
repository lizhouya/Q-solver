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
  apiKey: 'saved-key', baseURL: 'https://api.openai.com/v1', apiProtocol: 'openai_responses',
  model: 'saved-model', theme: 'light', windowWidth: 900,
}

function setup(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const ui = reactive({
    availableModels: [], isLoadingModels: false, showSettings: false,
    textProbeLoading: false, visionProbeLoading: false,
    probeStatus: { text: null, vision: null },
    modelCatalog: { status: 'idle', message: '', errorCode: '', lastSyncedAt: 0 },
    showToast() {},
  })
  const api = {
    getSettings: async () => ({ ...fixture }), getDomainCategories: async () => [],
    syncSettings: async value => { calls.push(['save', JSON.parse(value)]); return '' },
    listModelsForDraft: async draft => { calls.push(['models', { ...draft }]); return { models: ['route-a'], error: null } },
    probeModelConnection: async request => { calls.push(['probe', request]); return { kind: request.kind, outcome: 'passed', message: '通过', requestSent: true } },
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
  return { store: context.store, calls, ui }
}

let h = setup()
await h.store.loadSettings()
assert.equal(h.store.settings.connectionMode, 'direct', 'legacy settings must load as direct')
await h.store.openSettings()
assert.equal(await h.store.ensureCCRModelCatalog(), false, 'direct mode must not auto-sync CCR catalog')
assert.equal(h.calls.filter(call => call[0] === 'models').length, 0)

let catalogCalls = 0
let resolveFirst
h = setup({
  getSettings: async () => ({ ...fixture, connectionMode: 'ccr', baseURL: 'http://127.0.0.1:3456/v1', apiProtocol: 'openai_chat' }),
  listModelsForDraft: draft => {
    catalogCalls++
    if (catalogCalls === 1) return new Promise(resolve => { resolveFirst = () => resolve({ models: ['route-a', 'route-b'], error: null }) })
    return Promise.resolve({ models: ['route-a', 'route-b', 'route-c'], error: null })
  },
})
await h.store.loadSettings()
await h.store.openSettings()
const first = h.store.ensureCCRModelCatalog()
const reused = h.store.ensureCCRModelCatalog()
assert.equal(first, reused, 'same session/revision must reuse one in-flight catalog Promise')
assert.equal(catalogCalls, 1, 'CCR auto-sync must be single-flight')
resolveFirst()
assert.equal(await first, true)
assert.deepEqual([...h.ui.availableModels], ['route-a', 'route-b'])
assert.equal(h.ui.modelCatalog.status, 'synced')
assert.equal(await h.store.ensureCCRModelCatalog(), true)
assert.equal(catalogCalls, 1, 'successful catalog must not auto-repeat in one session/revision')

h.ui.probeStatus.text = { outcome: 'passed' }
h.store.tempSettings.model = 'route-b'
await nextTick()
assert.deepEqual([...h.ui.availableModels], ['route-a', 'route-b'], 'changing model must not clear catalog')
assert.equal(h.ui.probeStatus.text, null, 'changing model must invalidate probe only')
h.store.tempSettings.apiProtocol = 'anthropic_messages'
await nextTick()
assert.deepEqual([...h.ui.availableModels], ['route-a', 'route-b'], 'residual CCR protocol must not change catalog identity')

const probesBefore = h.calls.filter(call => call[0] === 'probe').length
assert.equal(await h.store.testTextConnection(), false)
assert.equal(await h.store.testVisionConnection(), false)
assert.equal(h.calls.filter(call => call[0] === 'probe').length, probesBefore, 'CCR store probes must make zero bridge calls')

await h.store.refreshModels()
assert.equal(catalogCalls, 2, 'manual refresh must start a new catalog request')
assert.deepEqual([...h.ui.availableModels], ['route-a', 'route-b', 'route-c'])

const beforeSaveCatalogCalls = catalogCalls
await h.store.saveSettings({ close: false })
const saved = h.calls.find(call => call[0] === 'save')[1]
assert.equal(saved.connectionMode, 'ccr')
assert.equal(saved.apiProtocol, 'openai_chat', 'CCR save must persist effective Chat protocol')
assert.equal(catalogCalls, beforeSaveCatalogCalls, 'saving must not trigger catalog sync')

h.store.tempSettings.apiKey = 'changed-key'
await nextTick()
assert.equal(h.ui.availableModels.length, 0, 'catalog identity change must clear catalog')
assert.equal(h.ui.modelCatalog.status, 'idle')

console.log('CCR settings lifecycle regression tests passed')
