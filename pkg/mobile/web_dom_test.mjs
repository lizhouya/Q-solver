import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'

const html = fs.readFileSync(new URL('./web/index.html', import.meta.url), 'utf8')
const match = html.match(/<script>([\s\S]*?)<\/script>/)
assert.ok(match, '未找到手机页内联脚本')
assert.doesNotMatch(match[1], /\b(?:localStorage|sessionStorage)\b|console\./, '手机页不得写浏览器存储或 console')

class StubElement {
  constructor(id = '') {
    this.id = id
    this.children = []
    this.listeners = new Map()
    this.style = {}
    this.hidden = false
    this.className = ''
    this.attributes = new Map()
    this.innerHTMLWrites = 0
    this.textClears = 0
    this._textContent = ''
    this._innerHTML = ''
  }

  set textContent(value) {
    this._textContent = String(value)
    if (value === '') {
      this.children = []
      this.textClears++
    }
  }

  get textContent() { return this._textContent }

  set innerHTML(value) {
    this._innerHTML = String(value)
    this.innerHTMLWrites++
  }

  get innerHTML() { return this._innerHTML }

  get childElementCount() {
    return this.children.filter((child) => child instanceof StubElement).length
  }

  appendChild(child) {
    this.children.push(child)
    return child
  }

  setAttribute(name, value) { this.attributes.set(name, String(value)) }
  addEventListener(type, listener) { this.listeners.set(type, listener) }
  click() { this.listeners.get('click')?.() }
  select() {}
}

const elements = new Map()
const document = {
  documentElement: { clientHeight: 700 },
  visibilityState: 'visible',
  getElementById(id) {
    if (!elements.has(id)) elements.set(id, new StubElement(id))
    return elements.get(id)
  },
  createElement() { return new StubElement() },
  createTextNode(text) { return { nodeType: 3, textContent: String(text) } },
  addEventListener() {},
}

class StubEventSource {
  static CLOSED = 2
  static instances = []

  constructor(url) {
    this.url = url
    this.readyState = 1
    this.listeners = new Map()
    StubEventSource.instances.push(this)
  }

  addEventListener(type, listener) { this.listeners.set(type, listener) }
  emit(type, payload) {
    this.listeners.get(type)?.({ data: JSON.stringify(payload) })
  }
  emitRaw(type, data) {
    this.listeners.get(type)?.({ data })
  }
}

const scrollCalls = []
const rafCallbacks = []
const window = {
  innerHeight: 800,
  scrollBy(options) { scrollCalls.push(options) },
}

const context = {
  Date,
  JSON,
  Math,
  Number,
  String,
  Array,
  URLSearchParams,
  document,
  location: { search: '?token=TEST' },
  EventSource: StubEventSource,
  requestAnimationFrame(callback) { rafCallbacks.push(callback) },
  window,
}
window.window = window
window.document = document
vm.createContext(context)
vm.runInContext(match[1], context, { filename: 'pkg/mobile/web/index.html' })

function flushAnimationFrames() {
  while (rafCallbacks.length > 0) rafCallbacks.shift()()
}

assert.equal(StubEventSource.instances.length, 1, '应建立一个 SSE 连接')
const source = StubEventSource.instances[0]
const hints = [{ label: '向上滚动', keyName: 'Alt+PgUp' }]
const snapshot = (sessionId, clearSeq, seq, answer, controlSeq = 0, control = '', overrides = {}) => ({
  sessionId,
  clearSeq,
  seq,
  ts: 1_700_000_000_000 + seq,
  status: 'done',
  stage: 'content',
  payload: { answer, model: 'test-model' },
  hints,
  controlSeq,
  control,
  ...overrides,
})

source.emit('snapshot', snapshot('session-a', 2, 10, '答案一', 5, 'down'))
assert.equal(scrollCalls.length, 0, '首帧不得重放旧控制')
assert.equal(rafCallbacks.length, 0, '首帧不得排队旧控制')
assert.match(elements.get('answer-text').innerHTML, /答案一/, '首帧应渲染业务答案')
assert.equal(elements.get('history-list-side').children.length, 1, '首帧应写入宽屏历史')
assert.equal(elements.get('history-list-strip').children.length, 1, '首帧应写入窄屏历史')

const answerAfterFirstFrame = elements.get('answer-text').innerHTML
const historyAfterFirstFrame = elements.get('history-list-side').textClears
const hintsAfterFirstFrame = elements.get('shortcut-hints').textClears
for (const invalid of [
  snapshot('', 2, 11, '空 session'),
  snapshot('   ', 2, 11, '空白 session'),
  snapshot('session-a', -1, 11, '负 clear'),
  snapshot('session-a', 2.5, 11, '小数 clear'),
  snapshot('session-a', '2', 11, '字符串 clear'),
  { ...snapshot('session-a', 2, 11, '缺 clear'), clearSeq: undefined },
]) {
  source.emit('message', invalid)
}
source.emitRaw('message', '{invalid json')
assert.equal(elements.get('answer-text').innerHTML, answerAfterFirstFrame, '非法协议帧必须整帧拒绝')
assert.equal(elements.get('history-list-side').textClears, historyAfterFirstFrame, '非法协议帧不得重建历史')
assert.equal(elements.get('shortcut-hints').textClears, hintsAfterFirstFrame, '非法协议帧不得更新快捷键')

source.emit('message', snapshot('session-a', 2, 11, '答案二', 6, 'down'))
const historySide = elements.get('history-list-side')
assert.equal(historySide.children.length, 2, '同一清屏代次的新业务快照应增加历史答案')
historySide.children[0].click()
assert.match(elements.get('answer-text').innerHTML, /答案一/, '测试前提：已选择历史答案')
assert.equal(rafCallbacks.length, 1, '新 controlSeq 应排队一次滚动')

const answerWritesBeforeDuplicate = elements.get('answer-text').innerHTMLWrites
const historyRendersBeforeDuplicate = historySide.textClears
const hintsRendersBeforeDuplicate = elements.get('shortcut-hints').textClears
source.emit('message', snapshot('session-a', 2, 11, '答案二', 6, 'down'))
assert.equal(rafCallbacks.length, 1, '相同 clear 下重复 controlSeq 不得重复排队')
assert.equal(elements.get('answer-text').innerHTMLWrites, answerWritesBeforeDuplicate, '相同 clear 下重复业务帧不得重建答案 DOM')
assert.equal(historySide.textClears, historyRendersBeforeDuplicate, '相同 clear 下重复业务帧不得重建历史 DOM')
assert.equal(elements.get('shortcut-hints').textClears, hintsRendersBeforeDuplicate, '相同 clear 下重复帧不得重建快捷键 DOM')
assert.match(elements.get('answer-text').innerHTML, /答案一/, '重复帧不得改变历史选择')

source.emit('message', snapshot('session-a', 3, 12, '', 0, '', {
  status: 'idle',
  stage: '',
  payload: {},
  hints: [],
}))
assert.equal(elements.get('answer-text').innerHTML, '', '更高 clearSeq 应擦除答案 DOM')
assert.equal(elements.get('error-text').innerHTML, '', '更高 clearSeq 应擦除错误 DOM')
assert.equal(elements.get('answer-card').style.display, 'none', '更高 clearSeq 应清除答案视觉状态')
assert.equal(elements.get('waiting-card').style.display, 'none', '更高 clearSeq 应清除等待视觉状态')
assert.equal(elements.get('error-card').style.display, 'none', '更高 clearSeq 应清除错误视觉状态')
assert.equal(elements.get('history-list-side').children.length, 0, '更高 clearSeq 应清空宽屏历史')
assert.equal(elements.get('history-list-strip').children.length, 0, '更高 clearSeq 应清空窄屏历史')
assert.equal(elements.get('shortcut-card').hidden, true, '更高 clearSeq 应清除快捷键视觉状态')
assert.equal(elements.get('model-label').textContent, '等待桌面连接…', '更高 clearSeq 应重置模型视觉状态')
flushAnimationFrames()
assert.equal(scrollCalls.length, 0, '清屏前已排队的 rAF 必须因 epoch 变化失效')

const writesAfterClear = elements.get('answer-text').innerHTMLWrites
const sideRendersAfterClear = historySide.textClears
source.emit('message', snapshot('session-a', 3, 12, '', 0, '', {
  status: 'idle',
  stage: '',
  payload: {},
  hints: [],
}))
assert.equal(elements.get('answer-text').innerHTMLWrites, writesAfterClear, '同 clear 重复帧必须幂等')
assert.equal(historySide.textClears, sideRendersAfterClear, '同 clear 重复帧不得再次清历史')

source.emit('message', snapshot('session-a', 2, 99, '旧 clear 污染', 99, 'down', {
  hints: [{ label: '旧提示', keyName: 'Old' }],
}))
assert.equal(elements.get('answer-text').innerHTML, '', '旧 clearSeq 必须整帧拒绝业务内容')
assert.equal(elements.get('history-list-side').children.length, 0, '旧 clearSeq 不得恢复历史')
assert.equal(rafCallbacks.length, 0, '旧 clearSeq 不得执行控制')

source.emit('message', snapshot('session-a', 3, 1, '清后新题', 1))
assert.match(elements.get('answer-text').innerHTML, /清后新题/, '清屏后低业务序号的新题应正常渲染')
assert.equal(elements.get('history-list-side').children.length, 1, '清屏后新题应建立新历史')

source.emit('message', snapshot('session-a', 3, 2, '', 2, '', {
  status: 'error',
  payload: {},
  error: '旧错误',
}))
assert.match(elements.get('error-text').innerHTML, /旧错误/, '测试前提：错误 DOM 已显示')
source.emit('snapshot', snapshot('session-a', 4, 3, '', 20, 'down', {
  status: 'idle',
  stage: '',
  payload: {},
  hints: [],
}))
assert.equal(elements.get('error-text').innerHTML, '', 'snapshot 的更高 clearSeq 应擦除错误 DOM')
assert.equal(rafCallbacks.length, 0, 'snapshot 首帧只同步 control 基线，不得滚动')

source.emit('message', snapshot('session-a', 4, 1, '再次清后新题', 21))
assert.match(elements.get('answer-text').innerHTML, /再次清后新题/, '更高 clear 后应重置业务去重命名空间')
source.emit('snapshot', snapshot('session-b', 0, 1, '新进程答案', 50, 'down'))
assert.match(elements.get('answer-text').innerHTML, /新进程答案/, '新 session 的低序号业务应被接受')
assert.equal(elements.get('history-list-side').children.length, 1, '新 session 应先清除旧会话历史')
assert.equal(rafCallbacks.length, 0, '新 session 的 snapshot 只同步控制基线')

source.emit('message', snapshot('session-b', 0, 1, '新进程答案', 51, 'up'))
assert.equal(rafCallbacks.length, 1, '新 session 的后续实时控制应使用新去重命名空间')

source.emit('message', snapshot('session-a', 99, 999, '已淘汰会话污染', 999, 'down'))
assert.match(elements.get('answer-text').innerHTML, /新进程答案/, '切到新 session 后不得被已淘汰会话切回')
assert.equal(rafCallbacks.length, 1, '已淘汰会话不得排入滚动控制')
flushAnimationFrames()
assert.equal(scrollCalls.length, 1, '有效实时控制应执行一次')
assert.equal(scrollCalls[0].top, -560)
assert.equal(scrollCalls[0].behavior, 'smooth')

const safeMarkdown = context.renderMarkdown('[安全](https://example.com?a=1&b=2) [邮件](mailto:test@example.com)')
assert.match(safeMarkdown, /rel="noopener noreferrer"/, '安全外链必须同时声明 noopener 与 noreferrer')
assert.match(safeMarkdown, /href="https:\/\/example\.com/, 'HTTPS 外链应保留')
assert.match(safeMarkdown, /href="mailto:test@example\.com"/, 'mailto 外链应保留')
const unsafeMarkdown = context.renderMarkdown('[脚本](javascript:alert(1)) [文件](file:///tmp/a)')
assert.doesNotMatch(unsafeMarkdown, /href=/, '不安全 scheme 不得生成链接')
assert.doesNotMatch(unsafeMarkdown, /javascript:/i, '不安全 URL 不得进入渲染结果')

// 纯答案复制、历史目标保持及错误反馈。
assert.match(html, /答题端快捷键/)
const copiedAnswers = []
context.navigator = { clipboard: { writeText: async text => { copiedAnswers.push(text) } } }
const answerPayload = (value) => ({ answer: `全文 ${value}`, answerItems: [{label:'1',value,status:'ready'}], answerCopyText:value })
source.emit('snapshot', snapshot('copy-session', 0, 1, '', 0, '', { payload: answerPayload('B') }))
assert.equal(elements.get('copy-answers').disabled, false)
await context.copyDisplayedAnswers()
assert.deepEqual(copiedAnswers, ['B'])
source.emit('message', snapshot('copy-session', 0, 2, '', 0, '', { payload: answerPayload('AC') }))
context.selectHistory(1)
source.emit('message', snapshot('copy-session', 0, 3, '', 0, '', { payload: answerPayload('D') }))
await context.copyDisplayedAnswers()
assert.deepEqual(copiedAnswers, ['B', 'B'])
assert.match(elements.get('answer-text').innerHTML, /全文 B/)
context.selectHistory(3)
await context.copyDisplayedAnswers()
assert.equal(copiedAnswers.at(-1), 'D')
source.emit('message', snapshot('copy-session', 0, 4, '', 0, '', {payload:{answer:'待核对',answerItems:[{status:'review'}]}}))
assert.equal(elements.get('copy-answers').disabled, true)
const count = copiedAnswers.length
await context.copyDisplayedAnswers()
assert.equal(copiedAnswers.length, count)
source.emit('message', snapshot('copy-session', 0, 5, '', 0, '', {payload:answerPayload('E')}))
context.navigator.clipboard.writeText = async () => { throw Error('permission denied') }
document.body = { appendChild() {} }
document.execCommand = () => false
StubElement.prototype.remove = function () {}
assert.equal(await context.copyDisplayedAnswers(), false)
assert.match(elements.get('copy-feedback').textContent, /失败/)
document.execCommand = () => true
assert.equal(await context.copyDisplayedAnswers(), true)
assert.match(elements.get('copy-feedback').textContent, /已复制/)
source.emit('snapshot', snapshot('copy-session', 1, 6, '', 0, '', {status:'idle',payload:{}}))
assert.equal(elements.get('copy-answers').hidden, true)
console.log('Mobile answer copy regression tests passed')

// 带题号复制保留真实编号；后台新轮不替换所选旧轮。
assert.ok(html.includes('复制题号＋答案'), 'missing numbered-copy button')
context.navigator.clipboard.writeText = async text => { copiedAnswers.push(text) }
const numberedPayload = {
  answer:'完整回答',
  answerItems:[{label:'16',value:'C',status:'ready'},{label:'20',value:'AB',status:'ready'},{label:'第二部分-3',value:'42',status:'ready'}],
  answerCopyText:`C
AB
42`,
}
source.emit('snapshot', snapshot('numbered-session', 0, 1, '', 0, '', {payload:numberedPayload}))
assert.equal(elements.get('copy-numbered-answers').disabled, false)
await context.copyDisplayedAnswers(true)
assert.equal(copiedAnswers.at(-1), `16. C
20. AB
第二部分-3. 42`)
await context.copyDisplayedAnswers()
assert.equal(copiedAnswers.at(-1), `C
AB
42`)
source.emit('message', snapshot('numbered-session', 0, 2, '', 0, '', {payload:answerPayload('B')}))
context.selectHistory(1)
source.emit('message', snapshot('numbered-session', 0, 3, '', 0, '', {payload:answerPayload('D')}))
await context.copyDisplayedAnswers(true)
assert.equal(copiedAnswers.at(-1), `16. C
20. AB
第二部分-3. 42`)
context.selectHistory(3)
source.emit('message', snapshot('numbered-session', 0, 4, '', 0, '', {payload:{...numberedPayload,answerCopyText:''}}))
assert.equal(elements.get('copy-numbered-answers').disabled, true)
source.emit('snapshot', snapshot('numbered-session', 1, 5, '', 0, '', {status:'idle',payload:{}}))
assert.equal(elements.get('copy-numbered-answers').hidden, true)

// 可选输入来自 Go 测试的真实 SSE 输出，而非手造的业务序号。
if (process.argv[2]) {
  elements.clear()
  rafCallbacks.length = 0
  scrollCalls.length = 0
  const freshContext = vm.createContext({ ...context })
  vm.runInContext(match[1], freshContext, { filename: 'pkg/mobile/web/index.html' })
  const realtimeSource = StubEventSource.instances.at(-1)
  let completed = 0
  for (const frame of JSON.parse(fs.readFileSync(process.argv[2], 'utf8'))) {
    realtimeSource.emit(frame.event, frame.snapshot)
    const snap = frame.snapshot
    if (snap.status === 'done' || (snap.status === 'streaming' && snap.stage === 'content')) {
      assert.ok(elements.get('answer-text').innerHTML.includes(snap.payload.answer),
        `真实 SSE ${snap.status} 帧 seq=${snap.seq} 未渲染，不能依赖刷新`)
    }
    if (snap.status === 'done') completed++
  }
  assert.equal(completed, 2, '同一连接应完成两题')
  assert.equal(elements.get('history-list-side').children.length, 2, '两题均应实时进入历史')
}
