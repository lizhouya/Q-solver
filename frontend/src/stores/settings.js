import { defineStore } from 'pinia'
import { reactive, ref, computed, watch } from 'vue'
import { api } from '../services/api'
import { useUIStore } from './ui'
import { currentTheme, setTheme } from '../services/theme'

export const useSettingsStore = defineStore('settings', () => {
  const ui = useUIStore()

  const settings = reactive({
    apiKey: '',
    baseURL: 'https://api.openai.com/v1',
    connectionMode: 'direct',
    apiProtocol: 'openai_chat',
    endpointPath: '/chat/completions',
    extraHeaders: {},
    responseFormat: 'text',
    requestTemplate: '',
    responseJSONPointer: '',
    authHeader: '',
    authPrefix: '',
    model: '',
    assistantModel: '',
    prompt: '',
    domainId: 'general-assistant',
    transparency: 0,
    keepContext: false,
    outputMode: 'desktop',
    screenshotMode: 'window',
    resumePath: '',
    resumeContent: '',
    compressionQuality: 80,
    sharpening: 0,
    grayscale: true,
    noCompression: false,
  })

  const settingsLoaded = ref(false)
  const settingsLoading = ref(false)
  const settingsLoadError = ref('')
  let backendSnapshot = {}
  let loadPromise = null
  let sessionRevision = 0
  let connectionRevision = 0
  let catalogRevision = 0
  let probeRevision = 0
  let connectionSaveRequestVersion = 0
  let profileLoadRequestVersion = 0
  let modelRequestVersion = 0
  let catalogPromise = null
  let catalogPromiseSession = -1
  let catalogPromiseRevision = -1
  let catalogLoadedSession = -1
  let catalogLoadedRevision = -1
  let textProbeRequestVersion = 0
  let visionProbeRequestVersion = 0

  const tempSettings = reactive({ ...settings })
  const extraHeadersText = ref('{}')
  const connectionValidationError = ref('')
  const connectionSaveLoading = ref(false)
  const legacyMigrationNotice = ref('')
  const connectionProfiles = ref([])
  const activeConnectionProfileID = ref('')
  const selectedConnectionProfileID = ref('')
  const connectionProfileName = ref('')
  const connectionProfileLoading = ref(false)
  const connectionProfileLoaded = ref(false)
  const pendingDeleteProfileID = ref('')
  const shortcuts = reactive({})
  const tempShortcuts = reactive({})
  const domainCategories = ref([])
  const resumeRawContent = ref('')
  const isResumeParsing = ref(false)

  const isMacOS = ref(
    typeof navigator !== 'undefined' &&
    (navigator.platform.toLowerCase().includes('mac') ||
      navigator.userAgent.toLowerCase().includes('mac'))
  )

  const recordingAction = ref(null)
  const recordingText = ref('')
  const allShortcutActions = [
    { action: 'screenshot', label: '截图', default: 'F8', macDefault: 'Cmd+1' },
    { action: 'send', label: '发送解题', default: 'Ctrl+J', macDefault: 'Cmd+J' },
    { action: 'send_fast', label: '快速解题', default: 'Ctrl+Shift+J', macDefault: '⌘⇧J' },
    { action: 'cancel_solve', label: '中断解题', default: 'Ctrl+Shift+X', macDefault: '⌘⇧X' },
    { action: 'delete', label: '删除截图', default: 'Ctrl+D', macDefault: 'Cmd+D' },
    { action: 'toggle', label: '隐身模式', default: 'F9', macDefault: 'Cmd+2' },
    { action: 'toggle_window', label: '隐藏/显示窗口', default: 'Ctrl+Shift+H', macDefault: 'Cmd+4' },
    { action: 'clickthrough', label: '鼠标穿透', default: 'F10', macDefault: 'Cmd+3' },
    { action: 'move_up', label: '向上移动', default: 'Alt+Up', macDefault: 'Cmd+Option+Up' },
    { action: 'move_down', label: '向下移动', default: 'Alt+Down', macDefault: 'Cmd+Option+Down' },
    { action: 'move_left', label: '向左移动', default: 'Alt+Left', macDefault: 'Cmd+Option+Left' },
    { action: 'move_right', label: '向右移动', default: 'Alt+Right', macDefault: 'Cmd+Option+Right' },
    { action: 'scroll_up', label: '向上滚动', default: 'Alt+PgUp', macDefault: 'Cmd+Option+Shift+Up' },
    { action: 'scroll_down', label: '向下滚动', default: 'Alt+PgDn', macDefault: 'Cmd+Option+Shift+Down' },
    { action: 'clear_mobile_display', label: '清空同步端显示', default: 'Shift+C', windowsOnly: true },
    { action: 'start_next_exam', label: '开始下一套', default: 'Ctrl+Shift+N', windowsOnly: true },
  ]
  const shortcutActions = computed(() =>
    allShortcutActions.filter(item => !item.windowsOnly || !isMacOS.value)
  )

  function shortcutDisplayName(key) {
    if (recordingAction.value === key.action) return recordingText.value
    if (Object.prototype.hasOwnProperty.call(tempShortcuts, key.action)) {
      return tempShortcuts[key.action]?.keyName || '未绑定（点击录制）'
    }
    return isMacOS.value ? key.macDefault : key.default
  }

  const maskedKey = computed(() => {
    if (!settings.apiKey) return ''
    if (settings.apiKey.length < 8) return settings.apiKey
    return settings.apiKey.substring(0, 3) + '****' + settings.apiKey.substring(settings.apiKey.length - 4)
  })

  const solveShortcut = computed(() => shortcuts.screenshot?.keyName || shortcuts.solve?.keyName || 'F8')
  const sendShortcut = computed(() => shortcuts.send?.keyName || 'Ctrl+J')
  const fastSendShortcut = computed(() => shortcuts.send_fast?.keyName || (isMacOS.value ? '⌘⇧J' : 'Ctrl+Shift+J'))
  const cancelSolveShortcut = computed(() => shortcuts.cancel_solve?.keyName || (isMacOS.value ? '⌘⇧X' : 'Ctrl+Shift+X'))
  const deleteShortcut = computed(() => shortcuts.delete?.keyName || 'Ctrl+D')
  const toggleShortcut = computed(() => shortcuts.toggle?.keyName || 'F9')

  const statusText = ref('就绪')
  const statusIcon = ref('●')

  function resetStatus() {
    if (!settings.baseURL) {
      statusText.value = '未配置'
      statusIcon.value = '!'
    } else {
      statusText.value = '已配置 / 未验证'
      statusIcon.value = '✓'
    }
  }

  watch(() => settings.baseURL, () => resetStatus(), { immediate: true })

  watch(() => tempSettings.transparency, (newVal) => {
    applyTransparency(1.0 - newVal)
  })

  window.addEventListener('theme-changed', () => {
    requestAnimationFrame(() => {
      applyTransparency(1.0 - (tempSettings.transparency ?? 0))
    })
  })

  watch(() => resumeRawContent.value, (newVal) => {
    tempSettings.resumeContent = newVal || ''
  })

  watch(currentTheme, () => {
    if (settingsLoaded.value) saveSettingsSilent()
  })

  function applyTransparency(opacity) {
    const root = document.documentElement
    if (!root) return
    const style = getComputedStyle(root)
    const base = style.getPropertyValue('--surface-base').trim()
    let r, g, b
    if (base.startsWith('rgba')) {
      const parts = base.match(/rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*[\d.]+\s*\)/)
      if (parts) {
        r = parts[1]
        g = parts[2]
        b = parts[3]
      }
    }
    if (!r) {
      const theme = root.getAttribute('data-theme')
      if (theme === 'light') {
        r = '252'
        g = '252'
        b = '255'
      } else {
        r = '10'
        g = '10'
        b = '14'
      }
    }
    root.style.setProperty('--app-bg-r', r)
    root.style.setProperty('--app-bg-g', g)
    root.style.setProperty('--app-bg-b', b)
    root.style.setProperty('--app-bg-a', String(opacity))
    root.style.setProperty('--app-panel-a', String(Math.max(0, opacity * 0.5)))
    root.style.setProperty('--text-shadow-a', String(Math.max(0, (1 - opacity) * 0.8)))

    const isLight = root.getAttribute('data-theme') === 'light'
    root.style.setProperty('--surface-elevated-a', String(opacity * (isLight ? 0.96 : 0.92)))
    root.style.setProperty('--surface-card-a', String(opacity * (isLight ? 0.78 : 0.80)))
    root.style.setProperty('--surface-card-hover-a', String(opacity * (isLight ? 0.86 : 0.88)))
  }

  function loadSettings() {
    if (loadPromise) return loadPromise
    settingsLoading.value = true
    settingsLoadError.value = ''
    settingsLoaded.value = false
    loadPromise = (async () => {
      const configTask = (async () => {
        try {
          const config = await api.getSettings()
          if (!config || typeof config !== 'object') throw new Error('invalid configuration')
          backendSnapshot = { ...config }
          applyConfig(config)
          Object.assign(shortcuts, config.shortcuts || {})
          Object.assign(tempShortcuts, shortcuts)
          if (config.theme) setTheme(config.theme)
          const profileLoaded = await loadConnectionProfilesForSettings(config.activeConnectionProfileId || '')
          if (!profileLoaded) throw new Error('connection profile unavailable')
          settingsLoaded.value = true
        } catch (_) {
          connectionProfileLoaded.value = false
          settingsLoadError.value = '配置读取失败，已阻止保存以保护原配置。请点击重新读取。'
          ui.showToast(settingsLoadError.value, 'error')
        }
      })()
      const categoriesTask = (async () => {
        try {
          const categories = await api.getDomainCategories()
          if (categories) domainCategories.value = categories
        } catch (_) {
          ui.showToast('场景列表加载失败，不影响 API 配置读取与保存', 'warning')
        }
      })()
      await Promise.all([configTask, categoriesTask])
      return settingsLoaded.value
    })().finally(() => {
      settingsLoading.value = false
      loadPromise = null
    })
    return loadPromise
  }

  function applyConfig(config) {
    settings.assistantModel = config.assistantModel || ''
    settings.prompt = config.prompt || ''
    settings.domainId = config.domainId || 'general-assistant'
    settings.compressionQuality = config.compressionQuality || 80
    settings.sharpening = config.sharpening || 0
    settings.grayscale = config.grayscale !== undefined ? config.grayscale : true
    settings.noCompression = config.noCompression || false
    settings.keepContext = config.keepContext || false
    settings.outputMode = config.outputMode || 'desktop'
    settings.resumePath = config.resumePath || ''
    settings.resumeContent = config.resumeContent || ''
    settings.screenshotMode = config.screenshotMode || 'window'

    const opacity = config.opacity !== undefined ? config.opacity : 1.0
    settings.transparency = 1.0 - opacity
    applyTransparency(opacity)
  }

  function defaultEndpointPath(apiProtocol) {
    if (apiProtocol === 'custom_json') return ''
    if (apiProtocol === 'openai_responses') return '/responses'
    if (apiProtocol === 'anthropic_messages') return '/v1/messages'
    return '/chat/completions'
  }

  function defaultCustomTemplate() {
    return JSON.stringify({ model: '{{model}}', messages: '{{openai_messages}}' }, null, 2)
  }

  function joinRequestURL(baseURL, endpointPath) {
    try {
      const url = new URL(baseURL)
      const basePath = url.pathname.replace(/\/+$/, '')
      const endpoint = String(endpointPath || '').replace(/^\/+/, '')
      url.pathname = [basePath, endpoint].filter(Boolean).join('/') || '/'
      url.search = ''
      url.hash = ''
      return url.toString().replace(/\/$/, '')
    } catch (_) {
      return `${String(baseURL || '').replace(/\/+$/, '')}/${String(endpointPath || '').replace(/^\/+/, '')}`.replace(/\/$/, '')
    }
  }

  function migrateLegacyConnectionDraft(config) {
    const draft = JSON.parse(JSON.stringify(config || {}))
    draft.connectionMode = 'direct'
    draft.requestTemplate = draft.requestTemplate || ''
    draft.responseJSONPointer = draft.responseJSONPointer || ''
    draft.authHeader = draft.authHeader || ''
    draft.authPrefix = draft.authPrefix || ''
    draft.extraHeaders = { ...(draft.extraHeaders || {}) }
    legacyMigrationNotice.value = ''

    if (draft.apiProtocol === 'openai_responses') {
      draft.apiProtocol = 'custom_json'
      draft.baseURL = joinRequestURL(draft.baseURL, draft.endpointPath || '/responses')
      draft.endpointPath = ''
      draft.responseFormat = 'text'
      draft.requestTemplate = JSON.stringify({ model: '{{model}}', input: '{{responses_input}}', store: false }, null, 2)
      draft.responseJSONPointer = '/output/0/content/0/text'
      draft.authHeader = 'Authorization'
      draft.authPrefix = 'Bearer '
      legacyMigrationNotice.value = '已将旧 OpenAI Responses 配置转换为自定义模板草稿；保存后改为非流式 JSON。'
    } else if (draft.apiProtocol === 'anthropic_messages') {
      draft.apiProtocol = 'custom_json'
      draft.baseURL = joinRequestURL(draft.baseURL, draft.endpointPath || '/v1/messages')
      draft.endpointPath = ''
      draft.responseFormat = 'text'
      draft.requestTemplate = JSON.stringify({
        model: '{{model}}', max_tokens: 8192,
        system: '{{anthropic_system}}', messages: '{{anthropic_messages}}',
      }, null, 2)
      draft.responseJSONPointer = '/content/0/text'
      draft.authHeader = 'x-api-key'
      draft.authPrefix = ''
      if (!Object.keys(draft.extraHeaders).some(key => key.toLowerCase() === 'anthropic-version')) {
        draft.extraHeaders['anthropic-version'] = '2023-06-01'
      }
      legacyMigrationNotice.value = '已将旧 Anthropic Messages 配置转换为自定义模板草稿；保存后改为非流式 JSON。'
    }
    return draft
  }

  function applyLegacyConnectionDraft(config) {
    const draft = migrateLegacyConnectionDraft(config)
    Object.assign(tempSettings, draft)
    extraHeadersText.value = formatExtraHeaders(draft.extraHeaders)
  }

  function applyConnectionDraft(target, draft) {
    const apiProtocol = draft?.apiProtocol || 'openai_chat'
    Object.assign(target, {
      connectionMode: 'direct',
      apiKey: draft?.apiKey || '',
      baseURL: draft?.baseURL || '',
      apiProtocol,
      endpointPath: draft?.endpointPath ?? defaultEndpointPath(apiProtocol),
      extraHeaders: { ...(draft?.extraHeaders || {}) },
      responseFormat: draft?.responseFormat || 'text',
      requestTemplate: draft?.requestTemplate || '',
      responseJSONPointer: draft?.responseJSONPointer || '',
      authHeader: draft?.authHeader || '',
      authPrefix: draft?.authPrefix ?? '',
      model: draft?.model || '',
    })
  }

  function activeProfileSummary() {
    return connectionProfiles.value.find(profile => profile.id === activeConnectionProfileID.value) || null
  }

  async function loadConnectionProfileByID(id, { asActive = false } = {}) {
    const version = ++profileLoadRequestVersion
    const requestSession = sessionRevision
    const wasLoaded = connectionProfileLoaded.value
    connectionProfileLoading.value = true
    clearConnectionResults()
    try {
      const result = await api.getConnectionProfile(id)
      const isCurrent = version === profileLoadRequestVersion && requestSession === sessionRevision
      if (!isCurrent) return false
      if (!result?.success || result?.error || !result?.profile?.connection) {
        ui.showToast(result?.error?.message || '连接记录读取失败，请重试', 'error')
        connectionProfileLoaded.value = wasLoaded
        return false
      }
      const profile = result.profile
      if (asActive) {
        applyConnectionDraft(settings, profile.connection)
        activeConnectionProfileID.value = profile.id
        backendSnapshot = { ...backendSnapshot, activeConnectionProfileId: profile.id }
      }
      const draftBase = asActive ? settings : tempSettings
      applyConnectionDraft(draftBase, profile.connection)
      applyLegacyConnectionDraft(draftBase)
      selectedConnectionProfileID.value = profile.id
      connectionProfileName.value = profile.name || ''
      pendingDeleteProfileID.value = ''
      connectionProfileLoaded.value = true
      connectionValidationError.value = ''
      return true
    } catch (_) {
      if (version === profileLoadRequestVersion && requestSession === sessionRevision) {
        connectionProfileLoaded.value = wasLoaded
        ui.showToast('连接记录读取失败，请检查应用状态后重试', 'error')
      }
      return false
    } finally {
      if (version === profileLoadRequestVersion) connectionProfileLoading.value = false
    }
  }

  async function loadConnectionProfilesForSettings(activeIDHint = '') {
    connectionProfileLoading.value = true
    connectionProfileLoaded.value = false
    try {
      const result = await api.listConnectionProfiles()
      if (result?.error || !Array.isArray(result?.profiles) || result.profiles.length === 0) {
        ui.showToast(result?.error?.message || '连接记录列表读取失败', 'error')
        return false
      }
      connectionProfiles.value = JSON.parse(JSON.stringify(result.profiles))
      const active = connectionProfiles.value.find(profile => profile.id === activeIDHint) ||
        connectionProfiles.value.find(profile => profile.active) || connectionProfiles.value[0]
      return await loadConnectionProfileByID(active.id, { asActive: true })
    } catch (_) {
      ui.showToast('连接记录列表读取失败，请检查应用状态后重试', 'error')
      return false
    } finally {
      if (!connectionProfileLoaded.value) connectionProfileLoading.value = false
    }
  }

  async function selectConnectionProfile(id) {
    if (!id) return false
    sessionRevision++
    connectionRevision++
    catalogRevision++
    probeRevision++
    return loadConnectionProfileByID(id)
  }

  function beginNewConnectionProfile() {
    const formalCount = connectionProfiles.value.filter(profile => profile.id !== 'legacy-current').length
    if (formalCount >= 20) {
      ui.showToast('最多只能保存 20 条连接记录', 'warning')
      return false
    }
    sessionRevision++
    connectionRevision++
    catalogRevision++
    probeRevision++
    profileLoadRequestVersion++
    selectedConnectionProfileID.value = ''
    connectionProfileName.value = ''
    pendingDeleteProfileID.value = ''
    applyConnectionDraft(tempSettings, {
      connectionMode: 'direct', apiKey: '', baseURL: '', apiProtocol: 'openai_chat',
      endpointPath: '/chat/completions', extraHeaders: {}, responseFormat: 'text',
      requestTemplate: '', responseJSONPointer: '', authHeader: '', authPrefix: '', model: '',
    })
    extraHeadersText.value = '{}'
    legacyMigrationNotice.value = ''
    connectionValidationError.value = ''
    connectionProfileLoading.value = false
    connectionProfileLoaded.value = true
    clearConnectionResults()
    return true
  }

  const automaticConnectionProfileName = computed(() => {
    let hostname = '未命名连接'
    try { hostname = new URL(tempSettings.baseURL).hostname || hostname } catch (_) {}
    const protocolName = tempSettings.apiProtocol === 'custom_json' ? '自定义模板' : 'OpenAI Chat'
    return `${hostname} · ${protocolName}`
  })

  function requestDeleteConnectionProfile(id) {
    const profile = connectionProfiles.value.find(item => item.id === id)
    if (!profile || profile.active || id === 'legacy-current') {
      pendingDeleteProfileID.value = ''
      return false
    }
    pendingDeleteProfileID.value = id
    return true
  }

  function cancelDeleteConnectionProfile() {
    pendingDeleteProfileID.value = ''
  }

  async function confirmDeleteConnectionProfile() {
    const id = pendingDeleteProfileID.value
    if (!id) return false
    const profile = connectionProfiles.value.find(item => item.id === id)
    if (!profile || profile.active || id === 'legacy-current') {
      pendingDeleteProfileID.value = ''
      return false
    }
    try {
      const result = await api.deleteConnectionProfile(id)
      if (!result?.success || result?.error) {
        ui.showToast(result?.error?.message || '连接记录删除失败，原配置未修改', 'error')
        return false
      }
      connectionProfiles.value = connectionProfiles.value.filter(item => item.id !== id)
      pendingDeleteProfileID.value = ''
      if (selectedConnectionProfileID.value === id) {
        const active = activeProfileSummary()
        if (active) await selectConnectionProfile(active.id)
      }
      ui.showToast('连接记录已删除', 'success')
      return true
    } catch (_) {
      ui.showToast('连接记录删除失败，请检查应用状态后重试', 'error')
      return false
    }
  }

  const isCustomTemplateMode = computed(() => tempSettings.apiProtocol === 'custom_json')
  const canRefreshModelCatalog = computed(() => !isCustomTemplateMode.value)

  function formatExtraHeaders(headers) {
    const sorted = Object.fromEntries(Object.entries(headers || {}).sort(([a], [b]) => a.localeCompare(b)))
    return JSON.stringify(sorted, null, 2)
  }

  function parseExtraHeaders() {
    try {
      const parsed = JSON.parse(extraHeadersText.value || '{}')
      if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
        throw new Error('额外 Headers 必须是 JSON 对象')
      }
      for (const [key, value] of Object.entries(parsed)) {
        if (typeof value !== 'string') throw new Error(`Header ${key} 的值必须是字符串`)
      }
      connectionValidationError.value = ''
      return parsed
    } catch (error) {
      connectionValidationError.value = '额外 Headers 必须是字符串到字符串的合法 JSON 对象'
      throw error
    }
  }

  function effectiveConnectionMode() {
    return 'direct'
  }

  function effectiveCatalogProtocol() {
    return tempSettings.apiProtocol || 'openai_chat'
  }

  function validateTemplateDraft() {
    const fail = () => {
      connectionValidationError.value = '请求模板或响应路径无效，请检查高级设置；占位符必须独立作为 JSON 值。'
      throw new Error('invalid template configuration')
    }
    const allowed = new Set(['{{model}}', '{{system_prompt}}', '{{user_text}}', '{{openai_messages}}', '{{responses_input}}', '{{anthropic_messages}}', '{{anthropic_system}}', '{{images_data_urls}}', '{{images_base64}}'])
    let root
    try { root = JSON.parse(tempSettings.requestTemplate) } catch (_) { fail() }
    if (!root || Array.isArray(root) || typeof root !== 'object' || tempSettings.requestTemplate.length > 65536) fail()
    let content = false
    function visit(value, depth = 0) {
      if (depth > 64) fail()
      if (typeof value === 'string') {
        if ((value.includes('{{') || value.includes('}}')) && !allowed.has(value)) fail()
        if (allowed.has(value) && value !== '{{model}}') content = true
      } else if (Array.isArray(value)) {
        value.forEach(child => visit(child, depth + 1))
      } else if (value && typeof value === 'object') {
        for (const [key, child] of Object.entries(value)) {
          if (key.includes('{{') || key.includes('}}')) fail()
          visit(child, depth + 1)
        }
      }
    }
    visit(root)
    const pointer = tempSettings.responseJSONPointer || ''
    if (!content || !pointer.startsWith('/') || /~(?![01])/.test(pointer)) fail()
    connectionValidationError.value = ''
  }

  function connectionDraft() {
    const apiProtocol = effectiveCatalogProtocol()
    if (apiProtocol === 'custom_json') validateTemplateDraft()
    return {
      connectionMode: 'direct',
      apiKey: tempSettings.apiKey || '',
      baseURL: tempSettings.baseURL || '',
      model: tempSettings.model || '',
      apiProtocol,
      endpointPath: apiProtocol === 'custom_json' ? '' : (tempSettings.endpointPath || defaultEndpointPath(apiProtocol)),
      extraHeaders: parseExtraHeaders(),
      responseFormat: apiProtocol === 'custom_json' ? 'text' : (tempSettings.responseFormat || 'text'),
      requestTemplate: apiProtocol === 'custom_json' ? (tempSettings.requestTemplate || '') : '',
      responseJSONPointer: apiProtocol === 'custom_json' ? (tempSettings.responseJSONPointer || '') : '',
      authHeader: apiProtocol === 'custom_json' ? (tempSettings.authHeader || 'Authorization') : '',
      authPrefix: apiProtocol === 'custom_json' ? (tempSettings.authPrefix ?? 'Bearer ') : '',
    }
  }

  function modelCatalogState() {
    if (!ui.modelCatalog) {
      ui.modelCatalog = reactive({ status: 'idle', message: '', errorCode: '', lastSyncedAt: 0 })
    }
    return ui.modelCatalog
  }

  function setModelCatalog(status, message = '', errorCode = '', lastSyncedAt = modelCatalogState().lastSyncedAt || 0) {
    Object.assign(modelCatalogState(), { status, message, errorCode, lastSyncedAt })
  }

  function clearCatalogResults() {
    modelRequestVersion++
    catalogPromise = null
    catalogLoadedSession = -1
    catalogLoadedRevision = -1
    ui.availableModels = []
    ui.isLoadingModels = false
    setModelCatalog('idle', '', '', 0)
  }

  function clearProbeResults() {
    textProbeRequestVersion++
    visionProbeRequestVersion++
    ui.textProbeLoading = false
    ui.visionProbeLoading = false
    ui.probeStatus.text = null
    ui.probeStatus.vision = null
  }

  function clearConnectionResults({ clearModels = true } = {}) {
    if (clearModels) clearCatalogResults()
    clearProbeResults()
  }

  watch(
    () => [
      tempSettings.apiKey,
      tempSettings.baseURL,
      effectiveCatalogProtocol(),
      tempSettings.endpointPath,
      extraHeadersText.value,
      tempSettings.responseFormat,
      tempSettings.requestTemplate,
      tempSettings.responseJSONPointer,
      tempSettings.authHeader,
      tempSettings.authPrefix,
    ],
    () => {
      connectionRevision++
      catalogRevision++
      probeRevision++
      clearCatalogResults()
      clearProbeResults()
    },
    { flush: 'sync' }
  )

  watch(() => tempSettings.model, () => {
    connectionRevision++
    probeRevision++
    clearProbeResults()
  }, { flush: 'sync' })

  async function fetchModels() {
    if (!canRefreshModelCatalog.value) {
      setModelCatalog('unsupported', '自定义请求模板不提供模型目录，请手动填写模型', 'catalog_unsupported', 0)
      return false
    }
    let draft
    try {
      draft = connectionDraft()
    } catch (_) {
      setModelCatalog('needs_config', connectionValidationError.value)
      ui.showToast(connectionValidationError.value, 'warning')
      return false
    }
    if (!draft.baseURL) {
      setModelCatalog('needs_config', '请先填写接口地址')
      return false
    }
    const version = ++modelRequestVersion
    const requestSession = sessionRevision
    const requestCatalog = catalogRevision
    const isCurrent = () => (
      version === modelRequestVersion &&
      requestSession === sessionRevision &&
      requestCatalog === catalogRevision
    )
    ui.isLoadingModels = true
    setModelCatalog('syncing', '正在同步模型目录')
    try {
      const result = await api.listModelsForDraft(draft)
      if (!isCurrent()) return false
      if (result?.error) {
        ui.availableModels = []
        setModelCatalog('failed', result.error.message || '获取模型列表失败', result.error.code || '')
        ui.showToast(result.error.message || '获取模型列表失败', 'error')
        return false
      }
      ui.availableModels = Array.isArray(result?.models) ? result.models : []
      const syncedAt = Date.now()
      if (ui.availableModels.length === 0) {
        setModelCatalog('empty', '接口未返回模型；仍可手动填写模型名称', '', syncedAt)
        ui.showToast('接口未返回模型；仍可手动填写模型名称', 'warning')
        return false
      }
      catalogLoadedSession = requestSession
      catalogLoadedRevision = requestCatalog
      setModelCatalog('synced', `已同步 ${ui.availableModels.length} 个模型`, '', syncedAt)
      return true
    } catch (error) {
      if (!isCurrent()) return false
      ui.availableModels = []
      if (error?.code === 'frontend_wait_timeout') {
        setModelCatalog('ui_timeout', error.message, error.code)
        ui.showToast(error.message, 'warning')
      } else {
        setModelCatalog('failed', '获取模型列表失败；仍可手动填写模型名称')
        ui.showToast('获取模型列表失败；仍可手动填写模型名称', 'error')
      }
      return false
    } finally {
      if (isCurrent()) ui.isLoadingModels = false
    }
  }

  async function refreshModels() {
    if (!canRefreshModelCatalog.value) return fetchModels()
    if (!tempSettings.baseURL) {
      setModelCatalog('needs_config', '请先填写接口地址')
      ui.showToast('请先填写接口地址', 'warning')
      return false
    }
    catalogPromise = null
    const success = await fetchModels()
    if (success) ui.showToast(`已加载 ${ui.availableModels.length} 个模型建议`, 'success')
    return success
  }

  function ensureCCRModelCatalog() {
    return refreshModels()
  }

  async function runProbe(kind) {
    let draft
    try {
      draft = connectionDraft()
    } catch (_) {
      ui.showToast(connectionValidationError.value, 'warning')
      return false
    }
    if (!draft.baseURL || !draft.model) {
      ui.showToast('请先填写接口地址和模型', 'warning')
      return false
    }
    const isText = kind === 'text'
    const version = isText ? ++textProbeRequestVersion : ++visionProbeRequestVersion
    const requestSession = sessionRevision
    const requestProbe = probeRevision
    const isCurrent = () => (
      requestSession === sessionRevision &&
      requestProbe === probeRevision &&
      version === (isText ? textProbeRequestVersion : visionProbeRequestVersion)
    )
    if (isText) ui.textProbeLoading = true
    else ui.visionProbeLoading = true
    ui.probeStatus[kind] = null
    try {
      const result = await api.probeModelConnection({ connection: draft, kind })
      if (!isCurrent()) return false
      ui.probeStatus[kind] = result || {
        kind, outcome: 'failed', message: '模型测试未返回有效结果',
      }
      if (result?.outcome === 'passed') {
        ui.showToast(kind === 'text' ? '文字实测通过' : '图片实测通过', 'success')
        return true
      }
      ui.showToast(result?.message || '模型实测失败', result?.outcome === 'inconclusive' ? 'warning' : 'error')
      return false
    } catch (error) {
      if (!isCurrent()) return false
      const timedOut = error?.code === 'frontend_wait_timeout'
      ui.probeStatus[kind] = {
        kind,
        outcome: 'failed',
        message: timedOut ? error.message : '模型测试调用失败，请检查应用状态',
        ...(timedOut ? { error: { code: error.code, operation: error.operation } } : {}),
      }
      ui.showToast(ui.probeStatus[kind].message, timedOut ? 'warning' : 'error')
      return false
    } finally {
      if (isCurrent()) {
        if (isText) ui.textProbeLoading = false
        else ui.visionProbeLoading = false
      }
    }
  }

  const testTextConnection = () => runProbe('text')
  const testVisionConnection = () => runProbe('vision')
  const testConnection = testTextConnection

  function connectionOrigin(value) {
    try { return new URL(value).origin.toLowerCase() } catch (_) { return '' }
  }

  function applyConnectionPreset(preset) {
    const nextURL = preset?.baseURL || ''
    const currentOrigin = connectionOrigin(tempSettings.baseURL)
    const nextOrigin = connectionOrigin(nextURL)
    let headers
    try {
      headers = parseExtraHeaders()
    } catch (_) {
      headers = {}
    }
    if (!currentOrigin || !nextOrigin || currentOrigin !== nextOrigin) {
      tempSettings.apiKey = ''
      tempSettings.model = ''
      for (const key of Object.keys(headers)) {
        const lower = key.toLowerCase()
        if (lower === 'authorization' || lower === 'x-api-key' || lower === tempSettings.authHeader.toLowerCase()) delete headers[key]
      }
    }
    tempSettings.connectionMode = 'direct'
    tempSettings.baseURL = nextURL
    tempSettings.apiProtocol = preset?.apiProtocol || 'openai_chat'
    tempSettings.endpointPath = preset?.endpointPath || defaultEndpointPath(tempSettings.apiProtocol)
    tempSettings.responseFormat = preset?.responseFormat || 'text'
    tempSettings.extraHeaders = { ...headers }
    extraHeadersText.value = formatExtraHeaders(headers)
    applyLegacyConnectionDraft(tempSettings)
    connectionValidationError.value = ''
    clearConnectionResults()
  }

  const applyProviderPreset = applyConnectionPreset

  function buildConfigToSave(sourceSettings, sourceShortcuts, profileID, profileName) {
    return {
      ...backendSnapshot,
      connectionProfileId: profileID,
      connectionProfileName: profileName,
      connectionMode: 'direct',
      apiKey: sourceSettings.apiKey,
      baseURL: sourceSettings.baseURL,
      apiProtocol: sourceSettings.apiProtocol || 'openai_chat',
      endpointPath: sourceSettings.endpointPath || defaultEndpointPath(sourceSettings.apiProtocol),
      extraHeaders: { ...(sourceSettings.extraHeaders || {}) },
      responseFormat: sourceSettings.responseFormat || 'text',
      requestTemplate: sourceSettings.requestTemplate || '',
      responseJSONPointer: sourceSettings.responseJSONPointer || '',
      authHeader: sourceSettings.authHeader || '',
      authPrefix: sourceSettings.authPrefix || '',
      model: sourceSettings.model,
      assistantModel: sourceSettings.assistantModel,
      prompt: '',
      domainId: sourceSettings.domainId,
      opacity: 1.0 - sourceSettings.transparency,
      keepContext: sourceSettings.keepContext,
      outputMode: sourceSettings.outputMode,
      screenshotMode: sourceSettings.screenshotMode,
      compressionQuality: sourceSettings.compressionQuality,
      sharpening: sourceSettings.sharpening,
      grayscale: sourceSettings.grayscale,
      noCompression: sourceSettings.noCompression,
      resumePath: sourceSettings.resumePath,
      resumeContent: sourceSettings.resumeContent,
      shortcuts: sourceShortcuts,
      theme: currentTheme.value,
    }
  }

  async function saveConnectionSettings() {
    if (!settingsLoaded.value || settingsLoading.value || connectionSaveLoading.value || connectionProfileLoading.value || !connectionProfileLoaded.value) {
      ui.showToast('配置尚未成功加载或正在保存，请稍后重试', 'warning')
      return false
    }
    let draft
    try {
      draft = connectionDraft()
    } catch (_) {
      ui.showToast(connectionValidationError.value, 'warning')
      return false
    }
    if (!draft.baseURL || !draft.model) {
      ui.showToast('请先填写接口地址和模型', 'warning')
      return false
    }
    const version = ++connectionSaveRequestVersion
    const requestSession = sessionRevision
    const requestRevision = connectionRevision
    const isCurrent = () => (
      version === connectionSaveRequestVersion &&
      requestSession === sessionRevision &&
      requestRevision === connectionRevision
    )
    connectionSaveLoading.value = true
    try {
      const result = await api.saveConnectionSettings({
        id: selectedConnectionProfileID.value,
        name: connectionProfileName.value,
        connection: draft,
      })
      if (!isCurrent()) return false
      if (!result?.success || result?.error || !result?.connection || !result?.profile) {
        ui.showToast(result?.error?.message || 'API 设置保存失败，原配置未修改', 'error')
        return false
      }
      const canonical = result.connection
      const savedConnection = {
        connectionMode: 'direct',
        apiKey: draft.apiKey,
        baseURL: canonical.baseURL || draft.baseURL,
        apiProtocol: canonical.apiProtocol || draft.apiProtocol,
        endpointPath: canonical.endpointPath ?? draft.endpointPath,
        extraHeaders: { ...draft.extraHeaders },
        responseFormat: canonical.responseFormat || draft.responseFormat,
        model: canonical.model || draft.model,
        requestTemplate: draft.requestTemplate,
        responseJSONPointer: draft.responseJSONPointer,
        authHeader: draft.authHeader,
        authPrefix: draft.authPrefix,
      }
      Object.assign(settings, savedConnection)
      Object.assign(tempSettings, JSON.parse(JSON.stringify(savedConnection)))
      extraHeadersText.value = formatExtraHeaders(savedConnection.extraHeaders)
      const summary = JSON.parse(JSON.stringify(result.profile))
      const existingIndex = connectionProfiles.value.findIndex(profile => profile.id === summary.id)
      const nextProfiles = connectionProfiles.value
        .filter(profile => profile.id !== 'legacy-current')
        .map(profile => ({ ...profile, active: false }))
      if (existingIndex >= 0) {
        const replacementIndex = nextProfiles.findIndex(profile => profile.id === summary.id)
        if (replacementIndex >= 0) nextProfiles[replacementIndex] = { ...summary, active: true }
        else nextProfiles.push({ ...summary, active: true })
      } else {
        nextProfiles.push({ ...summary, active: true })
      }
      connectionProfiles.value = nextProfiles
      activeConnectionProfileID.value = summary.id
      selectedConnectionProfileID.value = summary.id
      connectionProfileName.value = summary.name
      connectionProfileLoaded.value = true
      backendSnapshot = { ...backendSnapshot, activeConnectionProfileId: summary.id }
      legacyMigrationNotice.value = ''
      resetStatus()
      ui.showToast('API 设置已保存', 'success')
      return true
    } catch (_) {
      if (!isCurrent()) return false
      ui.showToast('API 设置保存失败，请检查应用状态后重试', 'error')
      return false
    } finally {
      if (version === connectionSaveRequestVersion) connectionSaveLoading.value = false
    }
  }

  async function saveSettings({ close = true } = {}) {
    if (!settingsLoaded.value || settingsLoading.value || connectionProfileLoading.value || !connectionProfileLoaded.value) {
      ui.showToast('配置尚未成功加载，请先重新读取，未保存任何默认空值', 'warning')
      return false
    }
    let connection
    try {
      connection = connectionDraft()
    } catch (_) {
      ui.showToast(connectionValidationError.value, 'warning')
      return false
    }
    const draft = JSON.parse(JSON.stringify(tempSettings))
    Object.assign(draft, connection)
    const draftShortcuts = JSON.parse(JSON.stringify(tempShortcuts))
    const payload = buildConfigToSave(draft, draftShortcuts, selectedConnectionProfileID.value, connectionProfileName.value)
    try {
      const err = await api.syncSettings(JSON.stringify(payload))
      if (err) {
        ui.showToast('设置保存失败，原界面配置未变，请重试', 'error')
        return false
      }
      const safeSnapshot = { ...payload }
      for (const key of ['connectionProfileId', 'connectionProfileName', 'apiKey', 'baseURL', 'connectionMode', 'apiProtocol', 'endpointPath', 'extraHeaders', 'responseFormat', 'requestTemplate', 'responseJSONPointer', 'authHeader', 'authPrefix', 'model']) {
        delete safeSnapshot[key]
      }
      backendSnapshot = safeSnapshot
      Object.assign(shortcuts, draftShortcuts)
      Object.assign(settings, draft)
      const profilesReloaded = await loadConnectionProfilesForSettings('')
      if (!profilesReloaded) {
        settingsLoaded.value = false
        settingsLoadError.value = '设置已保存，但连接记录重新读取失败，请重新打开设置。'
        ui.showToast(settingsLoadError.value, 'warning')
        return true
      }
      resetStatus()
      ui.showToast('设置已保存', 'success')
      if (close) closeSettings()
      return true
    } catch (_) {
      ui.showToast('保存失败，请检查应用状态后重试', 'error')
      return false
    }
  }

  async function openSettings() {
    api.restoreFocus()
    if (!settingsLoaded.value) await loadSettings()
    resetTempSettings()
    Object.assign(tempShortcuts, JSON.parse(JSON.stringify(shortcuts)))
    sessionRevision++
    connectionRevision++
    catalogRevision++
    probeRevision++
    clearConnectionResults()
    if (settings.resumeContent) resumeRawContent.value = settings.resumeContent
    ui.showSettings = true
  }

  function closeSettings() {
    if (isResumeParsing.value) {
      ui.showToast('简历正在解析中，请稍候...', 'warning')
      return
    }
    sessionRevision++
    connectionRevision++
    connectionSaveRequestVersion++
    profileLoadRequestVersion++
    connectionSaveLoading.value = false
    connectionProfileLoading.value = false
    catalogRevision++
    probeRevision++
    clearConnectionResults()
    api.removeFocus()
    ui.showSettings = false
    if (recordingAction.value) api.stopRecordingKey()
    recordingAction.value = null
    recordingText.value = ''
    resetTempSettings()
  }

  function resetTempSettings() {
    Object.assign(tempSettings, JSON.parse(JSON.stringify(settings)))
    applyLegacyConnectionDraft(settings)
    selectedConnectionProfileID.value = activeConnectionProfileID.value
    const active = activeProfileSummary()
    connectionProfileName.value = active?.id === 'legacy-current' ? '' : (active?.name || '')
    pendingDeleteProfileID.value = ''
    connectionProfileLoaded.value = settingsLoaded.value
    connectionValidationError.value = ''
    applyTransparency(1.0 - settings.transparency)
  }

  function recordKey(action) {
    if (isMacOS.value) return
    recordingAction.value = action
    recordingText.value = '请按键...'
    api.startRecordingKey(action)
  }

  async function selectResume() {
    const path = await api.selectResume()
    if (path) {
      tempSettings.resumePath = path
      resumeRawContent.value = ''
      ui.showToast('已选择文件，正在自动解析...', 'info')
      await parseResume()
    }
  }

  async function clearResume() {
    await api.clearResume()
    tempSettings.resumePath = ''
    resumeRawContent.value = ''
  }

  async function parseResume() {
    if (!tempSettings.resumePath) return
    isResumeParsing.value = true
    try {
      const result = await api.parseResume()
      resumeRawContent.value = result
      tempSettings.resumeContent = result
      ui.showToast('简历解析成功', 'success')
    } catch (e) {
      console.error(e)
      ui.showToast(e?.message || '简历解析失败', 'error')
    } finally {
      isResumeParsing.value = false
    }
  }

  async function saveSettingsSilent() {
    const active = activeProfileSummary()
    if (!settingsLoaded.value || settingsLoading.value || !connectionProfileLoaded.value || !active) return false
    try {
      const profileName = active.id === 'legacy-current' ? '' : active.name
      const payload = buildConfigToSave(settings, shortcuts, activeConnectionProfileID.value, profileName)
      const err = await api.syncSettings(JSON.stringify(payload))
      if (err) {
        ui.showToast('主题设置保存失败', 'warning')
        return false
      }
      return true
    } catch (e) {
      ui.showToast('主题设置保存失败', 'warning')
      return false
    }
  }

  function reloadConnectionProfiles() {
    return loadConnectionProfilesForSettings(activeConnectionProfileID.value)
  }

  return {
    settings, tempSettings, settingsLoaded, settingsLoading, settingsLoadError,
    extraHeadersText, connectionValidationError, connectionSaveLoading,
    legacyMigrationNotice, isCustomTemplateMode, canRefreshModelCatalog, defaultCustomTemplate,
    connectionProfiles, activeConnectionProfileID, selectedConnectionProfileID,
    connectionProfileName, connectionProfileLoading, connectionProfileLoaded,
    pendingDeleteProfileID, automaticConnectionProfileName,
    shortcuts, tempShortcuts,
    domainCategories,
    resumeRawContent, isResumeParsing,
    isMacOS,
    recordingAction, recordingText, shortcutActions, shortcutDisplayName,
    maskedKey, solveShortcut, sendShortcut, fastSendShortcut, cancelSolveShortcut, deleteShortcut, toggleShortcut,
    statusText, statusIcon, resetStatus,
    loadSettings, fetchModels, refreshModels, ensureCCRModelCatalog, testConnection,
    testTextConnection, testVisionConnection, applyProviderPreset, applyConnectionPreset,
    selectConnectionProfile, beginNewConnectionProfile, reloadConnectionProfiles,
    requestDeleteConnectionProfile, cancelDeleteConnectionProfile, confirmDeleteConnectionProfile,
    saveConnectionSettings, saveSettings, saveSettingsSilent, openSettings, closeSettings, resetTempSettings,
    recordKey, selectResume, clearResume, parseResume,
    applyTransparency,
  }
})
