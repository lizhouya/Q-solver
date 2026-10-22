import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'

const html = fs.readFileSync(new URL('./web/index.html', import.meta.url), 'utf8')
const match = html.match(/<script>([\s\S]*?)<\/script>/)
assert.ok(match, '未找到手机页内联脚本')

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
}

const scrollCalls = []
const window = {
  innerHeight: 800,
  scrollBy(options) { scrollCalls.push(options) },
}

const context = {
  console,
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
  requestAnimationFrame(callback) { callback() },
  window,
}
window.window = window
window.document = document
vm.createContext(context)
vm.runInContext(match[1], context, { filename: 'pkg/mobile/web/index.html' })

assert.equal(StubEventSource.instances.length, 1, '应建立一个 SSE 连接')
const source = StubEventSource.instances[0]
const hints = [{ label: '向上滚动', keyName: 'Alt+PgUp' }]
const snapshot = (seq, answer, controlSeq = 0, control = '') => ({
  seq,
  ts: 1_700_000_000_000 + seq,
  status: 'done',
  stage: 'content',
  payload: { answer, model: 'test-model' },
  hints,
  controlSeq,
  control,
})

source.emit('snapshot', snapshot(1, '答案一', 5, 'down'))
assert.equal(scrollCalls.length, 0, '首帧不得重放旧控制')
assert.equal(elements.get('answer-text').innerHTMLWrites, 1, '首帧应渲染业务答案')
assert.equal(elements.get('shortcut-hints').textClears, 1, '首帧应渲染快捷键提示')

source.emit('message', snapshot(2, '答案二', 5))
const historySide = elements.get('history-list-side')
assert.equal(historySide.children.length, 2, '新业务快照应增加历史答案')
historySide.children[0].click()
assert.match(elements.get('answer-text').innerHTML, /答案一/, '测试前提：已选择历史答案')

const answerWritesBeforeControl = elements.get('answer-text').innerHTMLWrites
const historyRendersBeforeControl = historySide.textClears
source.emit('message', snapshot(2, '答案二', 6, 'down'))
assert.equal(scrollCalls.length, 1, '新 controlSeq 应只滚动一次')
assert.equal(scrollCalls[0].top, 560)
assert.equal(scrollCalls[0].behavior, 'smooth')
assert.equal(elements.get('answer-text').innerHTMLWrites, answerWritesBeforeControl, '控制帧不得重建答案 DOM')
assert.equal(historySide.textClears, historyRendersBeforeControl, '控制帧不得重建历史 DOM')
assert.match(elements.get('answer-text').innerHTML, /答案一/, '控制帧不得跳回最新答案')

source.emit('message', snapshot(2, '答案二', 6, 'down'))
assert.equal(scrollCalls.length, 1, '重复 controlSeq 不得重复滚动')

const changedHints = snapshot(2, '答案二', 6)
changedHints.hints = [{ label: '向下滚动', keyName: 'Alt+PgDn' }]
source.emit('message', changedHints)
assert.equal(elements.get('shortcut-hints').textClears, 2, 'hints 变化时应更新一次')
source.emit('message', changedHints)
assert.equal(elements.get('shortcut-hints').textClears, 2, '相同 hints 不得重复重建')
assert.match(elements.get('answer-text').innerHTML, /答案一/, '仅 hints 变化不得改变历史选择')

source.emit('message', snapshot(3, '答案三', 7, 'up'))
assert.equal(scrollCalls.length, 2, '新的向上控制应执行一次')
assert.equal(scrollCalls[1].top, -560)
assert.equal(scrollCalls[1].behavior, 'smooth')
assert.match(elements.get('answer-text').innerHTML, /答案三/, '新业务 seq 应正常更新并切回最新答案')
assert.equal(historySide.children.length, 3, '新业务 seq 应继续维护历史记录')

// 可选输入来自 Go 测试的真实 SSE 输出，而非手造的业务序号。
if (process.argv[2]) {
  const frames = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'))
  const freshContext = vm.createContext({ ...context })
  vm.runInContext(match[1], freshContext)
  const realtimeSource = StubEventSource.instances.at(-1)
  let completed = 0
  for (const frame of frames) {
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

console.log('Mobile DOM control tests passed')
