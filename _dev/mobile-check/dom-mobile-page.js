// Mobile Web 页面 DOM 行为回归验证（Change 002：remote-output-ui）
//
// 背景：001 曾出现「Go 全绿但手机页 JS 空白」缺陷（classic <script> 顶层 return 是
// parse 期错误；运行时 setBadge 越界）。002 新增 History/去重/点击切换等行为。
//
// 盲区说明：本脚本用 new Function('console', inlineJs) 执行内联 JS——会把脚本体当成
// 函数体包裹，所以「classic <script> 顶层 return」会被放行；该语法盲区由
// _dev/mobile-check/check-raw-syntax.js（vm.Script 真实 Script 目标）兜底。
// 本脚本只负责「运行时行为」断言。
//
// 用法：node _dev/mobile-check/dom-mobile-page.js
// 退出码：0 = 全通过；非 0 = 有断言失败 / 运行时错误。

'use strict';

const fs = require('fs');
const path = require('path');

const htmlPath = path.resolve(__dirname, '../../pkg/mobile/web/index.html');
const html = fs.readFileSync(htmlPath, 'utf8');
const match = html.match(/<script>([\s\S]*?)<\/script>/);
if (!match) {
  console.error('FAIL：未找到 <script> 块');
  process.exit(1);
}
const inlineJs = match[1];

// ------------------------------------------------------------------
// 极简 DOM shim（双向：appendChild 累积 children；textContent/innerHTML 可直接读写）
// ------------------------------------------------------------------
function makeEl(tag) {
  const styleBox = {};
  return {
    tagName: String(tag).toUpperCase(),
    _className: '',
    _style: styleBox,
    children: [],
    _text: '',
    innerHTML: '',
    _attr: {},
    _handlers: {},
    style: {
      get display() { return styleBox.display; },
      set display(v) { styleBox.display = v; },
    },
    setAttribute(k, v) { this._attr[k] = String(v); },
    getAttribute(k) { return this._attr[k] == null ? null : this._attr[k]; },
    appendChild(c) { this.children.push(c); return c; },
    removeChild() {},
    addEventListener(name, fn) { this._handlers[name] = this._handlers[name] || []; this._handlers[name].push(fn); },
    removable() {},
    click() { (this._handlers.click || []).forEach((fn) => fn({})); },
    querySelectorAll(sel) {
      if (!sel) return [];
      const wantClass = sel.replace(/^\./, '');
      return this.children.filter((c) => String(c.className || '').split(' ').indexOf(wantClass) >= 0);
    },
    querySelector(sel) { return this.querySelectorAll(sel)[0] || null; },
    set textContent(v) {
      this._text = String(v == null ? '' : v);
      if (this._text === '') this.children = []; // 清空列表
    },
    get textContent() {
      // 读：若显式 set 过文本则返回；否则聚合子节点文本（含 createTextNode 文本节点）
      if (this._text) return this._text;
      return (this.children || []).map((c) => (c.nodeType === 3 ? c.text : c.textContent)).join('');
    },
    set className(v) { this._className = v; },
    get className() { return this._className || ''; },
    get id() { return this._attr.id || ''; },
  };
}

// 预建页面所需元素（含 History 两容器）
const ids = ['status-badge', 'conn-line', 'model-label', 'waiting-card', 'waiting-text',
  'answer-card', 'answer-text', 'error-card', 'error-text',
  'history-list-side', 'history-list-strip'];
const els = {};
ids.forEach((id) => { els[id] = makeEl('div'); });

global.document = {
  getElementById: (id) => els[id] || makeEl('div'),
  createElement: (tag) => makeEl(tag),
  createTextNode: (t) => ({ nodeType: 3, text: String(t) }),
  querySelector: (s) => (s === '.subtitle' ? els['model-label'] : makeEl('div')),
  addEventListener: () => {},
  visibilityState: 'visible',
};
global.location = { search: '?token=abc123' };
global.URLSearchParams = class {
  constructor(q) {
    this._map = new Map();
    (q || '').replace(/^\?/, '').split('&').filter(Boolean).forEach((p) => {
      const i = p.indexOf('=');
      this._map.set(p.slice(0, i), decodeURIComponent(p.slice(i + 1)));
    });
  }
  get(k) { return this._map.get(k) || null; }
};
// EventSource shim：实例暴露给 global.__esEmitter 供手动喂帧；含 onopen/snapshot/onerror 事件
global.EventSource = class {
  constructor(url) { this.url = url; this._cb = {}; global.__es = this; }
  addEventListener(name, cb) { this._cb[name] = cb; }
  get readyState() { return 1; }
  static get CLOSED() { return 2; }
};

// ---- 断言工具 ----
let passCount = 0, failCount = 0;
function ok(cond, msg) {
  if (cond) { passCount++; console.log('  ✓ ' + msg); }
  else { failCount++; console.error('  ✗ ' + msg); }
}
function snapshot(obj) { return JSON.stringify(obj); }
function emit(snap) {
  const es = global.__es;
  const cb = es && es._cb.snapshot;
  if (!cb) throw new Error('EventSource 未建连或 snapshot 回调缺失');
  cb({ data: snapshot(snap) });
}
function answerHtml() { return els['answer-text'].innerHTML || ''; }
function answerShown() { return els['answer-card'].style.display === 'block'; }
function waitingShown() { return els['waiting-card'].style.display === 'block'; }
function badgeText() {
  return els['status-badge'].textContent || '';
}
function histCards() {
  const side = els['history-list-side'];
  return side.querySelectorAll('.hist-card') || [];
}
function cardActive(seq) {
  const list = els['history-list-side'].children || [];
  const c = list.find((x) => String(x.getAttribute('data-seq')) === String(seq));
  return c && c.className && c.className.split(' ').indexOf('active') >= 0;
}

// ---- 执行内联 JS ----
const fn = new Function('console', inlineJs);
let runError = null;
try { fn.call(global, console); } catch (e) { runError = e; }

// ---- 行为断言序列（同步驱动；渲染函数均为同步调用） ----
setTimeout(() => {
  if (runError) {
    console.error(`✗ 页面 JS 运行时错误：${runError.message}`);
    process.exit(1);
  }
  try {
    // 1. idle → 待命；Answer 不显示
    emit({ seq: 1, status: 'idle', ts: 1000, stage: 'content', payload: { model: 'test-model', answer: '' } });
    ok(badgeText().indexOf('待命') >= 0, '1. idle → 业务状态「待命」');

    // 2. streaming 且 stage=thinking（携带 analysis）→ 业务状态=处理中，且不显示 analysis 正文
    emit({
      seq: 2, status: 'streaming', stage: 'thinking',
      payload: { model: 'test-model', analysis: '这是不该显示的思考过程', answer: '' },
    });
    ok(badgeText().indexOf('处理中') >= 0, '2a. streaming-thinking → 业务状态=处理中');
    ok(answerHtml().indexOf('思考') < 0 && waitingShown(), '2b. streaming-thinking → 不显示分析正文（仍等待）');

    // 3. streaming content → Answer 实时更新
    emit({
      seq: 3, status: 'streaming', stage: 'content',
      payload: { model: 'test-model', answer: '## 第一段\n\n计算 2+2=4。' },
    });
    ok(answerHtml().indexOf('2+2=4') >= 0 && answerShown(), '3. streaming-content → Answer 即时渲染');

    // 4. done → Answer 显示 + 入 History（1 张）
    emit({
      seq: 10, status: 'done', stage: 'content', ts: Date.now(),
      payload: { model: 'test-model', answer: '## 答案一\n解析：A。' },
    });
    ok(answerHtml().indexOf('答案一') >= 0, '4a. done → Answer 显示');
    ok(histCards().length === 1, '4b. done → History 已有 1 卡');

    // 5. 同 seq 重复 done → 不重复
    emit({
      seq: 10, status: 'done', stage: 'content', ts: Date.now(),
      payload: { model: 'test-model', answer: '## 答案一\n解析：A。' },
    });
    ok(histCards().length === 1, '5. 同 seq 重复 done → History 仍 1 卡（去重）');

    // 6. 新 done（seq=11）→ 追加第 2 张
    emit({
      seq: 11, status: 'done', stage: 'content', ts: Date.now(),
      payload: { model: 'test-model', answer: '## 答案二\n解析：B。' },
    });
    ok(histCards().length === 2, '6. 新 done → History 追加为 2 卡');
    ok(cardActive(11), '   + 最新 seq=11 选中高亮');

    // 7. 点击旧卡片（seq=10）→ 右侧切换为该历史 Answer，且高亮切换
    const list = els['history-list-side'];
    const oldCard = list.children.find((c) => String(c.getAttribute('data-seq')) === String(10));
    ok(!!oldCard, '7a. 找到旧卡片 seq=10');
    oldCard.click();
    ok(answerHtml().indexOf('答案一') >= 0, '7b. 点击历史 → Answer 切换为历史内容（答案一）');
    ok(cardActive(10), '7c. 点击历史 → 选中高亮切到 seq=10');

    // 8. 新的 done（seq=12）→ 最新卡片加入且自动选中最新
    emit({
      seq: 12, status: 'done', stage: 'content', ts: Date.now(),
      payload: { model: 'test-model', answer: '## 答案三\n解析：C。' },
    });
    ok(histCards().length === 3, '8a. 新 done → History 追加为 3 卡');
    ok(answerHtml().indexOf('答案三') >= 0, '8b. 最新答案显示');

    // 汇总
    console.log(`\n✓ DOM 行为断言通过（${passCount} 全过${failCount ? '，失败 ' + failCount : ''}）`);
    process.exit(failCount === 0 ? 0 : 1);
  } catch (e) {
    console.error('✗ 断言异常：' + e.message);
    console.error(e.stack);
    process.exit(1);
  }
}, 10);