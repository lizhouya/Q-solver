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

function continuation(id, epoch, records) {
  return {
    epoch,
    contextQuality: 'record-reconstructed',
    contextWarning: '按题库记录重建上下文，可能与当时实际上下文不同',
    detail: {
      id,
      state: 'active',
      name: `会话 ${id}`,
      startedAt: '2026-10-23T08:00:00Z',
      records,
    },
  }
}

function record(id, answer, createdAt) {
  return {
    id,
    status: 'completed',
    model: 'fixture-model',
    scene: 'fixture-scene',
    keepContext: true,
    answer,
    createdAt,
    images: [],
  }
}

function makeStore() {
  setActivePinia(createPinia())
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
    api: {
      getQuestionBankRecordImage: async () => 'data:image/png;base64,fixture',
      saveImageToFile: async () => {},
    },
    document,
    renderMarkdownWithLatex: text => `<p>${text || ''}</p>`,
    requestAnimationFrame: callback => { callback(); return 1 },
    cancelAnimationFrame: () => {},
    setTimeout,
    clearTimeout,
    console,
  })
  vm.runInContext(solutionSource + '\nglobalThis.useSolutionStore = useSolutionStore', context)
  return context.useSolutionStore()
}

let solution = makeStore()

// 一个 Session 的每个 Record 都投影为独立条目，Session 内最新轮优先。
assert.equal(solution.activateQuestionBankSession(continuation('session-a', 7, [
  record('record-a1', '第一轮答案', '2026-10-23T08:01:00Z'),
  record('record-a2', '第二轮答案', '2026-10-23T08:02:00Z'),
])), true)
assert.deepEqual(Array.from(solution.historyEntries, entry => entry.recordId), ['record-a2', 'record-a1'])
assert.deepEqual(Array.from(solution.historyEntries, entry => entry.roundLabel), ['第 2 轮', '第 1 轮'])
assert.deepEqual(Array.from(solution.historyEntries, entry => entry.summary), ['第二轮答案...', '第一轮答案...'])
assert.equal(solution.historyEntries[0].key, 'typed:session-a:record-a2')
assert.equal(solution.historyEntries[0].time, new Date('2026-10-23T08:02:00Z').toLocaleTimeString())
assert.equal(solution.currentRounds.length, 1)
assert.equal(solution.currentRounds[0].recordId, 'record-a2')

// 点击中间轮只改变浏览目标，不改变父 Session 或后台 stream target。
const firstEntry = solution.historyEntries.find(entry => entry.recordId === 'record-a1')
assert.equal(solution.selectHistoryEntry(firstEntry), true)
assert.equal(solution.currentHistoryItem.sessionId, 'session-a')
assert.deepEqual(Array.from(solution.currentRounds, round => round.recordId), ['record-a1'])
assert.equal(solution.isHistoryEntrySelected(firstEntry), true)

// 新请求自动选中新 Record；旧轮浏览不显示新轮流状态，chunk 仍写真实目标。
assert.equal(solution.handleDesktopSolveEvent({
  epoch: 7,
  sessionId: 'session-a',
  recordId: 'record-a3',
  event: 'request-start',
  images: [],
}), true)
assert.deepEqual(Array.from(solution.historyEntries, entry => entry.recordId), ['record-a3', 'record-a2', 'record-a1'])
assert.equal(solution.currentRounds[0].recordId, 'record-a3')
assert.equal(solution.isViewingStreamTarget, true)
const stableA1Key = solution.historyEntries.find(entry => entry.recordId === 'record-a1').key
assert.equal(solution.selectHistoryEntry(solution.historyEntries.find(entry => entry.recordId === 'record-a1')), true)
assert.equal(solution.isViewingStreamTarget, false)
assert.equal(solution.handleDesktopSolveEvent({
  epoch: 7,
  sessionId: 'session-a',
  recordId: 'record-a3',
  event: 'solution-stream-chunk',
  text: '后台 A3',
}), true)
assert.equal(solution.currentRounds[0].aiResponse, '第一轮答案')
assert.equal(solution.history[0].rounds.at(-1).aiResponse, '后台 A3')
assert.equal(solution.selectHistoryEntry(solution.historyEntries.find(entry => entry.recordId === 'record-a3')), true)
assert.equal(solution.isViewingStreamTarget, true)
assert.equal(solution.currentRounds[0].aiResponse, '后台 A3')

// 新 Session 插入后 typed key 保持稳定；浏览旧 Session 时 B 的流继续写 B。
solution.activateQuestionBankSession(continuation('session-b', 8, [
  record('record-b1', 'B1 历史答案', '2026-10-23T09:01:00Z'),
]))
assert.equal(solution.historyEntries.find(entry => entry.recordId === 'record-a1').key, stableA1Key)
assert.equal(solution.handleDesktopSolveEvent({
  epoch: 8,
  sessionId: 'session-b',
  recordId: 'record-b2',
  event: 'request-start',
  images: [],
}), true)
assert.equal(solution.currentRounds[0].recordId, 'record-b2')
assert.equal(solution.selectHistoryEntry(solution.historyEntries.find(entry => entry.recordId === 'record-a2')), true)
assert.equal(solution.isViewingStreamTarget, false)
assert.equal(solution.handleDesktopSolveEvent({
  epoch: 8,
  sessionId: 'session-b',
  recordId: 'record-b2',
  event: 'solution-stream-chunk',
  text: '只写入 B2',
}), true)
assert.equal(solution.currentRounds[0].aiResponse, '第二轮答案')
assert.equal(solution.history.find(item => item.sessionId === 'session-b').rounds.at(-1).aiResponse, '只写入 B2')

// 删除其他 Session 保持所选 Record；被删 Session 的 stream target 必须清理。
assert.equal(solution.currentHistoryItem.sessionId, 'session-a')
solution.deleteHistory(0)
assert.equal(solution.currentHistoryItem.sessionId, 'session-a')
assert.equal(solution.currentRounds[0].recordId, 'record-a2')
assert.equal(solution.streamTarget, null)
assert.deepEqual(Array.from(solution.historyEntries, entry => entry.recordId), ['record-a3', 'record-a2', 'record-a1'])

// 删除当前 Session 后所有派生条目和选择都消失。
solution.deleteHistory(0)
assert.equal(solution.historyEntries.length, 0)
assert.equal(solution.currentRounds.length, 0)
assert.equal(solution.currentHistoryItem, null)
assert.equal(solution.selectedRoundTarget, null)

// legacy 流也保存独立写入目标，浏览旧轮不能重定向新轮 chunk。
solution = makeStore()
solution.setUserScreenshot('data:image/png;base64,legacy-1')
solution.handleStreamStart(false)
assert.equal(solution.historyEntries.length, 1)
assert.equal(solution.currentRounds.length, 1)
assert.equal(solution.isViewingStreamTarget, true)
solution.handleStreamChunk('旧轮答案')
solution.handleSolution('旧轮答案')
const legacyFirstKey = solution.historyEntries[0].key
solution.setUserScreenshot('data:image/png;base64,legacy-2')
solution.handleStreamStart(true)
assert.equal(solution.historyEntries.length, 2)
assert.notEqual(solution.historyEntries[0].key, legacyFirstKey)
assert.equal(solution.currentRounds[0], solution.history[0].rounds[1])
assert.equal(solution.selectHistoryEntry(solution.historyEntries.find(entry => entry.key === legacyFirstKey)), true)
assert.equal(solution.isViewingStreamTarget, false)
solution.handleStreamChunk('新轮答案')
assert.equal(solution.currentRounds[0].aiResponse, '旧轮答案')
assert.equal(solution.history[0].rounds[1].aiResponse, '新轮答案')

solution.resetLiveHistory()
assert.equal(solution.historyEntries.length, 0)
assert.equal(solution.currentRounds.length, 0)
assert.equal(solution.selectedRoundTarget, null)
assert.equal(solution.streamTarget, null)

// 静态 UI 契约：左栏按 entry 展示；右栏不再渲染多轮分隔；菜单明确会话级语义。
const solveView = fs.readFileSync(new URL('./src/components/SolveView.vue', import.meta.url), 'utf8')
assert.ok(solveView.includes('solution.historyEntries'))
assert.ok(solveView.includes('solution.selectHistoryEntry(entry)'))
assert.ok(solveView.includes('solution.isHistoryEntrySelected(entry)'))
assert.equal(solveView.includes('RoundDivider'), false)
assert.equal(solveView.includes(':roundsCount='), false)

const historyItem = fs.readFileSync(new URL('./src/components/HistoryItem.vue', import.meta.url), 'utf8')
assert.ok(historyItem.includes('roundLabel'))
assert.ok(historyItem.includes('第 N 轮') || historyItem.includes('{{ roundLabel }}'))
assert.ok(historyItem.includes('导出此会话为图片'))
assert.ok(historyItem.includes('删除此会话'))
assert.equal(historyItem.includes('roundsCount'), false)
assert.equal(historyItem.includes('rounds-badge'), false)

const app = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
assert.ok(app.includes('current.rounds.pop()'))
assert.ok(app.includes('solution.selectHistory(0)'))

console.log('History round selection regression tests passed')

// 结构化结果跟随 Record，不跟随正在浏览的旧轮。
solution = makeStore()
const archived = record('old', '旧全文', '2026-10-23T08:01:00Z')
archived.answerItems = [{ label: '1', kind: 'single_choice', value: 'A', status: 'ready' }]
archived.answerCopyText = 'A'
solution.activateQuestionBankSession(continuation('copy-session', 30, [archived]))
assert.equal(solution.currentRounds[0].answerCopyText, 'A')
solution.handleDesktopSolveEvent({ epoch: 30, sessionId: 'copy-session', recordId: 'new', event: 'request-start', images: [] })
solution.selectHistoryEntry(solution.historyEntries.find(entry => entry.recordId === 'old'))
solution.handleDesktopSolveEvent({ epoch: 30, sessionId: 'copy-session', recordId: 'new', event: 'solution', text: '新全文', answerItems: [{ label: '2', kind: 'multiple_choice', value: 'AC', status: 'ready' }], answerCopyText: 'AC' })
assert.equal(solution.currentRounds[0].answerCopyText, 'A')
solution.selectHistoryEntry(solution.historyEntries.find(entry => entry.recordId === 'new'))
assert.equal(solution.currentRounds[0].answerCopyText, 'AC')
solution.resetLiveHistory()
assert.equal(solution.historyEntries.length, 0)
