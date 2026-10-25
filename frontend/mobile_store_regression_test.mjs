import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { reactive, ref, computed } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/mobile.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '')
  .replace('export const useMobileStore', 'const useMobileStore')

const readyInfo = (url = 'link-current', overrides = {}) => ({
  available: true,
  url,
  port: 38511,
  status: 'ready',
  retryable: false,
  stableLinkEnabled: true,
  canClear: true,
  canResetLink: true,
  ...overrides,
})

function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function setup(overrides = {}) {
  setActivePinia(createPinia())
  const calls = []
  const toasts = []
  const api = {
    getMobileOutputInfo: async () => readyInfo(),
    retryMobileOutput: async () => readyInfo(),
    clearMobileDisplay: async (...args) => {
      calls.push(['clear', ...args])
      return { success: true, code: 'cleared', info: readyInfo() }
    },
    resetMobileOutputLink: async (...args) => {
      calls.push(['reset', ...args])
      return { success: true, code: 'link_reset', info: readyInfo('link-reset') }
    },
    setMobileStableLink: async (...args) => {
      calls.push(['stable', ...args])
      return {
        success: true,
        code: args[0] ? 'stable_link_enabled' : 'stable_link_disabled',
        info: readyInfo('link-mode-changed', { stableLinkEnabled: args[0] }),
      }
    },
    ...overrides,
  }
  const ui = reactive({
    showToast: (...args) => toasts.push(args),
  })
  const context = vm.createContext({
    defineStore,
    reactive,
    ref,
    computed,
    api,
    useUIStore: () => ui,
  })
  vm.runInContext(source + '\nglobalThis.useMobileStore = useMobileStore', context)
  const store = context.useMobileStore()
  return { store, second: context.useMobileStore(), calls, toasts, api }
}

// 请求确认层之前，Clear/Reset/模式切换都不得调用后端。
let h = setup()
await h.store.refresh()
assert.equal(h.store.requestClear(), true)
assert.equal(h.calls.length, 0)
assert.equal(h.store.cancelPendingAction(), true)
assert.equal(h.calls.length, 0)
assert.equal(h.store.requestReset(), true)
assert.equal(h.store.cancelPendingAction(), true)
assert.equal(h.calls.length, 0)
assert.equal(h.store.requestStableMode(false), true)
assert.equal(h.store.cancelPendingAction(), true)
assert.equal(h.calls.length, 0)

// 明确确认后只提交一次，并共享同一个 busy 状态。
const clearResult = deferred()
h = setup({
  clearMobileDisplay: (...args) => {
    h.calls.push(['clear', ...args])
    return clearResult.promise
  },
})
await h.store.refresh()
h.store.requestClear()
const clearing = h.store.confirmPendingAction()
assert.equal(h.store.busy, true)
assert.equal(h.second.busy, true)
assert.equal(await h.second.confirmPendingAction(), false, 'busy 时不得重复提交')
assert.equal(h.calls.length, 1)
assert.deepEqual(h.calls[0], ['clear', true])
clearResult.resolve({ success: true, code: 'cleared', info: readyInfo() })
assert.equal(await clearing, true)
assert.equal(h.store.busy, false)
assert.equal(h.toasts.at(-1)[0], '同步端清屏状态已提交并广播')

// mutation 会使更早的读取响应失效，旧链接不能覆盖新链接。
const staleRead = deferred()
h = setup({
  getMobileOutputInfo: () => staleRead.promise,
  resetMobileOutputLink: async (...args) => {
    h.calls.push(['reset', ...args])
    return { success: true, code: 'link_reset', info: readyInfo('link-new') }
  },
})
const reading = h.store.refresh()
assert.equal(h.store.requestReset(), false, '能力尚未读取时不能发起重置')
staleRead.resolve(readyInfo('link-old'))
await reading
assert.equal(h.store.info.url, 'link-old')

const lateRead = deferred()
h.api.getMobileOutputInfo = () => lateRead.promise
const lateReading = h.store.refresh()
assert.equal(h.store.requestReset(), false, '读取中不得建立 Reset pending')
lateRead.resolve(readyInfo('link-refreshed'))
await lateReading
assert.equal(h.store.info.url, 'link-refreshed')
assert.equal(h.store.requestReset(), true)
assert.equal(await h.store.confirmPendingAction(), true)
assert.equal(h.store.info.url, 'link-new')

// 后端拒绝或抛错时保留当前链接，不把失败伪装为成功。
h = setup({
  resetMobileOutputLink: async (...args) => {
    h.calls.push(['reset', ...args])
    return { success: false, code: 'mobile_state_persist_failed', info: readyInfo('link-should-not-apply') }
  },
})
await h.store.refresh()
const originalLink = h.store.info.url
h.store.requestReset()
assert.equal(await h.store.confirmPendingAction(), false)
assert.equal(h.store.info.url, originalLink)
assert.equal(h.toasts.at(-1)[1], 'error')

h = setup({ resetMobileOutputLink: async () => { throw Error('synthetic mutation failure') } })
await h.store.refresh()
const linkBeforeThrow = h.store.info.url
h.store.requestReset()
assert.equal(await h.store.confirmPendingAction(), false)
assert.equal(h.store.info.url, linkBeforeThrow)
assert.equal(h.toasts.at(-1)[1], 'error')

// 模式切换必须把 confirmed=true 传给专用 API，且不经过普通设置。
h = setup()
await h.store.refresh()
h.store.requestStableMode(false)
assert.equal(await h.store.confirmPendingAction(), true)
assert.deepEqual(h.calls[0], ['stable', false, true])
assert.equal(h.store.info.stableLinkEnabled, false)

// 读取中即使已有可清屏能力，也不得建立第二个 pending。
h = setup()
await h.store.refresh()
const busyRead = deferred()
h.api.getMobileOutputInfo = () => busyRead.promise
const refreshing = h.store.refresh()
assert.equal(h.store.busy, true)
assert.equal(h.store.requestClear(), false, '读取中不得建立清屏 pending')
busyRead.resolve(readyInfo())
await refreshing
assert.equal(h.store.pendingAction, null)

// 已有 pending 时若又开始读取，确认提交也必须等待读取结束。
h = setup()
await h.store.refresh()
assert.equal(h.store.requestClear(), true)
const pendingRead = deferred()
h.api.getMobileOutputInfo = () => pendingRead.promise
const pendingRefreshing = h.store.refresh()
assert.equal(await h.store.confirmPendingAction(), false, '读取中不得提交 pending action')
assert.equal(h.calls.length, 0)
assert.equal(h.store.pendingAction?.type, 'clear')
pendingRead.resolve(readyInfo())
await pendingRefreshing
assert.equal(await h.store.confirmPendingAction(), true)
assert.equal(h.calls.length, 1)

// 状态读取失败时保持初始不可清屏，不建立 pending。
h = setup({ getMobileOutputInfo: async () => { throw Error('synthetic read failure') } })
assert.equal(await h.store.refresh(), false)
assert.equal(h.store.requestClear(), false)
assert.equal(h.store.pendingAction, null)

// 快捷键事件只能刷新后发起共享确认，不得直调已确认 API。
const appSource = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
assert.ok(appSource.includes("on('request-clear-mobile-display'"), 'App 必须监听清屏快捷键事件')
assert.ok(appSource.includes('await mobile.refresh()'), '快捷键必须先刷新 Mobile 状态')
assert.ok(appSource.includes('mobile.requestClear()'), '快捷键必须复用共享确认请求')
assert.equal(appSource.includes('api.clearMobileDisplay(true)'), false, '快捷键不得绕过确认层')

// 状态面板进入 pending 时同步关闭，且 leave 动画残留位于 modal 下方。
const topBarSource = fs.readFileSync(new URL('./src/components/TopBar.vue', import.meta.url), 'utf8')
assert.ok(topBarSource.includes('() => mobile.pendingAction'), 'TopBar 必须监听共享 pending')
assert.ok(topBarSource.includes("{ flush: 'sync' }"), '面板关闭 watcher 必须同步执行')
assert.ok(topBarSource.includes('z-index: var(--z-dropdown)'), '状态面板必须使用 dropdown 层级')

// Store 不得持久化或输出 Mobile URL。
assert.equal(/localStorage|sessionStorage|console\./.test(source), false)

console.log('Mobile store regression tests passed')
