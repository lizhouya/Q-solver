import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import { api } from '../services/api'
import { useUIStore } from './ui'

const emptyStatus = () => ({
  archiveDir: '',
  root: '',
  session: null,
  unsaved: false,
  busy: false,
  organizing: false,
  operation: '',
  progress: {},
  lastExport: {},
  message: '',
})

const emptyPreferences = () => ({
  version: 2,
  archiveDir: '',
  autoOrganize: false,
})

const emptyExportDraft = () => ({
  sessionId: '',
  company: '',
  type: '综合笔试',
  name: '',
  directory: '',
  autoOrganize: false,
  numberedPrefix: '',
  count: 0,
  startedAt: '',
})

function normalizePreferences(value) {
  return {
    version: value?.version || 2,
    archiveDir: value?.archiveDir || '',
    autoOrganize: value?.autoOrganize === true,
  }
}

function normalizeExportDraft(value) {
  return {
    ...emptyExportDraft(),
    ...(value || {}),
    autoOrganize: value?.autoOrganize === true,
    count: Number(value?.count || 0),
  }
}

function errorMessage(error) {
  return error?.message || String(error || '未知错误')
}

function safeNamePart(value, fallback) {
  const cleaned = String(value || '')
    .replace(/[<>:"/\\|?*\x00-\x1f]/g, '_')
    .trim()
    .replace(/[. ]+$/g, '')
  return (cleaned || fallback).slice(0, 40)
}

export const useQuestionBankStore = defineStore('questionbank', () => {
  const ui = useUIStore()
  const visible = ref(false)
  const status = ref(emptyStatus())
  const sessions = ref([])
  const preferences = reactive(emptyPreferences())
  const tempPreferences = reactive(emptyPreferences())
  const working = ref(false)
  const preferenceWorking = ref(false)
  const error = ref('')

  const exportVisible = ref(false)
  const exportDraft = reactive(emptyExportDraft())
  const exportLoading = ref(false)
  const exportSubmitting = ref(false)
  const exportDirectorySelecting = ref(false)
  const exportError = ref('')
  const exportNameEdited = ref(false)

  let refreshVersion = 0
  let exportRequestVersion = 0

  const archiveDir = computed(() => (
    status.value.archiveDir || status.value.root || preferences.archiveDir
  ))
  const configured = computed(() => Boolean(archiveDir.value))
  const sessionState = computed(() => status.value.session?.state || '')
  const busy = computed(() => (
    working.value ||
    preferenceWorking.value ||
    exportLoading.value ||
    exportSubmitting.value ||
    exportDirectorySelecting.value ||
    status.value.busy ||
    status.value.organizing
  ))
  const recording = computed(() => (
    configured.value && ['active', 'paused'].includes(sessionState.value)
  ))
  const canEndCurrent = computed(() => (
    configured.value &&
    Boolean(status.value.session) &&
    Number(status.value.session?.count || 0) > 0 &&
    ['active', 'paused'].includes(sessionState.value) &&
    !status.value.unsaved &&
    !busy.value
  ))
  const canExportCurrent = computed(() => (
    configured.value &&
    Boolean(status.value.session) &&
    Number(status.value.session?.count || 0) > 0 &&
    ['active', 'paused', 'ended'].includes(sessionState.value) &&
    !status.value.unsaved &&
    !busy.value
  ))
  const canDiscardCurrent = computed(() => (
    configured.value &&
    Boolean(status.value.session) &&
    ['active', 'paused', 'ended'].includes(sessionState.value) &&
    !status.value.unsaved &&
    !busy.value
  ))
  const label = computed(() => {
    if (!configured.value) return '题库 · 未配置'
    if (status.value.unsaved) return '题库 · 本段未保存'
    if (status.value.organizing) {
      if (status.value.operation === 'exporting') return '题库 · 导出中'
      const done = status.value.progress?.done || 0
      const total = status.value.progress?.total || 0
      return total > 0 ? `题库 · 整理中 ${done}/${total}` : '题库 · 整理中'
    }
    if (!status.value.session) return '题库 · 待记录'
    const count = status.value.session.count || 0
    if (['active', 'paused'].includes(sessionState.value)) return `题库 · 本段 ${count} 次提交`
    if (sessionState.value === 'exported') return '题库 · 上段已导出'
    if (sessionState.value === 'ended') return '题库 · 上段已结束'
    if (sessionState.value === 'discarded') return '题库 · 上段已清空'
    return '题库 · 待记录'
  })
  const preferencesDirty = computed(() => (
    tempPreferences.archiveDir !== preferences.archiveDir ||
    tempPreferences.autoOrganize !== preferences.autoOrganize
  ))
  const exportExpectedFile = computed(() => {
    if (!exportDraft.autoOrganize) return '仅生成 原始记录.md 与 assets/'
    const name = exportDraft.name.trim() || '题库'
    return `生成 ${name}.md、原始记录.md 与 assets/`
  })

  function reportError(prefix, cause) {
    const message = `${prefix}：${errorMessage(cause)}`
    error.value = message
    ui.showToast(message, 'error', 4000)
  }

  function setStatus(next) {
    status.value = {
      ...emptyStatus(),
      ...(next || {}),
      session: next?.session || null,
      progress: next?.progress || {},
      lastExport: next?.lastExport || {},
    }
  }

  async function refresh() {
    const version = ++refreshVersion
    try {
      const next = await api.getQuestionBankStatus()
      if (version === refreshVersion) setStatus(next)
      return true
    } catch (cause) {
      if (version === refreshVersion) reportError('读取题库状态失败', cause)
      return false
    }
  }

  async function loadSessions() {
    try {
      sessions.value = await api.listQuestionBankSessions() || []
      return true
    } catch (cause) {
      reportError('读取题库历史失败', cause)
      return false
    }
  }

  async function loadPreferences(resetTemporary = true) {
    if (preferenceWorking.value) return false
    preferenceWorking.value = true
    try {
      const next = normalizePreferences(await api.getQuestionBankPreferences())
      Object.assign(preferences, next)
      if (resetTemporary) Object.assign(tempPreferences, next)
      return true
    } catch (cause) {
      reportError('读取题库偏好失败', cause)
      return false
    } finally {
      preferenceWorking.value = false
    }
  }

  async function beginPreferenceEdit() {
    error.value = ''
    return loadPreferences(true)
  }

  async function selectOutput() {
    if (preferenceWorking.value) return false
    preferenceWorking.value = true
    error.value = ''
    try {
      const selected = await api.selectQuestionBankOutput()
      if (selected) tempPreferences.archiveDir = selected
      return Boolean(selected)
    } catch (cause) {
      reportError('选择题库目录失败', cause)
      return false
    } finally {
      preferenceWorking.value = false
    }
  }

  async function savePreferences({ notify = false } = {}) {
    if (preferenceWorking.value) return false
    preferenceWorking.value = true
    error.value = ''
    const payload = {
      version: tempPreferences.version || 2,
      archiveDir: tempPreferences.archiveDir.trim(),
      company: '',
      autoOrganize: tempPreferences.autoOrganize === true,
    }
    try {
      const saved = await api.updateQuestionBankPreferences(payload)
      Object.assign(preferences, normalizePreferences(saved || payload))
      Object.assign(tempPreferences, preferences)
      await refresh()
      if (notify) ui.showToast('题库偏好已保存', 'success')
      return true
    } catch (cause) {
      reportError('保存题库偏好失败', cause)
      return false
    } finally {
      preferenceWorking.value = false
    }
  }

  async function initialize() {
    await Promise.all([refresh(), loadPreferences(true)])
  }

  async function open() {
    visible.value = true
    error.value = ''
    try {
      await api.restoreFocus()
    } catch (cause) {
      reportError('打开题库面板失败', cause)
    }
    await Promise.all([refresh(), loadSessions()])
  }

  function close() {
    visible.value = false
    if (!exportVisible.value && !ui.showSettings) api.removeFocus().catch(() => {})
  }

  async function run(action, { successMessage = '', refreshHistory = true } = {}) {
    if (working.value) return false
    working.value = true
    error.value = ''
    try {
      const result = await action()
      const message = typeof successMessage === 'function' ? successMessage(result) : successMessage
      if (message) ui.showToast(message, 'success', 3000)
      return result ?? true
    } catch (cause) {
      reportError('题库操作失败', cause)
      return false
    } finally {
      working.value = false
      await refresh()
      if (refreshHistory) await loadSessions()
    }
  }

  function suggestedExportName() {
    const kind = safeNamePart(exportDraft.type, '综合笔试')
    const company = String(exportDraft.company || '').trim()
    if (company) return `${safeNamePart(company, '未指定单位')}_${kind}`
    const prefix = exportDraft.numberedPrefix || '笔试'
    return `${prefix}_${kind}`
  }

  function syncSuggestedExportName() {
    if (!exportNameEdited.value) exportDraft.name = suggestedExportName()
  }

  function setExportCompany(value) {
    exportDraft.company = value
    syncSuggestedExportName()
  }

  function setExportType(value) {
    exportDraft.type = value
    syncSuggestedExportName()
  }

  function setExportName(value) {
    exportDraft.name = value
    exportNameEdited.value = true
  }

  async function openExport(id = '') {
    if (!configured.value) {
      error.value = '请先在设置的“题库”标签中选择默认题库目录。'
      ui.showToast(error.value, 'warning', 3500)
      return false
    }
    if (exportSubmitting.value) return false
    const version = ++exportRequestVersion
    exportVisible.value = true
    exportLoading.value = true
    exportError.value = ''
    exportNameEdited.value = false
    Object.assign(exportDraft, emptyExportDraft())
    api.restoreFocus().catch(() => {})
    try {
      const draft = await api.getQuestionBankExportDraft(id || '')
      if (version !== exportRequestVersion || !exportVisible.value) return false
      Object.assign(exportDraft, normalizeExportDraft(draft))
      return true
    } catch (cause) {
      if (version === exportRequestVersion) exportError.value = `读取导出信息失败：${errorMessage(cause)}`
      return false
    } finally {
      if (version === exportRequestVersion) exportLoading.value = false
    }
  }

  function closeExport() {
    if (exportSubmitting.value) return false
    exportRequestVersion++
    exportVisible.value = false
    exportLoading.value = false
    exportDirectorySelecting.value = false
    exportError.value = ''
    exportNameEdited.value = false
    Object.assign(exportDraft, emptyExportDraft())
    if (!visible.value && !ui.showSettings) api.removeFocus().catch(() => {})
    return true
  }

  async function selectExportDirectory() {
    if (exportDirectorySelecting.value || exportSubmitting.value) return false
    const version = exportRequestVersion
    exportDirectorySelecting.value = true
    exportError.value = ''
    try {
      const selected = await api.selectQuestionBankExportDirectory(exportDraft.directory || archiveDir.value)
      if (selected && version === exportRequestVersion && exportVisible.value) {
        exportDraft.directory = selected
      }
      return Boolean(selected)
    } catch (cause) {
      if (version === exportRequestVersion) exportError.value = `选择本次导出目录失败：${errorMessage(cause)}`
      return false
    } finally {
      if (version === exportRequestVersion) exportDirectorySelecting.value = false
    }
  }

  async function submitExport() {
    if (exportSubmitting.value || exportLoading.value) return false
    const name = exportDraft.name.trim()
    if (!name) {
      exportError.value = '请填写题库名称。'
      return false
    }
    if (!exportDraft.directory.trim()) {
      exportError.value = '请选择本次导出目录。'
      return false
    }
    const version = exportRequestVersion
    const request = {
      sessionId: exportDraft.sessionId,
      company: exportDraft.company.trim(),
      type: exportDraft.type.trim(),
      name,
      directory: exportDraft.directory.trim(),
      autoOrganize: exportDraft.autoOrganize === true,
    }
    exportSubmitting.value = true
    exportError.value = ''
    try {
      await api.exportQuestionBankSegment(request)
      if (version !== exportRequestVersion) return false
      exportVisible.value = false
      ui.showToast(request.autoOrganize ? '已开始整理并导出所选分段。' : '已开始导出原始记录。', 'success', 3000)
      await refresh()
      if (visible.value) await loadSessions()
      if (!visible.value && !ui.showSettings) api.removeFocus().catch(() => {})
      return true
    } catch (cause) {
      if (version === exportRequestVersion) exportError.value = `提交导出失败：${errorMessage(cause)}`
      return false
    } finally {
      if (version === exportRequestVersion) exportSubmitting.value = false
    }
  }

  async function exportCurrent() {
    return openExport('')
  }

  async function exportSession(id) {
    if (!id) return false
    return openExport(id)
  }

  async function splitCurrent() {
    return run(
      () => api.splitQuestionBankSegment(),
      { successMessage: '当前段已结束，下一题将自动进入新段。' },
    )
  }

  async function discardCurrent(confirmed) {
    if (!confirmed) return false
    return run(
      () => api.discardQuestionBankSegment(true),
      { successMessage: '当前分段已清空，原始记录仍保留在本地。' },
    )
  }

  async function openSession(id) {
    if (!id) return false
    return run(() => api.openQuestionBankSession(id), { refreshHistory: false })
  }

  async function retrySave() {
    return run(
      () => api.saveQuestionBankSession(),
      { successMessage: '当前题库分段已保存。' },
    )
  }

  async function cancelOrganizing() {
    return run(
      () => api.cancelQuestionBankOrganizing(),
      { successMessage: '已请求取消题库整理。', refreshHistory: false },
    )
  }

  return {
    visible,
    status,
    sessions,
    preferences,
    tempPreferences,
    working,
    preferenceWorking,
    error,
    exportVisible,
    exportDraft,
    exportLoading,
    exportSubmitting,
    exportDirectorySelecting,
    exportError,
    exportNameEdited,
    exportExpectedFile,
    archiveDir,
    configured,
    sessionState,
    busy,
    recording,
    canEndCurrent,
    canExportCurrent,
    canDiscardCurrent,
    label,
    preferencesDirty,
    refresh,
    loadSessions,
    loadPreferences,
    beginPreferenceEdit,
    selectOutput,
    savePreferences,
    initialize,
    open,
    close,
    run,
    openExport,
    closeExport,
    selectExportDirectory,
    submitExport,
    setExportCompany,
    setExportType,
    setExportName,
    exportCurrent,
    exportSession,
    splitCurrent,
    discardCurrent,
    openSession,
    retrySave,
    cancelOrganizing,
  }
})
