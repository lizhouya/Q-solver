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

function deferred() {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

function makeSolutionStore(overrides = {}) {
  setActivePinia(createPinia())
  const api = {
    getQuestionBankRecordImage: async () => '',
    saveImageToFile: async () => {},
    retrySolve: async () => ({ success: false, code: 'solve_no_retry_available', message: '当前没有可重试的题目' }),
    cancelSolve: async () => ({ success: false, code: 'solve_no_running_task', message: '当前没有正在进行的解题' }),
    ...overrides,
  }
  const context = vm.createContext({
    defineStore,
    ref,
    reactive,
    computed,
    api,
    document: {
      getElementById: () => null,
      createElement: () => ({ style: {}, appendChild() {}, removeChild() {} }),
      body: { appendChild() {}, removeChild() {} },
    },
    renderMarkdownWithLatex: text => text || '',
    requestAnimationFrame: fn => { fn(); return 1 },
    cancelAnimationFrame() {},
    setTimeout,
    clearTimeout,
    console,
  })
  vm.runInContext(solutionSource + '\nglobalThis.useSolutionStore = useSolutionStore', context)
  return { store: context.useSolutionStore(), api }
}

let harness = makeSolutionStore()
let store = harness.store
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'request-start', mode: 'fast', images: ['image'] })
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'solution-stream-chunk', text: 'partial' })
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'solution-cancelled', text: '模型请求已取消', canRetry: true })
assert.equal(store.currentRounds[0].status, 'cancelled')
assert.equal(store.currentRounds[0].aiResponse, 'partial')
assert.equal(store.currentRounds[0].canRetry, true)
assert.equal(store.currentRounds[0].error, null)
assert.equal(store.currentRounds[0].interruption.desc, '模型请求已取消')
assert.equal(store.currentSolveMode, 'fast')

const roundsBefore = store.history[0].rounds.length
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'request-start', mode: 'fast', images: ['image'] })
assert.equal(store.history[0].rounds.length, roundsBefore)
assert.equal(store.currentRounds[0].aiResponse, '')
assert.equal(store.currentRounds[0].thinking, '')
assert.equal(store.currentRounds[0].error, null)
assert.equal(store.currentRounds[0].interruption, null)
assert.equal(store.currentRounds[0].status, 'running')

const retryGate = deferred()
const calls = []
harness = makeSolutionStore({
  retrySolve: async (sessionId, recordId) => {
    calls.push(['retry', sessionId, recordId])
    return retryGate.promise
  },
})
store = harness.store
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'request-start', images: ['image'] })
store.handleDesktopSolveEvent({ epoch: 7, sessionId: 'session-a', recordId: 'record-a', event: 'solution-error', text: 'network', canRetry: true })
const round = store.currentRounds[0]
const firstRetry = store.retryRound(round)
const secondRetry = await store.retryRound(round)
assert.equal(secondRetry, false)
assert.deepEqual(calls, [['retry', 'session-a', 'record-a']])
assert.equal(JSON.stringify(calls).includes('image'), false)
retryGate.resolve({ success: true })
assert.equal(await firstRetry, true)

const cancelGate = deferred()
calls.length = 0
harness = makeSolutionStore({
  cancelSolve: async () => {
    calls.push(['cancel'])
    return cancelGate.promise
  },
})
store = harness.store
store.handleDesktopSolveEvent({ epoch: 8, sessionId: 'session-b', recordId: 'record-b', event: 'request-start', mode: 'normal', images: ['image'] })
const cancellingRound = store.currentRounds[0]
const firstCancel = store.cancelCurrentSolve()
assert.equal(cancellingRound.status, 'cancelling')
assert.equal(await store.cancelCurrentSolve(), false)
assert.deepEqual(calls, [['cancel']])
cancelGate.resolve({ success: true, code: 'solve_cancel_requested' })
assert.equal(await firstCancel, true)

const source = fs.readFileSync(new URL('./src/stores/solution.js', import.meta.url), 'utf8')
assert.doesNotMatch(source, /retrySolve\([^)]*(image|base64|config)/i)
const apiSource = fs.readFileSync(new URL('./src/services/api.js', import.meta.url), 'utf8')
assert.match(apiSource, /retrySolve:\s*\(sessionId, recordId\)/)
assert.match(apiSource, /cancelSolve:\s*\(\)/)
assert.match(apiSource, /triggerQuickSend:\s*\(\)/)

const appSource = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
assert.match(appSource, /vkCode:\s*data\.comboID/)
assert.match(appSource, /delete\s+recordedBinding\.comboID/)
assert.doesNotMatch(appSource, /comboID:\s*data\.comboID/)

const solveView = fs.readFileSync(new URL('./src/components/SolveView.vue', import.meta.url), 'utf8')
const controls = fs.readFileSync(new URL('./src/components/SolveControls.vue', import.meta.url), 'utf8')
const errorView = fs.readFileSync(new URL('./src/components/ErrorView.vue', import.meta.url), 'utf8')
const settings = fs.readFileSync(new URL('./src/stores/settings.js', import.meta.url), 'utf8')
const dock = fs.readFileSync(new URL('./src/components/ScreenshotDock.vue', import.meta.url), 'utf8')
assert.match(solveView, /<SolveControls/)
assert.match(solveView, /solution\.retryRound\(round\)/)
assert.match(solveView, /solution\.cancelCurrentSolve/)
assert.match(controls, /重试本题/)
assert.match(controls, /中断解题/)
assert.match(controls, /正在中断/)
assert.match(settings, /send_fast/)
assert.match(settings, /cancel_solve/)
assert.match(dock, /fastSendShortcut/)
assert.doesNotMatch(controls, /暂停后继续|恢复生成/)
assert.doesNotMatch(errorView, /按[\s\S]*截图快捷键[\s\S]*重试/)

const { parse, compileScript } = require('@vue/compiler-sfc')
const { createSSRApp } = require('vue')
const { renderToString } = require('@vue/server-renderer')
const { descriptor } = parse(controls)
const compiled = compileScript(descriptor, { id: 'solve-controls-test', inlineTemplate: true })
const code = compiled.content.replace(/import \{([^}]+)\} from ["']vue["']/g, (_, names) =>
  'const {' + names.replace(/ as /g, ': ') + '} = Vue').replace('export default', 'module.exports =')
const componentContext = { Vue: require('vue'), module: { exports: {} } }
vm.runInNewContext(code, componentContext)
const component = componentContext.module.exports
const runningHTML = await renderToString(createSSRApp(component, { running: true, mode: 'fast' }))
assert.match(runningHTML, /快速解题中/)
assert.match(runningHTML, /中断解题/)
const cancellingHTML = await renderToString(createSSRApp(component, { running: true, cancelling: true, busy: true }))
assert.match(cancellingHTML, /正在中断/)
assert.match(cancellingHTML, /disabled/)
const cancelledHTML = await renderToString(createSSRApp(component, { cancelled: true, canRetry: true }))
assert.match(cancelledHTML, /重试本题/)

console.log('Solve retry cancel fast regression tests passed')
