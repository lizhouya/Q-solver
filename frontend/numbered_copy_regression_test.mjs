import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { computed, ref } = require('vue')
const clipboardSource = fs.readFileSync(new URL('./src/utils/clipboard.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace(/^export /gm, '')
const utilities = vm.createContext({})
vm.runInContext(clipboardSource, utilities)
const format = utilities.formatNumberedAnswers
assert.equal(typeof format, 'function')
const items = [{label:'16', value:'C', status:'ready'}, {label:'20', value:'AB', status:'ready'}, {label:'第二部分-3', value:'42', status:'ready'}]
assert.equal(format(items, 'C\nAB\n42'), '16. C\n20. AB\n第二部分-3. 42')
assert.equal(format(items, ''), '')
assert.equal(format(items, 'C\nA\n42'), '')
assert.equal(format([{...items[0], status:'review'}], 'C'), '')
assert.equal(format([{...items[0], label:'待核对-1'}], 'C'), '')
assert.equal(format([{...items[0], label:'16\n17'}], 'C'), '')
assert.equal(format([], 'C'), '')

const vue = fs.readFileSync(new URL('./src/components/ActionBar.vue', import.meta.url), 'utf8')
assert.match(vue, /复制题号＋答案/)
const script = vue.match(/<script setup>([\s\S]*?)<\/script>/)[1].replace(/^import .*$/gm, '')
const writes = []
const props = {content:'完整答案与解析', answerItems:items, answerCopyText:'C\nAB\n42'}
const ctx = vm.createContext({
  computed, ref, watch() {}, onBeforeUnmount() {},
  defineProps: () => props,
  useQuestionBankStore: () => ({status:{}, busy:false}),
  formatNumberedAnswers: format,
  copyText: async text => { writes.push(text); return true },
  setTimeout: () => 1, clearTimeout() {},
})
vm.runInContext(script, ctx)
await ctx.handleCopyNumberedAnswers()
await ctx.handleCopyAnswers()
await ctx.handleCopy()
assert.deepEqual(writes, ['16. C\n20. AB\n第二部分-3. 42', 'C\nAB\n42', '完整答案与解析'])
console.log('Numbered answer copy regression tests passed')
