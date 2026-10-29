import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { reactive, ref, computed, createSSRApp } = require('vue')
const { defineStore, createPinia, setActivePinia } = require('pinia')
const source = fs.readFileSync(new URL('./src/stores/solution.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace('export const useSolutionStore', 'const useSolutionStore')
setActivePinia(createPinia())
const context = vm.createContext({ defineStore, ref, reactive, computed,
  api: {}, document: { getElementById: () => null },
  renderMarkdownWithLatex: text => text || '', requestAnimationFrame: () => 1,
  cancelAnimationFrame() {}, setTimeout, clearTimeout,
})
vm.runInContext(source + '\nglobalThis.store = useSolutionStore()', context)
const store = context.store
const detail = { code: 'provider_unavailable', stage: 'stream_response', httpStatus: 502,
  protocol: 'openai_chat', retryable: true, message: '模型服务暂时不可用，请稍后重试' }
const event = (recordId, type, extras = {}) => ({ epoch: 7, sessionId: 'session-a', recordId, event: type, ...extras })
store.handleDesktopSolveEvent(event('a', 'request-start'))
assert.equal(store.handleDesktopSolveEvent(event('a', 'solution-error', { text: detail.message, errorDetail: detail })), true)
assert.equal(store.currentRounds[0].error.detail.httpStatus, 502)
assert.equal(store.currentRounds[0].error.desc, detail.message)
assert.equal(store.currentRounds[0].error.detail.retryable, true)
assert.equal(store.handleDesktopSolveEvent(event('a', 'solution-error', { text: 'late', errorDetail: { ...detail, httpStatus: 503 } })), false)
assert.equal(store.currentRounds[0].error.detail.httpStatus, 502)
store.handleDesktopSolveEvent(event('b', 'request-start'))
assert.equal(store.currentRounds[0].error, null)
assert.equal(store.errorState.detail, null)
assert.equal(store.handleDesktopSolveEvent(event('a', 'solution-error', { text: 'old', errorDetail: detail })), false)
store.handleDesktopSolveEvent(event('b', 'solution-error', { text: 'legacy failure' }))
assert.equal(store.currentRounds[0].error.detail, null)
store.handleDesktopSolveEvent(event('c', 'request-start'))
store.handleDesktopSolveEvent(event('c', 'solution-error', { text: 'safe', errorDetail: {
  code: 'TEST_ONLY_PRIVATE_KEY', stage: 'TEST_ONLY_PRIVATE_TOKEN', protocol: 'TEST_ONLY_PRIVATE_URL',
  httpStatus: 999, retryable: 'TEST_ONLY_PRIVATE_BODY', message: 'TEST_ONLY_PRIVATE_PROMPT', raw: 'TEST_ONLY_PRIVATE_RESPONSE',
} }))
assert.doesNotMatch(JSON.stringify(store.currentRounds[0].error.detail), /TEST_ONLY_PRIVATE/)
assert.equal(store.currentRounds[0].error.detail.code, 'unknown')
assert.equal(store.currentRounds[0].error.detail.httpStatus, null)
store.handleStreamStart(false)
assert.equal(store.handleInlineError({ title: '请求出错', desc: detail.message, detail }), true)
assert.equal(store.currentRounds[0].error.detail.protocol, 'openai_chat')
store.handleStreamStart(true)
assert.equal(store.currentRounds[0].error, null)

const partial = store.normalizeErrorDetail({})
assert.equal(partial.code, 'unknown')
assert.equal(partial.stage, 'unknown')
assert.equal(partial.protocol, 'unknown')
assert.equal(partial.httpStatus, null)
assert.equal(partial.retryable, null)

const app = fs.readFileSync(new URL('./src/App.vue', import.meta.url), 'utf8')
assert.match(app, /on\('solution-error',\s*\(rawErrMsg,\s*errorDetail\)/)
for (const name of ['ErrorView.vue', 'SolveView.vue']) {
  const ui = fs.readFileSync(new URL('./src/components/' + name, import.meta.url), 'utf8')
  assert.match(ui, /<ErrorDiagnostics/)
}
// 编译并 SSR 渲染真实 Vue 组件，检验安全文本和缺省显示，不仅匹配源码。
const { parse, compileScript } = require('@vue/compiler-sfc')
const { renderToString } = require('@vue/server-renderer')
const componentSource = fs.readFileSync(new URL('./src/components/ErrorDiagnostics.vue', import.meta.url), 'utf8')
const { descriptor } = parse(componentSource)
const compiled = compileScript(descriptor, { id: 'diagnostics-test', inlineTemplate: true })
const code = compiled.content.replace(/import \{([^}]+)\} from ["']vue["']/g, (_, names) =>
  'const {' + names.replace(/ as /g, ': ') + '} = Vue').replace('export default', 'module.exports =')
const componentContext = { Vue: require('vue'), module: { exports: {} } }
vm.runInNewContext(code, componentContext)
const html = await renderToString(createSSRApp(componentContext.module.exports, { detail }))
for (const value of ['诊断详情', '502', 'provider_unavailable', 'stream_response', 'openai_chat']) assert.ok(html.includes(value))
const missing = await renderToString(createSSRApp(componentContext.module.exports, { detail: null }))
assert.ok(missing.includes('暂无信息'))
console.log('LLM error diagnostics regression tests passed')
