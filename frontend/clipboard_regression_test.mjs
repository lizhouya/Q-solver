import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'

const source = fs.readFileSync(new URL('./src/utils/clipboard.js', import.meta.url), 'utf8')
  .replace(/^import .*$/gm, '').replace(/^export /gm, '')
function helper({ runtime, clipboard, fallback = false } = {}) {
  const writes = []
  let removed = 0
  const context = vm.createContext({
    window: runtime ? { runtime: {} } : {},
    ClipboardSetText: async text => { writes.push(['wails', text]); return runtime(text) },
    navigator: clipboard ? { clipboard: { writeText: async text => { writes.push(['web', text]); return clipboard(text) } } } : {},
    document: {
      activeElement: { focus() {} },
      createElement: () => ({ value: '', style: {}, setAttribute() {}, select() {}, remove() { removed++ } }),
      body: { appendChild() {} },
      execCommand: () => { writes.push(['fallback']); return fallback },
    },
  })
  vm.runInContext(source + '\nglobalThis.copyText = copyText', context)
  return { copy: context.copyText, writes, removed: () => removed }
}
let h = helper({ runtime: () => true })
assert.equal(await h.copy('B\nAC'), true)
assert.deepEqual(h.writes, [['wails', 'B\nAC']])
h = helper({ runtime: () => false, clipboard: () => undefined })
assert.equal(await h.copy('text'), true)
assert.equal(h.writes.length, 2)
h = helper({ runtime: () => { throw Error('blocked') }, clipboard: () => { throw Error('denied') }, fallback: true })
assert.equal(await h.copy('text'), true)
assert.equal(h.removed(), 1)
h = helper()
assert.equal(await h.copy('text'), false)
assert.equal(h.removed(), 1)
assert.equal(await h.copy(''), false)
console.log('clipboard regression tests passed')
