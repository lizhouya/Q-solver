import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { reactive, ref, computed } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/questionbank.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '')
  .replace('export const useQuestionBankStore', 'const useQuestionBankStore')

function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

const baseStatus = (overrides = {}) => ({
  archiveDir: 'C:/bank',
  root: 'C:/bank',
  session: { id: 'current', state: 'ended', count: 1 },
  unsaved: false,
  busy: false,
  organizing: false,
  operation: '',
  progress: {},
  ...overrides,
})

const baseDraft = (overrides = {}) => ({
  sessionId: 'session-a',
  company: '',
  issuer: '',
  type: '综合笔试',
  name: '笔试_20261023_01_综合笔试',
  nameCustomized: false,
  directory: 'C:/bank',
  autoOrganize: false,
  numberedPrefix: '笔试_20261023_01',
  count: 2,
  startedAt: '2026-10-23T08:00:00Z',
  metadataSuggestion: null,
  suggestionConflict: false,
  organizeOutcome: 'pending',
  eligibleOrganize: 2,
  organizedRecords: 0,
  structuredQuestions: 0,
  organizeReady: false,
  pendingOrganize: 2,
  failedOrganize: 0,
  ...overrides,
})

function setup(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const toasts = []
  const resets = []
  let status = baseStatus()
  let statusReads = 0
  let recent = { sessions: [], total: 0, hidden: 0, limit: 20 }
  let draft = baseDraft()
  const api = {
    getQuestionBankStatus: async () => {
      statusReads += 1
      return status
    },
    listRecentQuestionBankSessions: async () => recent,
    getQuestionBankPreferences: async () => ({ version: 2, archiveDir: 'C:/bank', autoOrganize: false }),
    getQuestionBankSessionDetail: async id => ({ id, records: [] }),
    getQuestionBankRecordImage: async () => 'data:image/png;base64,fixture',
    getQuestionBankExportDraft: async () => draft,
    prepareQuestionBankSegment: async id => {
      calls.push(['prepare', id])
      status = baseStatus({ organizing: true, operation: 'organizing' })
    },
    retryFailedQuestionBankRecords: async id => {
      calls.push(['retry-failed', id])
      status = baseStatus({ organizing: true, operation: 'organizing' })
    },
    exportQuestionBankSegment: async request => { calls.push(['export', { ...request }]) },
    discardQuestionBankSegment: async confirmed => { calls.push(['discard', confirmed]) },
    startNextExam: async confirmed => { calls.push(['start-next-exam', confirmed]) },
    restoreFocus: async () => {},
    removeFocus: async () => {},
    ...overrides,
  }
  const ui = reactive({
    hasStarted: true,
    showSettings: false,
    showToast: (...args) => toasts.push(args),
  })
  const solution = { resetLiveHistory: () => resets.push('reset') }
  const context = vm.createContext({
    defineStore,
    reactive,
    ref,
    computed,
    api,
    useUIStore: () => ui,
    useSolutionStore: () => solution,
  })
  vm.runInContext(source + '\nglobalThis.useQuestionBankStore = useQuestionBankStore', context)
  return {
    store: context.useQuestionBankStore(), api, calls, toasts, resets, ui,
    setStatus: value => { status = value },
    getStatusReads: () => statusReads,
    setRecent: value => { recent = value },
    setDraft: value => { draft = value },
  }
}

// 软清空只有在后端成功后才清桌面实时历史，且不调用其他清理接口。
let h = setup()
assert.equal(await h.store.discardCurrent(false), false)
assert.equal(h.resets.length, 0)
assert.equal(await h.store.discardCurrent(true), true)
assert.deepEqual(h.calls, [['discard', true]])
assert.equal(h.resets.length, 1)
assert.equal(h.ui.hasStarted, false)
for (const forbidden of ['cancelTask', 'clearScreenshots', 'clearMobileDisplay', 'clearHistory']) {
  assert.equal(h.calls.some(call => call[0] === forbidden), false)
}

h = setup({ discardQuestionBankSegment: async () => { throw Error('synthetic discard failure') } })
assert.equal(await h.store.discardCurrent(true), false)
assert.equal(h.resets.length, 0)
assert.equal(h.ui.hasStarted, true)

// “开始下一套”请求是单例确认层；重复请求不创建第二份状态，取消没有任何后端或清理副作用。
h = setup()
assert.equal(h.store.nextExamPending, false)
assert.equal(h.store.requestStartNextExam(), true)
assert.equal(h.store.nextExamPending, true)
assert.equal(h.store.requestStartNextExam(), false)
assert.equal(h.store.cancelStartNextExam(), true)
assert.equal(h.store.nextExamPending, false)
assert.deepEqual(h.calls, [])
assert.equal(h.resets.length, 0)
assert.equal(h.ui.hasStarted, true)

// 确认提交必须防重；提交期间不能取消，也不能由重复请求再打开确认层。
const nextExamRequest = deferred()
let nextExamCalls = 0
h = setup({
  startNextExam: confirmed => {
    nextExamCalls += 1
    assert.equal(confirmed, true)
    return nextExamRequest.promise
  },
})
assert.equal(h.store.requestStartNextExam(), true)
const confirmingNextExam = h.store.confirmStartNextExam()
assert.equal(h.store.nextExamSubmitting, true)
assert.equal(await h.store.confirmStartNextExam(), false)
assert.equal(h.store.cancelStartNextExam(), false)
assert.equal(h.store.requestStartNextExam(), false)
assert.equal(nextExamCalls, 1)
assert.equal(h.resets.length, 0)
nextExamRequest.resolve()
assert.equal(await confirmingNextExam, true)
assert.equal(h.store.nextExamSubmitting, false)
assert.equal(h.store.nextExamPending, false)
assert.equal(h.store.nextExamError, '')
assert.equal(h.resets.length, 1)
assert.equal(h.ui.hasStarted, false)
assert.equal(h.getStatusReads(), 1)

// 后端失败保留确认层和错误，可直接重试；失败前后都不能清桌面历史。
let shouldFailNextExam = true
nextExamCalls = 0
h = setup({
  startNextExam: async confirmed => {
    nextExamCalls += 1
    assert.equal(confirmed, true)
    if (shouldFailNextExam) throw Error('synthetic next exam failure')
  },
})
assert.equal(h.store.requestStartNextExam(), true)
assert.equal(await h.store.confirmStartNextExam(), false)
assert.equal(h.store.nextExamPending, true)
assert.equal(h.store.nextExamSubmitting, false)
assert.match(h.store.nextExamError, /synthetic next exam failure/)
assert.equal(h.resets.length, 0)
assert.equal(h.ui.hasStarted, true)
assert.equal(h.getStatusReads(), 0)
shouldFailNextExam = false
assert.equal(await h.store.confirmStartNextExam(), true)
assert.equal(nextExamCalls, 2)
assert.equal(h.store.nextExamPending, false)
assert.equal(h.resets.length, 1)
assert.equal(h.ui.hasStarted, false)
assert.equal(h.getStatusReads(), 1)
for (const forbidden of ['clearMobileDisplay', 'clearScreenshots', 'cancelTask']) {
  assert.equal(h.calls.some(call => call[0] === forbidden), false)
}

// 最近列表仅接受后端给出的 20 条边界和隐藏计数。
h = setup()
h.setRecent({ sessions: [{ id: 'a' }, { id: 'b' }], total: 21, hidden: 1, limit: 20 })
assert.equal(await h.store.loadSessions(), true)
assert.equal(h.store.sessions.length, 2)
assert.equal(h.store.sessionTotal, 21)
assert.equal(h.store.sessionHidden, 1)
assert.equal(h.store.sessionLimit, 20)

// A→B 快速选择必须 latest-wins，A 的迟到详情不能覆盖 B。
const detailA = deferred()
const detailB = deferred()
h = setup({
  getQuestionBankSessionDetail: id => id === 'a' ? detailA.promise : detailB.promise,
})
h.setRecent({ sessions: [{ id: 'a' }, { id: 'b' }], total: 2, hidden: 0, limit: 20 })
h.store.visible = true
await h.store.loadSessions()
const selectingA = h.store.selectSession('a')
const selectingB = h.store.selectSession('b')
detailB.resolve({ id: 'b', records: [] })
assert.equal(await selectingB, true)
detailA.resolve({ id: 'a', records: [] })
assert.equal(await selectingA, false)
assert.equal(h.store.sessionDetail.id, 'b')

// 切换分段会释放图片缓存，旧分段迟到图片不能回写；同段不同图片互不取消。
const imageA = deferred()
h.api.getQuestionBankRecordImage = () => imageA.promise
const loadingImageA = h.store.loadRecordImage('record-a', 0)
h.api.getQuestionBankSessionDetail = async id => ({ id, records: [] })
await h.store.selectSession('a')
imageA.resolve('data:image/png;base64,late')
assert.equal(await loadingImageA, false)
assert.equal(h.store.imageCache['record-a:0'], undefined)

const image0 = deferred()
const image1 = deferred()
h.api.getQuestionBankRecordImage = (_session, _record, index) => index === 0 ? image0.promise : image1.promise
const loading0 = h.store.loadRecordImage('record-a', 0)
const loading1 = h.store.loadRecordImage('record-a', 1)
image1.resolve('data:image/png;base64,one')
image0.resolve('data:image/png;base64,zero')
assert.equal(await loading0, true)
assert.equal(await loading1, true)
assert.ok(h.store.imageCache['record-a:0'].data.endsWith('zero'))
assert.ok(h.store.imageCache['record-a:1'].data.endsWith('one'))

// 关闭面板后，迟到详情不能重新显示。
const lateDetail = deferred()
h.api.getQuestionBankSessionDetail = () => lateDetail.promise
const lateSelection = h.store.selectSession('b')
h.store.close()
lateDetail.resolve({ id: 'b', records: [{ answer: 'late' }] })
assert.equal(await lateSelection, false)
assert.equal(h.store.sessionDetail, null)
assert.equal(h.store.selectedSessionId, '')

// 自动命名包含单位、出题方、题型；手工名称受保护，显式恢复后才重新联动。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({ company: '单位A', issuer: '机构B', type: '行测', name: '单位A_机构B_行测' }))
await h.store.openExport()
h.store.setExportIssuer('机构C')
assert.equal(h.store.exportDraft.name, '单位A_机构C_行测')
h.store.setExportName('手工名称')
h.store.setExportCompany('单位D')
assert.equal(h.store.exportDraft.name, '手工名称')
h.store.restoreAutomaticExportName()
assert.equal(h.store.exportDraft.name, '单位D_机构C_行测')

// 启用整理时第一次提交仅启动准备阶段，不创建导出；准备完成后最终提交携带核对字段且不再准备。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({ autoOrganize: true, name: '', directory: '', organizeReady: false }))
await h.store.openExport()
assert.equal(await h.store.submitExport(), true)
assert.deepEqual(h.calls, [['prepare', 'session-a']])

h = setup()
await h.store.refresh()
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeReady: true,
  pendingOrganize: 0,
  company: '单位A',
  issuer: '机构B',
  type: '行测',
  name: '单位A_机构B_行测',
}))
await h.store.openExport()
h.store.setExportName('人工核对名称')
assert.equal(await h.store.submitExport(), true)
assert.equal(h.calls.length, 1)
assert.equal(h.calls[0][0], 'export')
assert.equal(h.calls[0][1].issuer, '机构B')
assert.equal(h.calls[0][1].nameCustomized, true)
assert.equal(h.calls.some(call => call[0] === 'prepare'), false)

// 准备阶段完成后的草稿刷新保留用户已编辑字段，同时接收新的整理状态与建议。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({ autoOrganize: true, organizeReady: false }))
await h.store.openExport()
h.store.setExportCompany('人工单位')
h.store.setExportIssuer('人工出题方')
h.store.setExportType('人工题型')
h.store.setExportName('人工名称')
assert.equal(await h.store.submitExport(), true)
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeReady: true,
  pendingOrganize: 0,
  company: '模型单位',
  issuer: '模型出题方',
  type: '模型题型',
  name: '模型单位_模型出题方_模型题型',
  metadataSuggestion: { company: '模型单位', issuer: '模型出题方', type: '模型题型' },
}))
h.setStatus(baseStatus({ organizing: false }))
await h.store.refresh()
assert.equal(h.store.exportDraft.organizeReady, true)
assert.equal(h.store.exportDraft.company, '人工单位')
assert.equal(h.store.exportDraft.issuer, '人工出题方')
assert.equal(h.store.exportDraft.type, '人工题型')
assert.equal(h.store.exportDraft.name, '人工名称')
assert.equal(h.store.exportDraft.nameCustomized, true)
assert.equal(h.store.exportDraft.metadataSuggestion.company, '模型单位')

// 未编辑字段采用模型准备结果，自动名称按刷新后的候选重新生成。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({ autoOrganize: true, organizeReady: false }))
await h.store.openExport()
assert.equal(await h.store.submitExport(), true)
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeReady: true,
  pendingOrganize: 0,
  company: '模型单位',
  issuer: '模型出题方',
  type: '模型题型',
  name: '服务端旧名称',
}))
h.setStatus(baseStatus({ organizing: false }))
await h.store.refresh()
assert.equal(h.store.exportDraft.company, '模型单位')
assert.equal(h.store.exportDraft.name, '模型单位_模型出题方_模型题型')
assert.equal(h.store.exportDraft.nameCustomized, false)

// Change 024：全失败不能冒充准备完成，只能显式重试失败项或切换为仅原稿。
const retryFailed = deferred()
let retryFailedCalls = 0
h = setup({
  retryFailedQuestionBankRecords: id => {
    retryFailedCalls += 1
    h.calls.push(['retry-failed', id])
    return retryFailed.promise
  },
})
await h.store.refresh()
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeOutcome: 'all_failed',
  eligibleOrganize: 2,
  organizedRecords: 0,
  structuredQuestions: 0,
  organizeReady: false,
  pendingOrganize: 0,
  failedOrganize: 2,
}))
await h.store.openExport()
assert.equal(await h.store.submitExport(), false)
assert.deepEqual(h.calls, [])
const retryingFailed = h.store.retryFailedExport()
assert.equal(await h.store.retryFailedExport(), false)
assert.equal(retryFailedCalls, 1)
retryFailed.resolve()
assert.equal(await retryingFailed, true)
assert.deepEqual(h.calls, [['retry-failed', 'session-a']])

// “改为仅原稿”只改本弹窗草稿，不调用偏好 API，随后按 RawOnly 导出。
const preferenceBeforeRawOnly = { ...h.store.preferences }
assert.equal(h.store.useRawOnlyExport(), true)
assert.equal(h.store.exportDraft.autoOrganize, false)
assert.deepEqual({ ...h.store.preferences }, preferenceBeforeRawOnly)
assert.equal(h.calls.length, 1)
assert.equal(await h.store.submitExport(), true)
assert.equal(h.calls[1][0], 'export')
assert.equal(h.calls[1][1].autoOrganize, false)
assert.equal(h.calls[1][1].allowPartial, false)

// partial 必须通过明确动作导出已有结果，并携带 allowPartial=true；普通提交拒绝。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeOutcome: 'partial',
  eligibleOrganize: 5,
  organizedRecords: 3,
  structuredQuestions: 8,
  organizeReady: true,
  pendingOrganize: 0,
  failedOrganize: 2,
}))
await h.store.openExport()
assert.equal(h.store.exportDraft.organizedRecords, 3)
assert.equal(h.store.exportDraft.failedOrganize, 2)
assert.equal(h.store.exportDraft.structuredQuestions, 8)
assert.equal(await h.store.submitExport(), false)
assert.deepEqual(h.calls, [])
assert.equal(await h.store.exportPartialResults(), true)
assert.equal(h.calls.length, 1)
assert.equal(h.calls[0][0], 'export')
assert.equal(h.calls[0][1].allowPartial, true)
assert.equal(h.calls[0][1].autoOrganize, true)

// complete 正常导出且不声明部分导出；no_eligible 不启动整理，只能切换仅原稿。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeOutcome: 'complete',
  eligibleOrganize: 2,
  organizedRecords: 2,
  structuredQuestions: 6,
  organizeReady: true,
  pendingOrganize: 0,
}))
await h.store.openExport()
assert.equal(await h.store.submitExport(), true)
assert.equal(h.calls[0][1].allowPartial, false)

h = setup()
await h.store.refresh()
h.setDraft(baseDraft({
  autoOrganize: true,
  organizeOutcome: 'no_eligible',
  eligibleOrganize: 0,
  organizeReady: false,
  pendingOrganize: 0,
}))
await h.store.openExport()
assert.equal(await h.store.submitExport(), false)
assert.deepEqual(h.calls, [])
assert.equal(h.store.useRawOnlyExport(), true)
assert.equal(await h.store.submitExport(), true)
assert.equal(h.calls[0][1].autoOrganize, false)

// 草稿刷新保留本次 autoOrganize 选择和目录，不被后端长期偏好值覆盖。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({ autoOrganize: true, directory: 'C:/initial', organizeReady: false }))
await h.store.openExport()
h.store.exportDraft.directory = 'D:/one-off'
assert.equal(await h.store.submitExport(), true)
h.setDraft(baseDraft({
  autoOrganize: false,
  directory: 'C:/preference',
  organizeOutcome: 'complete',
  eligibleOrganize: 2,
  organizedRecords: 2,
  structuredQuestions: 4,
  organizeReady: true,
  pendingOrganize: 0,
}))
h.setStatus(baseStatus({ organizing: false }))
await h.store.refresh()
assert.equal(h.store.exportDraft.autoOrganize, true)
assert.equal(h.store.exportDraft.directory, 'D:/one-off')

// RawOnly 仍是单阶段、零模型准备。
h = setup()
await h.store.refresh()
h.setDraft(baseDraft({ autoOrganize: false, organizeReady: false }))
await h.store.openExport()
assert.equal(await h.store.submitExport(), true)
assert.equal(h.calls[0][0], 'export')
assert.equal(h.calls[0][1].autoOrganize, false)
assert.equal(h.calls.some(call => call[0] === 'prepare'), false)

// 静态模板约束：只读文本不使用 v-html，重新导出阻止冒泡，两阶段文案完整。
const panel = fs.readFileSync(new URL('./src/components/QuestionBankPanel.vue', import.meta.url), 'utf8')
assert.equal(panel.includes('v-html'), false)
assert.ok(panel.includes('@click.stop="bank.exportSession(session.id)"'))
assert.ok(panel.includes('@click="browseSession(session.id)"'))
assert.ok(panel.includes('detailPanel.value?.scrollIntoView'))
assert.ok(panel.includes('session.id === bank.selectedSessionId && bank.detailLoading">正在读取'))
assert.ok(panel.includes('session.id === bank.sessionDetail?.id">正在浏览'))
assert.ok(panel.includes('session.id === bank.selectedSessionId && bank.detailError">读取失败'))
assert.ok(panel.includes('该分段没有提交记录，因此没有可显示的题目或答案'))
assert.ok(panel.includes("Number(session.count || 0) > 0 && ['ended', 'exported'].includes(session.state)"))
assert.ok(panel.includes('另有 {{ bank.sessionHidden }} 个更早分段仅保留在题库文件夹中'))
assert.ok(panel.includes('已清空，仅保留本地历史；不可恢复、整理或导出'))
assert.ok(panel.includes('正在整理并生成建议'))
assert.equal(panel.includes('正在整理并导出'), false)

// 分段操作区使用稳定的桌面操作列，并在窄屏切换为全宽单列，避免按钮数量挤压卡片。
assert.match(panel, /\.qb-session-row\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s+7\.5rem;/)
assert.match(panel, /\.qb-session-actions\s*\{[^}]*flex-direction:\s*column;[^}]*align-items:\s*stretch;/)
assert.match(panel, /\.qb-session-actions\s+button\s*\{[^}]*width:\s*100%;/)
assert.match(panel, /\.qb-session\s*\{[^}]*flex-direction:\s*column;/)
assert.doesNotMatch(panel, /\.qb-session-meta\s*\{[^}]*flex-shrink:\s*0;/)
assert.match(panel, /@media\s*\(max-width:\s*560px\)[\s\S]*?\.qb-session-row\s*\{[^}]*grid-template-columns:\s*1fr;/)
assert.match(panel, /@media\s*\(max-width:\s*560px\)[\s\S]*?\.qb-session-actions\s*\{[^}]*width:\s*100%;/)

const apiSource = fs.readFileSync(new URL('./src/services/api.js', import.meta.url), 'utf8')
const questionBankStore = fs.readFileSync(new URL('./src/stores/questionbank.js', import.meta.url), 'utf8')
const modal = fs.readFileSync(new URL('./src/components/QuestionBankExportModal.vue', import.meta.url), 'utf8')
assert.ok(modal.includes('模型元数据建议（待核对）'))
assert.ok(modal.includes("return bank.exportDraft.organizeReady ? '确认命名并导出' : '整理并生成建议'"))
assert.ok(modal.includes('@input="bank.setExportIssuer($event.target.value)"'))
assert.ok(modal.includes('@click="bank.restoreAutomaticExportName"'))
assert.ok(modal.includes('最终确认导出不会再次调用模型'))
assert.ok(modal.includes('每道题紧跟参考答案和解析'))
assert.ok(questionBankStore.includes('生成逐题解析整理稿 ${name}.md、原始记录.md 与 assets/'))
assert.equal(modal.includes('v-html'), false)
assert.ok(modal.includes('全部整理失败'))
assert.ok(modal.includes('只重试失败记录'))
assert.ok(modal.includes('导出已有整理结果'))
assert.ok(modal.includes('改为仅原稿'))
assert.ok(modal.includes('已整理 {{ bank.exportDraft.organizedRecords }} 条记录'))
assert.ok(modal.includes('共 {{ bank.exportDraft.structuredQuestions }} 道题'))
assert.ok(apiSource.includes("retryFailedQuestionBankRecords: (sessionID) => callApp('RetryFailedQuestionBankRecords', sessionID)"))

// Change 021 静态接线契约：按钮与快捷键共用 Store 请求函数，App 只挂一个独立确认层。
const actionBar = fs.readFileSync(new URL('./src/components/ActionBar.vue', import.meta.url), 'utf8')
const app = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
const nextExamConfirm = fs.readFileSync(new URL('./src/components/NextExamConfirm.vue', import.meta.url), 'utf8')
assert.ok(actionBar.includes('开始下一套'))
assert.ok(actionBar.includes('@click="questionBank.requestStartNextExam"'))
assert.ok(app.includes("on('request-start-next-exam', questionBank.requestStartNextExam)"))
assert.equal((app.match(/<NextExamConfirm\s*\/>/g) || []).length, 1)
assert.equal((app.match(/import NextExamConfirm from '\.\/components\/NextExamConfirm\.vue'/g) || []).length, 1)
assert.equal((nextExamConfirm.match(/<Teleport to="body">/g) || []).length, 1)
assert.ok(nextExamConfirm.includes('@click="questionBank.confirmStartNextExam"'))
assert.ok(nextExamConfirm.includes(':disabled="questionBank.nextExamSubmitting"'))
assert.ok(nextExamConfirm.includes('封存当前段'))
assert.ok(nextExamConfirm.includes('桌面历史'))
assert.ok(nextExamConfirm.includes('未发送截图'))
assert.ok(nextExamConfirm.includes('模型上下文'))
assert.ok(nextExamConfirm.includes('题库原始记录'))
assert.ok(nextExamConfirm.includes('归档截图'))
assert.ok(nextExamConfirm.includes('旧导出'))
assert.ok(nextExamConfirm.includes('同步端当前显示'))
assert.equal(nextExamConfirm.includes('useMobileStore'), false)
assert.equal(nextExamConfirm.includes('clearMobileDisplay'), false)
assert.ok(apiSource.includes("startNextExam: (confirmed) => callApp('StartNextExam', confirmed)"))

console.log('Question bank history regression tests passed')
