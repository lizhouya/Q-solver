import assert from 'node:assert/strict'
import fs from 'node:fs'

const dock = fs.readFileSync(new URL('./src/components/ScreenshotDock.vue', import.meta.url), 'utf8')

assert.match(dock, /import\s*\{\s*storeToRefs\s*\}\s*from\s*['"]pinia['"]/)
assert.match(
  dock,
  /const\s*\{\s*solveShortcut\s*,\s*sendShortcut\s*,\s*fastSendShortcut\s*,\s*cancelSolveShortcut\s*,\s*deleteShortcut\s*\}\s*=\s*storeToRefs\(settingsStore\)/,
)
assert.match(dock, /\{\{\s*screenshots\.length\s*\}\}\/3/)
assert.doesNotMatch(dock, /<div\s+v-if="screenshots\.length < 3"\s+class="dock-hint">/)
assert.match(dock, /screenshots\.length < 3 \? '继续截图' : '已满 3 张'/)

const orderedTokens = [
  '{{ solveShortcut }}',
  '{{ sendShortcut }}',
  '{{ fastSendShortcut }}',
  '{{ cancelSolveShortcut }}',
  '{{ deleteShortcut }}',
]
let previousIndex = -1
for (const token of orderedTokens) {
  const index = dock.indexOf(token)
  assert.ok(index > previousIndex, `${token} must appear in F8-F12 solve order`)
  previousIndex = index
}
assert.match(dock, />发送</)
assert.match(dock, /快速解题/)
assert.match(dock, />中断</)
assert.match(dock, />删除</)
assert.match(dock, /on\('screenshot-taken'/)
assert.match(dock, /on\('screenshot-removed'/)
assert.match(dock, /event\?\.event === 'request-start'/)

console.log('Screenshot dock regression tests passed')
