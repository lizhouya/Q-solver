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
  apiKey: '', baseURL: 'http://127.0.0.1:11434/v1', connectionMode: 'direct',
  apiProtocol: 'openai_chat', endpointPath: '/chat/completions', extraHeaders: {},
  responseFormat: 'text', requestTemplate: '', responseJSONPointer: '', authHeader: '', authPrefix: '',
  model: 'local-model', outputMode: 'dual', resumeContent: 'keep-resume',
  theme: 'light',
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
    listConnectionProfiles: async () => ({
      profiles: [{ id: 'profile-a', name: '测试连接', addressLabel: '127.0.0.1:11434', apiProtocol: fixture.apiProtocol, model: fixture.model, active: true, apiKeySet: Boolean(fixture.apiKey), templateSet: Boolean(fixture.requestTemplate) }],
      error: null,
    }),
    getConnectionProfile: async () => ({ success: true, profile: { id: 'profile-a', name: '测试连接', connection: { ...fixture } } }),
    deleteConnectionProfile: async id => ({ success: true, deletedId: id }),
    getDomainCategories: async () => [],
    syncSettings: async value => { calls.push(['full-save', JSON.parse(value)]); return '' },
    listModelsForDraft: async draft => { calls.push(['models', structuredClone(draft)]); return { models: ['local-model'], error: null } },
    probeModelConnection: async request => { calls.push(['probe', structuredClone(request)]); return { kind: request.kind, outcome: 'passed', protocol: request.connection.apiProtocol, message: '通过', requestSent: true } },
    saveConnectionSettings: async profileDraft => {
      calls.push(['connection-save', structuredClone(profileDraft)])
      const draft = profileDraft.connection
      return {
        success: true,
        profile: {
          id: profileDraft.id || 'generated-profile', name: profileDraft.name || '测试连接',
          addressLabel: '127.0.0.1:11434', apiProtocol: draft.apiProtocol, model: draft.model,
          active: true, apiKeySet: Boolean(draft.apiKey), templateSet: Boolean(draft.requestTemplate),
        },
        connection: {
          connectionMode: 'direct', baseURL: draft.baseURL, model: draft.model,
          apiProtocol: draft.apiProtocol, endpointPath: draft.endpointPath,
          responseFormat: draft.responseFormat, apiKeySet: Boolean(draft.apiKey),
          extraHeadersSet: Object.keys(draft.extraHeaders).length > 0,
          templateSet: Boolean(draft.requestTemplate),
        },
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

let h = setup()
await h.store.loadSettings()
await h.store.openSettings()
assert.equal(h.store.tempSettings.connectionMode, 'direct')
assert.equal(h.store.extraHeadersText.trim(), '{}')
assert.equal(await h.store.refreshModels(), true, 'keyless model refresh must be allowed')
assert.equal(await h.store.testTextConnection(), true, 'keyless text probe must be allowed')
assert.equal(await h.store.saveConnectionSettings(), true, 'keyless connection save must be allowed')
const saveCall = h.calls.find(call => call[0] === 'connection-save')
assert.deepEqual(Object.keys(saveCall[1]).sort(), ['connection', 'id', 'name'])
assert.deepEqual(Object.keys(saveCall[1].connection).sort(), [
  'apiKey', 'apiProtocol', 'authHeader', 'authPrefix', 'baseURL', 'connectionMode',
  'endpointPath', 'extraHeaders', 'model', 'requestTemplate', 'responseFormat', 'responseJSONPointer',
].sort())
for (const forbidden of ['outputMode', 'resumeContent', 'shortcuts', 'transparency']) {
  assert.equal(Object.hasOwn(saveCall[1].connection, forbidden), false, `connection save must omit ${forbidden}`)
}
assert.equal(h.store.settings.outputMode, fixture.outputMode)
assert.equal(h.store.settings.resumeContent, fixture.resumeContent)

h = setup()
await h.store.loadSettings()
await h.store.openSettings()
h.store.extraHeadersText = '["invalid"]'
assert.equal(await h.store.refreshModels(), false)
assert.equal(await h.store.testTextConnection(), false)
assert.equal(await h.store.saveConnectionSettings(), false)
assert.equal(h.calls.length, 0, 'invalid headers must block all bridge calls')
assert.ok(h.store.connectionValidationError)

h = setup()
await h.store.loadSettings()
await h.store.openSettings()
h.store.extraHeadersText = JSON.stringify({ Authorization: 'custom-secret', 'X-Route': 'route-a' })
h.store.applyConnectionPreset({ id: 'ccr', baseURL: 'http://127.0.0.1:3456/v1', apiProtocol: 'openai_chat', endpointPath: '/chat/completions' })
assert.equal(h.calls.length, 0, 'preset selection must not list, probe, or save')
assert.equal(h.store.tempSettings.connectionMode, 'direct')
assert.equal(h.store.tempSettings.apiKey, '')
assert.equal(h.store.tempSettings.model, '')
assert.equal(JSON.parse(h.store.extraHeadersText).Authorization, undefined, 'origin change must clear custom auth header')
assert.equal(JSON.parse(h.store.extraHeadersText)['X-Route'], 'route-a')

h = setup()
await h.store.loadSettings()
await h.store.openSettings()
await h.store.refreshModels()
await h.store.testTextConnection()
assert.equal(h.ui.availableModels.length, 1)
assert.equal(h.ui.probeStatus.text.outcome, 'passed')
h.store.tempSettings.endpointPath = '/custom/chat'
await nextTick()
assert.equal(h.ui.availableModels.length, 0, 'endpoint change must invalidate catalog')
assert.equal(h.ui.probeStatus.text, null, 'endpoint change must invalidate probe')
await h.store.refreshModels()
h.ui.probeStatus.text = { outcome: 'passed' }
h.store.tempSettings.model = 'other-model'
await nextTick()
assert.equal(h.ui.availableModels.length, 1, 'model change must retain catalog')
assert.equal(h.ui.probeStatus.text, null, 'model change must invalidate probe')

h = setup()
await h.store.loadSettings()
await h.store.openSettings()
h.store.tempSettings.apiProtocol = 'custom_json'
h.store.tempSettings.baseURL = 'http://127.0.0.1:11434/generate'
h.store.tempSettings.requestTemplate = JSON.stringify({ model: '{{model}}', messages: '{{openai_messages}}' }, null, 2)
h.store.tempSettings.responseJSONPointer = '/result/text'
h.store.tempSettings.authHeader = 'Authorization'
h.store.tempSettings.authPrefix = 'Bearer '
await nextTick()
const callsBeforeTemplateRefresh = h.calls.length
assert.equal(h.store.canRefreshModelCatalog, false)
assert.equal(await h.store.refreshModels(), false, 'custom template mode must not request a model catalog')
assert.equal(h.calls.length, callsBeforeTemplateRefresh, 'custom template refresh must stop before bridge invocation')
assert.equal(h.ui.modelCatalog.status, 'unsupported')
assert.match(h.ui.modelCatalog.message, /手动填写模型/)
assert.equal(await h.store.testTextConnection(), true, 'custom template text probe must remain available')
const customProbe = h.calls.find(call => call[0] === 'probe')[1].connection
assert.equal(customProbe.requestTemplate.includes('{{openai_messages}}'), true)
assert.equal(customProbe.responseJSONPointer, '/result/text')
assert.equal(customProbe.authHeader, 'Authorization')
assert.equal(customProbe.authPrefix, 'Bearer ')

const legacyResponses = {
  ...fixture,
  apiProtocol: 'openai_responses',
  baseURL: 'https://api.example.com/v1',
  endpointPath: '/responses',
  model: 'responses-model',
}
h = setup({
  getConnectionProfile: async () => ({ success: true, profile: { id: 'profile-a', name: '旧 Responses', connection: { ...legacyResponses } } }),
})
await h.store.loadSettings()
await h.store.openSettings()
assert.equal(h.store.settings.apiProtocol, 'openai_responses', 'saved legacy protocol must remain unchanged before save')
assert.equal(h.store.tempSettings.apiProtocol, 'custom_json')
assert.equal(h.store.tempSettings.baseURL, 'https://api.example.com/v1/responses')
assert.equal(h.store.tempSettings.requestTemplate.includes('{{responses_input}}'), true)
assert.equal(h.store.tempSettings.responseJSONPointer, '/output/0/content/0/text')
assert.match(h.store.legacyMigrationNotice, /OpenAI Responses/)
assert.equal(h.calls.length, 0, 'legacy migration must not save or call the network while opening')

const legacyAnthropic = {
  ...fixture,
  apiProtocol: 'anthropic_messages',
  baseURL: 'https://api.anthropic.com',
  endpointPath: '/v1/messages',
  extraHeaders: {},
  model: 'anthropic-model',
}
h = setup({
  getConnectionProfile: async () => ({ success: true, profile: { id: 'profile-a', name: '旧 Anthropic', connection: { ...legacyAnthropic } } }),
})
await h.store.loadSettings()
await h.store.openSettings()
assert.equal(h.store.settings.apiProtocol, 'anthropic_messages')
assert.equal(h.store.tempSettings.apiProtocol, 'custom_json')
assert.equal(h.store.tempSettings.baseURL, 'https://api.anthropic.com/v1/messages')
assert.equal(h.store.tempSettings.requestTemplate.includes('{{anthropic_system}}'), true)
assert.equal(h.store.tempSettings.requestTemplate.includes('{{anthropic_messages}}'), true)
assert.equal(h.store.tempSettings.responseJSONPointer, '/content/0/text')
assert.equal(h.store.tempSettings.authHeader, 'x-api-key')
assert.equal(h.store.tempSettings.authPrefix, '')
assert.equal(JSON.parse(h.store.extraHeadersText)['anthropic-version'], '2023-06-01')
assert.match(h.store.legacyMigrationNotice, /Anthropic Messages/)
assert.equal(h.calls.length, 0)

let resolveSave
h = setup({
  saveConnectionSettings: profileDraft => new Promise(resolve => {
    const draft = profileDraft.connection
    resolveSave = () => resolve({
      success: true,
      profile: { id: profileDraft.id, name: profileDraft.name, active: true, apiProtocol: draft.apiProtocol, model: draft.model },
      connection: { ...draft, apiKeySet: false, extraHeadersSet: false },
    })
  }),
})
await h.store.loadSettings()
await h.store.openSettings()
const pendingSave = h.store.saveConnectionSettings()
h.store.tempSettings.baseURL = 'http://127.0.0.1:9999/v1'
await nextTick()
resolveSave()
assert.equal(await pendingSave, false, 'stale save result must be ignored')
assert.equal(h.store.settings.baseURL, fixture.baseURL)

h = setup()
await h.store.loadSettings()
await h.store.openSettings()
h.store.tempSettings.apiProtocol = 'custom_json'
h.store.tempSettings.requestTemplate = '{"messages":'
h.store.tempSettings.responseJSONPointer = '/answer'
await nextTick()
assert.equal(await h.store.testTextConnection(), false, 'invalid template must be rejected before bridge')
assert.equal(await h.store.saveConnectionSettings(), false)
assert.equal(h.calls.length, 0)
assert.ok(h.store.connectionValidationError)
h.store.tempSettings.requestTemplate = '{"messages":"{{openai_messages}}"}'
h.store.tempSettings.authHeader = 'x-api-key'
h.store.tempSettings.authPrefix = ''
h.store.tempSettings.domainId = 'unsaved-scene'
await nextTick()
assert.equal(await h.store.saveConnectionSettings(), true)
assert.equal(h.store.settings.requestTemplate, h.store.tempSettings.requestTemplate)
assert.equal(h.store.settings.authPrefix, '', 'empty custom prefix must survive save')
assert.notEqual(h.store.settings.domainId, 'unsaved-scene')
await h.store.saveSettingsSilent()
const silentPayload = h.calls.find(call => call[0] === 'full-save')[1]
assert.equal(silentPayload.requestTemplate, h.store.settings.requestTemplate)
assert.notEqual(silentPayload.domainId, 'unsaved-scene')
h.store.closeSettings()
await h.store.openSettings()
assert.equal(h.store.tempSettings.authPrefix, '')
assert.equal(h.store.tempSettings.responseJSONPointer, '/answer')

console.log('Unified API settings regression tests passed')
