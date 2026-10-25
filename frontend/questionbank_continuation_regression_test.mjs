import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { reactive, ref, computed } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')

const solutionSource = fs.readFileSync(new URL('./src/stores/solution.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '')
  .replace('export const useSolutionStore', 'const useSolutionStore')
const questionBankSource = fs.readFileSync(new URL('./src/stores/questionbank.js', import.meta.url), 'utf8')
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

function continuation(overrides = {}) {
  return {
    epoch: 7,
    contextQuality: 'record-reconstructed',
    contextWarning: '按题库记录重建上下文，可能与当时实际上下文不同',
    detail: {
      id: 'session-a',
      state: 'active',
      name: '单位A_行测',
      company: '单位A',
      issuer: '机构B',
      type: '行测',
      startedAt: '2026-10-23T08:00:00Z',
      records: [
        {
          id: 'record-a1',
          status: 'completed',
          model: 'old-model',
          scene: 'old-scene',
          keepContext: true,
          answer: '第一题完整答案',
          images: [
            { index: 0, mime: 'image/png' },
            { index: 1, mime: 'image/jpeg' },
            { index: 2, mime: 'image/png' },
          ],
        },
        {
          id: 'record-a2',
          status: 'failed',
          model: 'old-model-2',
          scene: 'old-scene-2',
          keepContext: false,
          answer: '失败记录仍保留原答案',
          images: [{ index: 0, mime: 'image/png' }],
        },
      ],
    },
    ...overrides,
  }
}

function makeSolutionStore(apiOverrides = {}) {
  setActivePinia(createPinia())
  const api = {
    getQuestionBankRecordImage: async () => 'data:image/png;base64,fixture',
    saveImageToFile: async () => {},
    ...apiOverrides,
  }
  const document = {
    getElementById: () => null,
    createElement: () => ({ style: {}, appendChild() {}, removeChild() {} }),
    body: { appendChild() {}, removeChild() {} },
  }
  const context = vm.createContext({
    defineStore,
    ref,
    reactive,
    computed,
    api,
    document,
    renderMarkdownWithLatex: text => `<p>${text || ''}</p>`,
    requestAnimationFrame: callback => { callback(); return 1 },
    cancelAnimationFrame: () => {},
    setTimeout,
    clearTimeout,
    console,
  })
  vm.runInContext(solutionSource + '\nglobalThis.useSolutionStore = useSolutionStore', context)
  return { store: context.useSolutionStore(), api }
}

// 续接 DTO 必须完整映射为主界面会话，保留 Record 状态、旧模型/场景、完整答案和 1～3 张图片描述符。
let solutionHarness = makeSolutionStore()
let solution = solutionHarness.store
assert.equal(solution.activateQuestionBankSession(continuation()), true)
assert.equal(solution.history.length, 1)
assert.equal(solution.history[0].id, 'session:session-a')
assert.equal(solution.history[0].sessionId, 'session-a')
assert.equal(solution.history[0].epoch, 7)
assert.equal(solution.history[0].isWritable, true)
assert.equal(solution.history[0].contextQuality, 'record-reconstructed')
assert.equal(solution.history[0].contextWarning, '按题库记录重建上下文，可能与当时实际上下文不同')
assert.equal(solution.history[0].rounds.length, 2)
assert.deepEqual(
  solution.history[0].rounds.map(round => [round.recordId, round.status, round.model, round.scene, round.keepContext, round.aiResponse]),
  [
    ['record-a1', 'completed', 'old-model', 'old-scene', true, '第一题完整答案'],
    ['record-a2', 'failed', 'old-model-2', 'old-scene-2', false, '失败记录仍保留原答案'],
  ],
)
assert.equal(solution.history[0].rounds[0].images.length, 3)
assert.deepEqual(solution.history[0].rounds[0].images.map(image => [image.index, image.mime]), [
  [0, 'image/png'],
  [1, 'image/jpeg'],
  [2, 'image/png'],
])

// request-start 创建稳定目标；后续事件必须同时匹配 epoch、sessionId 和 recordId。
solution.errorState.show = true
assert.equal(solution.handleDesktopSolveEvent({
  epoch: 7,
  sessionId: 'session-a',
  recordId: 'record-a3',
  event: 'request-start',
  images: ['data:image/png;base64,new'],
}), true)
assert.equal(solution.errorState.show, false)
assert.equal(solution.handleDesktopSolveEvent({ epoch: 6, sessionId: 'session-a', recordId: 'record-a3', event: 'solution-stream-chunk', text: '旧 epoch' }), false)
assert.equal(solution.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-x', recordId: 'record-a3', event: 'solution-stream-chunk', text: '错误 session' }), false)
assert.equal(solution.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-x', event: 'solution-stream-chunk', text: '错误 record' }), false)
assert.equal(solution.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a3', event: 'solution-stream-chunk', text: '正确内容' }), true)
assert.equal(solution.history[0].rounds.at(-1).aiResponse, '正确内容')

// 查看只读会话时，后台流只更新真正的 writable 会话，且全局流展示不会覆盖只读项。
solution.activateQuestionBankSession(continuation({
  epoch: 8,
  detail: { ...continuation().detail, id: 'session-b', name: '会话B', records: [{
    id: 'record-b1', status: 'completed', answer: 'B 的历史答案', images: [],
  }] },
}))
assert.equal(solution.handleDesktopSolveEvent({ epoch: 8, sessionId: 'session-b', recordId: 'record-b2', event: 'request-start', images: [] }), true)
solution.selectHistory(1)
assert.equal(solution.currentHistoryItem.sessionId, 'session-a')
assert.equal(solution.isViewingStreamTarget, false)
assert.equal(solution.handleDesktopSolveEvent({ epoch: 8, sessionId: 'session-b', recordId: 'record-b2', event: 'solution-stream-chunk', text: '只写入 B' }), true)
assert.equal(solution.currentHistoryItem.sessionId, 'session-a')
assert.equal(solution.history.find(item => item.sessionId === 'session-b').rounds.at(-1).aiResponse, '只写入 B')
assert.equal(solution.history.find(item => item.sessionId === 'session-a').rounds.some(round => round.aiResponse === '只写入 B'), false)

// A→B 后，A 的迟到 chunk/error/solution 都必须被拒绝。
for (const event of ['solution-stream-chunk', 'solution-error', 'solution']) {
  assert.equal(solution.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a3', event, text: 'A 的迟到事件' }), false)
}
assert.equal(solution.history.find(item => item.sessionId === 'session-b').rounds.at(-1).aiResponse, '只写入 B')

// 图片按需读取；会话卡片被移除后，迟到响应不得回写已失效图片。
const lateImage = deferred()
solutionHarness = makeSolutionStore({ getQuestionBankRecordImage: async () => lateImage.promise })
solution = solutionHarness.store
solution.activateQuestionBankSession(continuation())
const imageRound = solution.history[0].rounds[0]
const image = imageRound.images[0]
const loadingImage = solution.loadRoundImage(imageRound, image)
assert.equal(image.loading, true)
solution.deleteHistory(0)
lateImage.resolve('data:image/png;base64,late')
assert.equal(await loadingImage, false)
assert.equal(image.data, '')
assert.equal(image.loading, false)

// 同一 Record 的多张图片互不取消，第一张图片同步给兼容导出字段。
const image0 = deferred()
const image1 = deferred()
let imageReadCalls = 0
solutionHarness = makeSolutionStore({
  getQuestionBankRecordImage: async (_sessionId, _recordId, index) => {
    imageReadCalls += 1
    return index === 0 ? image0.promise : image1.promise
  },
})
solution = solutionHarness.store
solution.activateQuestionBankSession(continuation())
const multiImageRound = solution.history[0].rounds[0]
const loading0 = solution.loadRoundImage(multiImageRound, multiImageRound.images[0])
const loading1 = solution.loadRoundImage(multiImageRound, multiImageRound.images[1])
image1.resolve('data:image/jpeg;base64,one')
image0.resolve('data:image/png;base64,zero')
assert.equal(await loading0, true)
assert.equal(await loading1, true)
assert.ok(multiImageRound.images[0].data.endsWith('zero'))
assert.ok(multiImageRound.images[1].data.endsWith('one'))
assert.equal(multiImageRound.userScreenshot, multiImageRound.images[0].data)
assert.equal(imageReadCalls, 2)
const cachedFirstImage = multiImageRound.images[0].data
assert.equal(await solution.loadRoundImage(multiImageRound, multiImageRound.images[0]), false)
assert.equal(imageReadCalls, 2)
assert.equal(multiImageRound.images[0].data, cachedFirstImage)
assert.equal(multiImageRound.userScreenshot, cachedFirstImage)

function makeQuestionBankStore(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const toasts = []
  const ui = reactive({ hasStarted: false, mainVisible: false, showSettings: false, showToast: (...args) => toasts.push(args) })
  const solution = {
    activateQuestionBankSession: value => { calls.push(['activate', value.detail.id]); return true },
    resetLiveHistory: () => {},
  }
  let status = {
    archiveDir: 'C:/bank', root: 'C:/bank', session: { id: 'current', state: 'active', count: 1 },
    unsaved: false, busy: false, organizing: false, operation: '', progress: {},
  }
  let sessions = {
    sessions: [
      { id: 'ended', state: 'ended', count: 1 },
      { id: 'exported', state: 'exported', count: 1 },
      { id: 'paused', state: 'paused', count: 1 },
      { id: 'discarded', state: 'discarded', count: 1 },
    ],
    total: 4, hidden: 0, limit: 20,
  }
  const api = {
    getQuestionBankStatus: async () => { calls.push(['refresh']); return status },
    listRecentQuestionBankSessions: async () => sessions,
    getQuestionBankPreferences: async () => ({ version: 2, archiveDir: 'C:/bank', autoOrganize: false }),
    continueQuestionBankSession: async id => { calls.push(['continue', id]); return continuation({ detail: { ...continuation().detail, id } }) },
    removeFocus: async () => { calls.push(['removeFocus']) },
    restoreFocus: async () => {},
    ...overrides,
  }
  const context = vm.createContext({ defineStore, ref, reactive, computed, api, useUIStore: () => ui, useSolutionStore: () => solution })
  vm.runInContext(questionBankSource + '\nglobalThis.useQuestionBankStore = useQuestionBankStore', context)
  return {
    store: context.useQuestionBankStore(), calls, toasts, ui,
    setStatus: value => { status = value },
    setSessions: value => { sessions = value },
  }
}

// 只有 ended/exported/paused 可继续；成功路径必须先激活 solution，再刷新并关闭面板。
let bankHarness = makeQuestionBankStore()
let bank = bankHarness.store
bank.visible = true
await bank.loadSessions()
assert.equal(await bank.continueSession('discarded'), false)
assert.equal(await bank.continueSession('ended'), true)
assert.equal(bank.visible, false)
assert.equal(bankHarness.ui.hasStarted, true)
assert.equal(bankHarness.ui.mainVisible, true)
assert.deepEqual(bankHarness.calls.slice(0, 3), [['continue', 'ended'], ['activate', 'ended'], ['refresh']])
assert.ok(bankHarness.toasts.some(toast => toast[0].includes('已切换到历史分段')))

// API 失败必须保持题库面板与主界面原状，不得激活 solution。
bankHarness = makeQuestionBankStore({ continueQuestionBankSession: async () => { throw new Error('synthetic continuation failure') } })
bank = bankHarness.store
bank.visible = true
bankHarness.ui.hasStarted = true
bankHarness.ui.mainVisible = true
await bank.loadSessions()
assert.equal(await bank.continueSession('paused'), false)
assert.equal(bank.visible, true)
assert.equal(bankHarness.ui.hasStarted, true)
assert.equal(bankHarness.ui.mainVisible, true)
assert.equal(bankHarness.calls.some(call => call[0] === 'activate'), false)
assert.ok(bankHarness.toasts.some(toast => toast[0].includes('synthetic continuation failure')))

// 静态 UI 约束：继续按钮状态矩阵、上下文警告、按 Record 展示截图、稳定 key 与不可删除规则均存在。
const panel = fs.readFileSync(new URL('./src/components/QuestionBankPanel.vue', import.meta.url), 'utf8')
assert.ok(panel.includes("v-if=\"['ended', 'exported', 'paused'].includes(session.state)\""))
assert.ok(panel.includes('@click.stop="bank.continueSession(session.id)"'))
assert.ok(panel.includes("bank.continueSessionWorking === session.id ? '切换中…' : '继续此分段'"))
assert.equal(panel.includes("['ended', 'exported', 'paused', 'discarded'].includes(session.state)"), false)

const solveView = fs.readFileSync(new URL('./src/components/SolveView.vue', import.meta.url), 'utf8')
assert.ok(solveView.includes('solution.currentHistoryItem?.contextWarning'))
assert.ok(solveView.includes('<RoundScreenshots'))
assert.ok(solveView.includes(':round="round"'))
assert.ok(solveView.includes(':scope-key="roundScopeKey(round, idx)"'))
assert.ok(solveView.includes('@load="solution.loadRoundImage"'))
assert.ok(solveView.includes(':key="round.recordId || round.localId || idx"'))
assert.ok(solveView.includes('solution.historyEntries'))
assert.ok(solveView.includes('solution.isHistoryEntrySelected(entry)'))
assert.ok(solveView.includes('@select="solution.selectHistoryEntry(entry)"'))
assert.ok(solveView.includes('solution.isViewingStreamTarget && solution.isLoading'))
assert.ok(solveView.includes(':deletable="!(entry.parent.isWritable && (solution.isLoading || solution.isAppending || solution.isThinking))"'))

const historyItem = fs.readFileSync(new URL('./src/components/HistoryItem.vue', import.meta.url), 'utf8')
assert.ok(historyItem.includes("isWritable ? '后台当前' : '只读查看'"))
assert.ok(historyItem.includes('v-if="deletable" class="menu-item danger"'))

const screenshots = fs.readFileSync(new URL('./src/components/RoundScreenshots.vue', import.meta.url), 'utf8')
assert.ok(screenshots.includes('v-for="image in round.images"'))
assert.ok(screenshots.includes('正在读取截图…'))
assert.ok(screenshots.includes('<figcaption v-if="image.error">'))

const app = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
assert.ok(app.includes("on('desktop-solve-event', (event) =>"))
assert.ok(app.includes('solution.handleDesktopSolveEvent(event)'))
assert.ok(app.includes("on('resume-warning-required', () =>"))
assert.ok(app.includes('await api.confirmSolveWithoutResume()'))

const dock = fs.readFileSync(new URL('./src/components/ScreenshotDock.vue', import.meta.url), 'utf8')
assert.ok(dock.includes("on('desktop-solve-event', event =>"))
assert.ok(dock.includes("event?.event === 'request-start'"))

const mobile = fs.readFileSync(new URL('../app/mobile.go', import.meta.url), 'utf8')
assert.ok(mobile.includes('bridge.PublishIfCurrent(expectedDisplay, event, data...)'))
assert.ok(mobile.includes('Epoch:     target.epoch'))
assert.ok(mobile.includes('runtimeEmit(a.ctx, "desktop-solve-event", envelope)'))

console.log('Question bank continuation regression tests passed')
