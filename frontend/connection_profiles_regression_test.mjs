import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { reactive, ref, computed, watch, nextTick } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/settings.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace('export const useSettingsStore', 'const useSettingsStore')
const clone = value => JSON.parse(JSON.stringify(value))

const globalFixture = {
  activeConnectionProfileId: 'profile-a',
  outputMode: 'dual', resumeContent: 'keep-resume', theme: 'light',
  shortcuts: {}, domainId: 'general-assistant', opacity: 1,
}

const profileA = {
  id: 'profile-a', name: 'A 连接',
  connection: {
    connectionMode: 'direct', apiKey: 'test-a', baseURL: 'https://a.example.test/v1',
    apiProtocol: 'openai_chat', endpointPath: '/chat/completions', extraHeaders: {},
    responseFormat: 'text', requestTemplate: '', responseJSONPointer: '',
    authHeader: '', authPrefix: '', model: 'model-a',
  },
}

const profileB = {
  id: 'profile-b', name: 'B 连接',
  connection: {
    ...profileA.connection, apiKey: '', baseURL: 'https://b.example.test/v1', model: 'model-b',
  },
}

function setup(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const profiles = { 'profile-a': clone(profileA), 'profile-b': clone(profileB) }
  let activeID = 'profile-a'
  const summaries = () => Object.values(profiles).map(profile => ({
    id: profile.id, name: profile.name,
    addressLabel: new URL(profile.connection.baseURL).hostname,
    apiProtocol: profile.connection.apiProtocol, model: profile.connection.model,
    active: profile.id === activeID,
    apiKeySet: Boolean(profile.connection.apiKey),
    templateSet: Boolean(profile.connection.requestTemplate),
  }))
  const ui = reactive({
    availableModels: [], isLoadingModels: false, showSettings: false,
    textProbeLoading: false, visionProbeLoading: false,
    probeStatus: { text: null, vision: null },
    modelCatalog: { status: 'idle', message: '', errorCode: '', lastSyncedAt: 0 },
    showToast() {},
  })
  const api = {
    getSettings: async () => clone({ ...globalFixture, activeConnectionProfileId: activeID }),
    listConnectionProfiles: async () => {
      calls.push(['list-profiles'])
      return { profiles: clone(summaries()), error: null }
    },
    getConnectionProfile: async id => {
      calls.push(['get-profile', id])
      return { success: true, profile: clone(profiles[id]) }
    },
    saveConnectionSettings: async draft => {
      calls.push(['save-profile', clone(draft)])
      const id = draft.id || 'generated-profile-id'
      const name = draft.name || 'new.example.test · OpenAI Chat'
      profiles[id] = { id, name, connection: clone(draft.connection) }
      activeID = id
      return {
        success: true,
        profile: {
          id, name, addressLabel: new URL(draft.connection.baseURL).hostname,
          apiProtocol: draft.connection.apiProtocol, model: draft.connection.model,
          active: true, apiKeySet: Boolean(draft.connection.apiKey),
          templateSet: Boolean(draft.connection.requestTemplate),
        },
        connection: {
          connectionMode: 'direct', baseURL: draft.connection.baseURL,
          model: draft.connection.model, apiProtocol: draft.connection.apiProtocol,
          endpointPath: draft.connection.endpointPath, responseFormat: draft.connection.responseFormat,
          apiKeySet: Boolean(draft.connection.apiKey),
          extraHeadersSet: Object.keys(draft.connection.extraHeaders || {}).length > 0,
          templateSet: Boolean(draft.connection.requestTemplate),
        },
      }
    },
    deleteConnectionProfile: async id => {
      calls.push(['delete-profile', id])
      delete profiles[id]
      return { success: true, deletedId: id }
    },
    syncSettings: async value => {
      const payload = JSON.parse(value)
      calls.push(['full-save', payload])
      const id = payload.connectionProfileId || 'generated-full-id'
      const name = payload.connectionProfileName || 'full.example.test · OpenAI Chat'
      profiles[id] = {
        id, name,
        connection: {
          connectionMode: 'direct', apiKey: payload.apiKey, baseURL: payload.baseURL,
          apiProtocol: payload.apiProtocol, endpointPath: payload.endpointPath,
          extraHeaders: payload.extraHeaders, responseFormat: payload.responseFormat,
          requestTemplate: payload.requestTemplate, responseJSONPointer: payload.responseJSONPointer,
          authHeader: payload.authHeader, authPrefix: payload.authPrefix, model: payload.model,
        },
      }
      activeID = id
      return ''
    },
    getDomainCategories: async () => [],
    listModelsForDraft: async draft => { calls.push(['models', clone(draft)]); return { models: ['model-a'], error: null } },
    probeModelConnection: async request => { calls.push(['probe', clone(request)]); return { outcome: 'passed', kind: request.kind } },
    restoreFocus: async () => {}, removeFocus: async () => {}, stopRecordingKey: async () => {},
    ...overrides,
  }
  const currentTheme = ref('light')
  const context = vm.createContext({
    defineStore, reactive, ref, computed, watch, URL, api, useUIStore: () => ui, currentTheme,
    setTheme: value => { currentTheme.value = value }, navigator: { platform: 'Win32', userAgent: '' },
    window: { addEventListener() {} },
    document: { documentElement: { style: { setProperty() {} }, getAttribute: () => 'light' } },
    getComputedStyle: () => ({ getPropertyValue: () => '' }), requestAnimationFrame: fn => fn(),
    console: { error() {} },
  })
  vm.runInContext(source + '\nglobalThis.store = useSettingsStore()', context)
  return { store: context.store, calls, ui, profiles, getActiveID: () => activeID }
}

let h = setup()
assert.equal(await h.store.loadSettings(), true)
assert.equal(h.store.activeConnectionProfileID, 'profile-a')
assert.equal(h.store.selectedConnectionProfileID, 'profile-a')
assert.equal(h.store.settings.apiKey, 'test-a')
assert.equal(h.store.tempSettings.apiKey, 'test-a')
assert.equal(h.calls.filter(call => call[0] === 'get-profile').length, 1)

const beforeSelectCalls = h.calls.length
assert.equal(await h.store.selectConnectionProfile('profile-b'), true)
assert.equal(h.store.selectedConnectionProfileID, 'profile-b')
assert.equal(h.store.tempSettings.baseURL, profileB.connection.baseURL)
assert.equal(h.store.settings.baseURL, profileA.connection.baseURL, 'selection must not activate profile')
assert.deepEqual(h.calls.slice(beforeSelectCalls).map(call => call[0]), ['get-profile'])

assert.equal(await h.store.saveSettingsSilent(), true)
const silentPayload = h.calls.filter(call => call[0] === 'full-save').at(-1)[1]
assert.equal(silentPayload.connectionProfileId, 'profile-a', 'silent save must use active profile')
assert.equal(silentPayload.baseURL, profileA.connection.baseURL)

h.store.beginNewConnectionProfile()
assert.equal(h.store.selectedConnectionProfileID, '')
assert.equal(h.store.connectionProfileName, '')
assert.equal(h.store.connectionProfileLoaded, true)
assert.equal(h.store.tempSettings.apiKey, '')
assert.equal(h.store.tempSettings.baseURL, '')
assert.equal(h.store.tempSettings.model, '')
h.store.tempSettings.baseURL = 'https://new.example.test/v1'
h.store.tempSettings.model = 'new-model'
h.store.tempSettings.apiKey = 'new-test-key'
assert.equal(await h.store.saveConnectionSettings(), true)
assert.equal(h.store.activeConnectionProfileID, 'generated-profile-id')
assert.equal(h.store.selectedConnectionProfileID, 'generated-profile-id')
assert.equal(h.store.connectionProfileName, 'new.example.test · OpenAI Chat')
assert.equal(h.store.settings.apiKey, 'new-test-key')
const connectionSave = h.calls.find(call => call[0] === 'save-profile')[1]
assert.equal(connectionSave.id, '')
assert.equal(connectionSave.name, '')
assert.equal(connectionSave.connection.baseURL, 'https://new.example.test/v1')

h = setup()
await h.store.loadSettings()
await h.store.selectConnectionProfile('profile-b')
h.store.tempSettings.model = 'model-b-updated'
assert.equal(await h.store.saveSettings({ close: false }), true)
const fullPayload = h.calls.find(call => call[0] === 'full-save')[1]
assert.equal(fullPayload.connectionProfileId, 'profile-b')
assert.equal(fullPayload.connectionProfileName, 'B 连接')
assert.equal(Object.hasOwn(fullPayload, 'connectionProfiles'), false)
assert.equal(Object.hasOwn(fullPayload, 'modelHistory'), false)
assert.equal(h.store.activeConnectionProfileID, 'profile-b')

h = setup()
await h.store.loadSettings()
h.store.requestDeleteConnectionProfile('profile-a')
assert.equal(h.store.pendingDeleteProfileID, '', 'active profile deletion must be blocked')
h.store.requestDeleteConnectionProfile('profile-b')
assert.equal(h.store.pendingDeleteProfileID, 'profile-b')
assert.equal(await h.store.confirmDeleteConnectionProfile(), true)
assert.equal(h.calls.some(call => call[0] === 'delete-profile' && call[1] === 'profile-b'), true)
assert.equal(h.store.connectionProfiles.some(profile => profile.id === 'profile-b'), false)

let resolveProfile
h = setup({
  getConnectionProfile: id => {
    if (id === 'profile-b') return new Promise(resolve => { resolveProfile = resolve })
    return Promise.resolve({ success: true, profile: clone(profileA) })
  },
})
await h.store.loadSettings()
const staleSelection = h.store.selectConnectionProfile('profile-b')
await nextTick()
await h.store.selectConnectionProfile('profile-a')
resolveProfile({ success: true, profile: clone(profileB) })
assert.equal(await staleSelection, false)
assert.equal(h.store.selectedConnectionProfileID, 'profile-a')
assert.equal(h.store.tempSettings.baseURL, profileA.connection.baseURL)

h = setup({
  getSettings: async () => clone({ ...globalFixture, activeConnectionProfileId: '' }),
  listConnectionProfiles: async () => ({
    profiles: [{ id: 'legacy-current', name: '当前连接', addressLabel: 'a.example.test', apiProtocol: 'openai_chat', model: 'model-a', active: true, apiKeySet: true }],
    error: null,
  }),
  getConnectionProfile: async () => ({ success: true, profile: { id: 'legacy-current', name: '', connection: clone(profileA.connection) } }),
})
await h.store.loadSettings()
await h.store.openSettings()
h.store.closeSettings()
await h.store.openSettings()
assert.equal(h.store.connectionProfileName, '', 'legacy display name must not become an editable persisted name')
assert.equal(await h.store.saveSettingsSilent(), true)
const legacySilentPayload = h.calls.find(call => call[0] === 'full-save')[1]
assert.equal(legacySilentPayload.connectionProfileId, 'legacy-current')
assert.equal(legacySilentPayload.connectionProfileName, '', 'legacy save must let backend generate the formal name')

const profileComponentURL = new URL('./src/components/ConnectionProfileManager.vue', import.meta.url)
assert.equal(fs.existsSync(profileComponentURL), true, 'connection profile manager component must exist')
const profileSource = fs.readFileSync(profileComponentURL, 'utf8')
const modalSource = fs.readFileSync(new URL('./src/components/SettingsModal.vue', import.meta.url), 'utf8')
const modelSource = fs.readFileSync(new URL('./src/components/ModelSelect.vue', import.meta.url), 'utf8')
assert.ok(modalSource.includes('<ConnectionProfileManager'))
assert.ok(profileSource.includes('已配置连接'))
assert.ok(profileSource.includes('新建连接'))
assert.ok(profileSource.includes('记录名称'))
assert.ok(profileSource.includes('确认删除'))
assert.equal(profileSource.includes('confirm('), false)
assert.equal(modelSource.includes('savedModels'), false)
assert.equal(modelSource.includes('已保存模型'), false)

console.log('Connection profiles regression tests passed')
