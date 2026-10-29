import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import { api } from '../services/api'
import { useUIStore } from './ui'
import { useSolutionStore } from './solution'

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
  issuer: '',
  type: '综合笔试',
  name: '',
  nameCustomized: false,
  directory: '',
  autoOrganize: false,
  numberedPrefix: '',
  count: 0,
  startedAt: '',
  metadataSuggestion: null,
  suggestionConflict: false,
  organizeReady: false,
  organizeOutcome: 'pending',
  eligibleOrganize: 0,
  organizedRecords: 0,
  structuredQuestions: 0,
  pendingOrganize: 0,
  failedOrganize: 0,
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
    nameCustomized: value?.nameCustomized === true,
    suggestionConflict: value?.suggestionConflict === true,
    organizeReady: value?.organizeReady === true,
    organizeOutcome: value?.organizeOutcome || 'pending',
    eligibleOrganize: Number(value?.eligibleOrganize || 0),
    organizedRecords: Number(value?.organizedRecords || 0),
    structuredQuestions: Number(value?.structuredQuestions || 0),
    pendingOrganize: Number(value?.pendingOrganize || 0),
    failedOrganize: Number(value?.failedOrganize || 0),
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
  const solution = useSolutionStore()
  const visible = ref(false)
  const status = ref(emptyStatus())
  const sessions = ref([])
  const sessionTotal = ref(0)
  const sessionHidden = ref(0)
  const sessionLimit = ref(20)
  const selectedSessionId = ref('')
  const sessionDetail = ref(null)
  const detailLoading = ref(false)
  const detailError = ref('')
  const continueSessionWorking = ref('')
  const imageCache = reactive({})
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
  const exportTouched = reactive({ company: false, issuer: false, type: false, name: false })

  const nextExamPending = ref(false)
  const nextExamSubmitting = ref(false)
  const nextExamError = ref('')

  let refreshVersion = 0
  let sessionsRequestVersion = 0
  let detailRequestVersion = 0
  let continuationRequestVersion = 0
  let exportRequestVersion = 0
  const imageRequestVersions = new Map()

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
    nextExamSubmitting.value ||
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
    return `生成逐题解析整理稿 ${name}.md、原始记录.md 与 assets/`
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
      if (version === refreshVersion) {
        const finishedOrganizing = status.value.organizing && !next?.organizing
        setStatus(next)
        if (finishedOrganizing && exportVisible.value && exportDraft.autoOrganize) await reloadExportDraft()
      }
      return true
    } catch (cause) {
      if (version === refreshVersion) reportError('读取题库状态失败', cause)
      return false
    }
  }

  function clearSessionDetail() {
    detailRequestVersion++
    selectedSessionId.value = ''
    sessionDetail.value = null
    detailLoading.value = false
    detailError.value = ''
    Object.keys(imageCache).forEach(key => delete imageCache[key])
    imageRequestVersions.clear()
  }

  async function loadSessions() {
    const version = ++sessionsRequestVersion
    try {
      const result = await api.listRecentQuestionBankSessions()
      if (version !== sessionsRequestVersion) return false
      sessions.value = result?.sessions || []
      sessionTotal.value = Number(result?.total || sessions.value.length)
      sessionHidden.value = Number(result?.hidden || 0)
      sessionLimit.value = Number(result?.limit || 20)
      if (selectedSessionId.value && !sessions.value.some(item => item.id === selectedSessionId.value)) {
        clearSessionDetail()
      } else if (selectedSessionId.value) {
        await selectSession(selectedSessionId.value)
      }
      return true
    } catch (cause) {
      if (version === sessionsRequestVersion) reportError('读取题库历史失败', cause)
      return false
    }
  }

  async function selectSession(id) {
    if (!id || !sessions.value.some(item => item.id === id)) return false
    const version = ++detailRequestVersion
    selectedSessionId.value = id
    sessionDetail.value = null
    detailLoading.value = true
    detailError.value = ''
    Object.keys(imageCache).forEach(key => delete imageCache[key])
    imageRequestVersions.clear()
    try {
      const detail = await api.getQuestionBankSessionDetail(id)
      if (version !== detailRequestVersion || selectedSessionId.value !== id || !visible.value) return false
      sessionDetail.value = detail
      return true
    } catch (cause) {
      if (version === detailRequestVersion && selectedSessionId.value === id) detailError.value = errorMessage(cause)
      return false
    } finally {
      if (version === detailRequestVersion) detailLoading.value = false
    }
  }

  async function loadRecordImage(recordId, index) {
    const sessionId = selectedSessionId.value
    if (!sessionId || !recordId) return false
    const key = `${recordId}:${index}`
    const version = (imageRequestVersions.get(key) || 0) + 1
    imageRequestVersions.set(key, version)
    imageCache[key] = { loading: true, data: '', error: '' }
    try {
      const data = await api.getQuestionBankRecordImage(sessionId, recordId, index)
      if (selectedSessionId.value !== sessionId || imageRequestVersions.get(key) !== version || !visible.value) return false
      imageCache[key] = { loading: false, data, error: '' }
      return true
    } catch (cause) {
      if (selectedSessionId.value === sessionId && imageRequestVersions.get(key) === version) {
        imageCache[key] = { loading: false, data: '', error: errorMessage(cause) }
      }
      return false
    }
  }

  async function continueSession(id) {
    if (!id || continueSessionWorking.value || busy.value) return false
    const session = sessions.value.find(item => item.id === id)
    if (!session || !['ended', 'exported', 'paused'].includes(session.state)) return false
    const version = ++continuationRequestVersion
    continueSessionWorking.value = id
    error.value = ''
    try {
      const continuation = await api.continueQuestionBankSession(id)
      if (version !== continuationRequestVersion || continueSessionWorking.value !== id) return false
      if (!solution.activateQuestionBankSession(continuation)) throw new Error('续接结果缺少分段详情')
      ui.hasStarted = true
      ui.mainVisible = true
      await refresh()
      close()
      ui.showToast('已切换到历史分段，下一次解题会继续写入该分段。', 'success', 3500)
      return true
    } catch (cause) {
      if (version === continuationRequestVersion) reportError('继续历史分段失败', cause)
      return false
    } finally {
      if (version === continuationRequestVersion) continueSessionWorking.value = ''
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
    sessionsRequestVersion++
    clearSessionDetail()
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
    const values = [exportDraft.company, exportDraft.issuer]
      .map(value => String(value || '').trim())
      .filter((value, index, items) => value && items.findIndex(item => item.toLocaleLowerCase() === value.toLocaleLowerCase()) === index)
    if (!values.length) values.push(exportDraft.numberedPrefix || '笔试')
    const kind = String(exportDraft.type || '综合笔试').trim() || '综合笔试'
    if (!values.some(value => value.toLocaleLowerCase() === kind.toLocaleLowerCase())) values.push(kind)
    return values.map(value => safeNamePart(value, '未命名')).join('_')
  }

  function syncSuggestedExportName() {
    if (!exportNameEdited.value) exportDraft.name = suggestedExportName()
  }

  function setExportCompany(value) {
    exportTouched.company = true
    exportDraft.company = value
    syncSuggestedExportName()
  }

  function setExportIssuer(value) {
    exportTouched.issuer = true
    exportDraft.issuer = value
    syncSuggestedExportName()
  }

  function setExportType(value) {
    exportTouched.type = true
    exportDraft.type = value
    syncSuggestedExportName()
  }

  function setExportName(value) {
    exportTouched.name = true
    exportDraft.name = value
    exportDraft.nameCustomized = true
    exportNameEdited.value = true
  }

  function restoreAutomaticExportName() {
    exportTouched.name = false
    exportNameEdited.value = false
    exportDraft.nameCustomized = false
    syncSuggestedExportName()
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
    Object.assign(exportTouched, { company: false, issuer: false, type: false, name: false })
    Object.assign(exportDraft, emptyExportDraft())
    api.restoreFocus().catch(() => {})
    try {
      const draft = await api.getQuestionBankExportDraft(id || '')
      if (version !== exportRequestVersion || !exportVisible.value) return false
      Object.assign(exportDraft, normalizeExportDraft(draft))
      exportNameEdited.value = exportDraft.nameCustomized
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
    Object.assign(exportTouched, { company: false, issuer: false, type: false, name: false })
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

  async function reloadExportDraft() {
    const version = exportRequestVersion
    const localDraft = { ...exportDraft }
    const touched = { ...exportTouched }
    try {
      const draft = await api.getQuestionBankExportDraft(exportDraft.sessionId)
      if (version !== exportRequestVersion || !exportVisible.value) return false
      Object.assign(exportDraft, normalizeExportDraft(draft))
      exportDraft.autoOrganize = localDraft.autoOrganize
      if (localDraft.directory) exportDraft.directory = localDraft.directory
      for (const field of ['company', 'issuer', 'type']) {
        if (touched[field]) exportDraft[field] = localDraft[field]
      }
      if (touched.name || localDraft.nameCustomized) {
        exportDraft.name = localDraft.name
        exportDraft.nameCustomized = true
        exportNameEdited.value = true
      } else {
        exportNameEdited.value = false
        syncSuggestedExportName()
      }
      return true
    } catch (cause) {
      if (version === exportRequestVersion) exportError.value = `刷新整理建议失败：${errorMessage(cause)}`
      return false
    }
  }

  async function prepareExport() {
    if (exportSubmitting.value || exportLoading.value || status.value.organizing) return false
    const version = exportRequestVersion
    exportSubmitting.value = true
    exportError.value = ''
    try {
      await api.prepareQuestionBankSegment(exportDraft.sessionId)
      if (version !== exportRequestVersion) return false
      ui.showToast('已开始整理；完成后请核对模型建议再导出。', 'success', 3500)
      await refresh()
      return true
    } catch (cause) {
      if (version === exportRequestVersion) exportError.value = `开始整理失败：${errorMessage(cause)}`
      return false
    } finally {
      if (version === exportRequestVersion) exportSubmitting.value = false
    }
  }

  function effectiveOrganizeOutcome() {
    const outcome = exportDraft.organizeOutcome || 'pending'
    if (exportDraft.organizeReady && outcome === 'pending' && exportDraft.pendingOrganize === 0) {
      return exportDraft.failedOrganize > 0 ? 'partial' : 'complete'
    }
    return outcome
  }

  function useRawOnlyExport() {
    if (exportSubmitting.value || exportLoading.value || status.value.organizing) return false
    exportDraft.autoOrganize = false
    exportError.value = ''
    return true
  }

  async function retryFailedExport() {
    if (exportSubmitting.value || exportLoading.value || status.value.organizing || exportDraft.failedOrganize <= 0) return false
    const version = exportRequestVersion
    exportSubmitting.value = true
    exportError.value = ''
    try {
      await api.retryFailedQuestionBankRecords(exportDraft.sessionId)
      if (version !== exportRequestVersion) return false
      ui.showToast('已开始只重试整理失败记录；成功项不会重复调用模型。', 'success', 3500)
      await refresh()
      return true
    } catch (cause) {
      if (version === exportRequestVersion) exportError.value = `重试整理失败：${errorMessage(cause)}`
      return false
    } finally {
      if (version === exportRequestVersion) exportSubmitting.value = false
    }
  }

  async function performExport(allowPartial = false) {
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
      issuer: exportDraft.issuer.trim(),
      type: exportDraft.type.trim(),
      name,
      nameCustomized: exportNameEdited.value,
      directory: exportDraft.directory.trim(),
      autoOrganize: exportDraft.autoOrganize === true,
      allowPartial: allowPartial === true,
    }
    exportSubmitting.value = true
    exportError.value = ''
    try {
      await api.exportQuestionBankSegment(request)
      if (version !== exportRequestVersion) return false
      exportVisible.value = false
      ui.showToast(request.autoOrganize ? '已确认元数据并开始本地导出；不会再次调用模型。' : '已开始导出原始记录。', 'success', 3500)
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

  async function submitExport() {
    if (exportSubmitting.value || exportLoading.value || status.value.organizing) return false
    if (!exportDraft.autoOrganize) return performExport(false)
    const outcome = effectiveOrganizeOutcome()
    if (outcome === 'pending') return prepareExport()
    if (outcome === 'partial') {
      exportError.value = '仍有记录整理失败；请先重试，或点击“导出已有整理结果”。'
      return false
    }
    if (outcome === 'all_failed') {
      exportError.value = '全部整理失败，不能生成整理稿；请只重试失败记录或改为仅原稿。'
      return false
    }
    if (outcome === 'no_eligible') {
      exportError.value = '当前没有可整理的已完成记录，请改为仅原稿。'
      return false
    }
    return performExport(false)
  }

  async function exportPartialResults() {
    if (exportSubmitting.value || exportLoading.value || status.value.organizing) return false
    if (!exportDraft.autoOrganize || effectiveOrganizeOutcome() !== 'partial') return false
    return performExport(true)
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

  function requestStartNextExam() {
    if (nextExamPending.value || nextExamSubmitting.value) return false
    nextExamError.value = ''
    nextExamPending.value = true
    return true
  }

  function cancelStartNextExam() {
    if (nextExamSubmitting.value || !nextExamPending.value) return false
    nextExamPending.value = false
    nextExamError.value = ''
    return true
  }

  async function confirmStartNextExam() {
    if (!nextExamPending.value || nextExamSubmitting.value) return false
    nextExamSubmitting.value = true
    nextExamError.value = ''
    try {
      await api.startNextExam(true)
      solution.resetLiveHistory()
      ui.hasStarted = false
      await refresh()
      nextExamPending.value = false
      ui.showToast('当前段已封存，可以开始下一套笔试。', 'success', 3500)
      return true
    } catch (cause) {
      nextExamError.value = `开始下一套失败：${errorMessage(cause)}`
      return false
    } finally {
      nextExamSubmitting.value = false
    }
  }

  async function discardCurrent(confirmed) {
    if (!confirmed) return false
    const success = await run(
      () => api.discardQuestionBankSegment(true),
      { successMessage: '桌面历史已清空；题库原始记录和模型上下文仍保留。' },
    )
    if (!success) return false
    solution.resetLiveHistory()
    ui.hasStarted = false
    return true
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
    sessionTotal,
    sessionHidden,
    sessionLimit,
    selectedSessionId,
    sessionDetail,
    detailLoading,
    detailError,
    continueSessionWorking,
    imageCache,
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
    nextExamPending,
    nextExamSubmitting,
    nextExamError,
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
    selectSession,
    continueSession,
    clearSessionDetail,
    loadRecordImage,
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
    prepareExport,
    retryFailedExport,
    useRawOnlyExport,
    exportPartialResults,
    submitExport,
    setExportCompany,
    setExportIssuer,
    setExportType,
    setExportName,
    restoreAutomaticExportName,
    exportCurrent,
    exportSession,
    splitCurrent,
    requestStartNextExam,
    cancelStartNextExam,
    confirmStartNextExam,
    discardCurrent,
    openSession,
    retrySave,
    cancelOrganizing,
  }
})
