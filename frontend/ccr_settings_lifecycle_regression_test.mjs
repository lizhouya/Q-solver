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
  apiKey: 'saved-key', baseURL: 'http://127.0.0.1:3456/v1', connectionMode: 'ccr',
  apiProtocol: 'openai_chat', endpointPath: '/chat/completions', extraHeaders: {},
  responseFormat: 'text', model: 'route-a', theme: 'light',
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
    getSettings: async () => ({ ...fixture, activeConnectionProfileId: 'profile-a' }),
    listConnectionProfiles: async () => ({ profiles: [{ id: 'profile-a', name: 'CCR', active: true, apiProtocol: 'openai_chat', model: fixture.model, apiKeySet: true }], error: null }),
    getConnectionProfile: async () => ({ success: true, profile: { id: 'profile-a', name: 'CCR', connection: { ...fixture } } }),
    deleteConnectionProfile: async id => ({ success: true, deletedId: id }),
    getDomainCategories: async () => [],
    syncSettings: async value => { calls.push(['full-save', JSON.parse(value)]); return '' },
    listModelsForDraft: async draft => { calls.push(['models', structuredClone(draft)]); return { models: ['route-a', 'route-b'], error: null } },
    probeModelConnection: async request => { calls.push(['probe', structuredClone(request)]); return { kind: request.kind, outcome: 'passed', protocol: request.connection.apiProtocol, message: '通过', requestSent: true } },
    saveConnectionSettings: async profileDraft => {
      calls.push(['connection-save', structuredClone(profileDraft)])
      const draft = profileDraft.connection
      return {
        success: true,
        profile: { id: profileDraft.id, name: profileDraft.name, active: true, apiProtocol: draft.apiProtocol, model: draft.model, apiKeySet: true },
        connection: { ...draft, connectionMode: 'direct', apiKeySet: true, extraHeadersSet: false },
      }
    },
    restoreFocus: async () => {}, removeFocus: async () => {}, ...overrides,
  }
  const currentTheme = ref('light')
  const context = vm.createContext({
    defineStore, reactive, ref, computed, watch, URL, api, useUIStore: () => ui, currentTheme,
    setTheme: value => { currentTheme.value = value }, navigator: { platform: 'Win32', userAgent: '' },
    window: { addEventListener() {} }, document: { documentElement: { style: { setProperty() {} }, getAttribute: () => 'light' } },
    getComputedStyle: () => ({ getPropertyValue: () => '' }), requestAnimationFrame: fn => fn(),
    structuredClone, console: { error() {} },
  })
  vm.runInContext(source + '\nglobalThis.store = useSettingsStore()', context)
  return { store: context.store, calls, ui }
}

const h = setup()
await h.store.loadSettings()
assert.equal(h.store.settings.connectionMode, 'direct', 'legacy CCR must be represented as direct')
await h.store.openSettings()
assert.equal(h.calls.length, 0, 'opening legacy CCR settings must not contact the gateway')

h.store.applyConnectionPreset({
  id: 'ccr', baseURL: 'http://127.0.0.1:3456/v1',
  apiProtocol: 'openai_chat', endpointPath: '/chat/completions', responseFormat: 'text',
})
assert.equal(h.store.tempSettings.connectionMode, 'direct')
assert.equal(h.calls.length, 0, 'CCR preset must only change the draft')

assert.equal(await h.store.refreshModels(), true)
assert.equal(h.calls.filter(call => call[0] === 'models').length, 1, 'catalog refresh must be explicit')
assert.deepEqual([...h.ui.availableModels], ['route-a', 'route-b'])

assert.equal(await h.store.testTextConnection(), true)
assert.equal(await h.store.testVisionConnection(), true)
assert.equal(h.calls.filter(call => call[0] === 'probe').length, 2, 'CCR preset uses normal probes')

assert.equal(await h.store.saveConnectionSettings(), true)
const saved = h.calls.find(call => call[0] === 'connection-save')[1]
assert.equal(saved.connection.connectionMode, 'direct')
assert.equal(saved.connection.apiProtocol, 'openai_chat')

h.ui.probeStatus.text = { outcome: 'passed' }
h.store.tempSettings.model = 'route-b'
await nextTick()
assert.deepEqual([...h.ui.availableModels], ['route-a', 'route-b'], 'model change must retain catalog')
assert.equal(h.ui.probeStatus.text, null, 'model change must invalidate probe')

h.store.tempSettings.apiProtocol = 'anthropic_messages'
await nextTick()
assert.equal(h.ui.availableModels.length, 0, 'protocol change must invalidate catalog')

console.log('CCR settings lifecycle regression tests passed')
