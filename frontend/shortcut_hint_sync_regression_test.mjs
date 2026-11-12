import assert from 'node:assert/strict'
import fs from 'node:fs'

const read = name => fs.readFileSync(new URL(name, import.meta.url), 'utf8')
const app = read('./src/App.vue')
const settings = read('./src/stores/settings.js')
const topBar = read('./src/components/TopBar.vue')
const welcome = read('./src/components/WelcomeView.vue')
const dock = read('./src/components/ScreenshotDock.vue')
const solve = read('./src/components/SolveView.vue')
const empty = read('./src/components/EmptyState.vue')
const error = read('./src/components/ErrorView.vue')

assert.ok(app.includes('vkCode: data.comboID'), 'recorded shortcut must persist the backend vkCode field')
assert.ok(app.includes('delete recordedBinding.comboID'), 'temporary comboID must not leak into saved settings')
assert.equal(app.includes('comboID: data.comboID'), false, 'event-only comboID must not be persisted')
assert.match(app, /settingsStore\.shortcuts\[action\]\s*=\s*JSON\.parse\(JSON\.stringify\(draft\)\)/, 'saved recording must update desktop runtime hint baseline')
assert.ok(app.includes('保存全部设置后将在重启后保留'))

assert.ok(settings.includes("{ action: 'toggle', label: '隐身模式'"))
assert.ok(settings.includes("{ action: 'toggle_window', label: '隐藏/显示窗口'"))
assert.ok(welcome.includes('<span>隐身模式</span>'))
assert.equal(welcome.includes('<span>隐藏窗口</span>'), false)

for (const token of [
  'settingsStore.solveShortcut',
  'settingsStore.sendShortcut',
  'settingsStore.fastSendShortcut',
  'settingsStore.cancelSolveShortcut',
  'settingsStore.toggleShortcut',
]) {
  assert.ok(topBar.includes(token), `TopBar must use ${token}`)
}
assert.match(dock, /storeToRefs\(settingsStore\)/)
for (const token of ['solveShortcut', 'sendShortcut', 'fastSendShortcut', 'cancelSolveShortcut', 'deleteShortcut']) assert.ok(dock.includes(`{{ ${token} }}`))
assert.ok(solve.includes(':shortcut="settingsStore.solveShortcut"'))
assert.ok(solve.includes(':sendShortcut="settingsStore.sendShortcut"'))
assert.ok(solve.includes(':cancel-shortcut="settingsStore.cancelSolveShortcut"'))
assert.ok(empty.includes('{{ shortcut }}') && empty.includes('{{ sendShortcut }}'))
assert.equal(error.includes('{{ solveShortcut }}'), false)
assert.ok(error.includes('重试本题'))

console.log('Shortcut hint sync regression tests passed')
