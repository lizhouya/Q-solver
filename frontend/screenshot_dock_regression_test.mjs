import assert from 'node:assert/strict'
import fs from 'node:fs'

const dock = fs.readFileSync(new URL('./src/components/ScreenshotDock.vue', import.meta.url), 'utf8')

assert.match(dock, /import\s*\{\s*storeToRefs\s*\}\s*from\s*['"]pinia['"]/)
assert.match(
  dock,
  /const\s*\{\s*solveShortcut\s*,\s*sendShortcut\s*,\s*deleteShortcut\s*\}\s*=\s*storeToRefs\(settingsStore\)/,
)
assert.match(dock, /\{\{\s*screenshots\.length\s*\}\}\/3/)
assert.match(dock, /v-if="screenshots\.length < 3"/)
assert.match(dock, /\{\{\s*solveShortcut\s*\}\}/)
assert.match(dock, />继续截图</)
assert.match(dock, /v-else[^>]*>已满 3 张</)
assert.match(dock, /\{\{\s*deleteShortcut\s*\}\}/)
assert.match(dock, /\{\{\s*sendShortcut\s*\}\}/)
assert.match(dock, /on\('screenshot-taken'/)
assert.match(dock, /on\('screenshot-removed'/)
assert.match(dock, /event\?\.event === 'request-start'/)

console.log('Screenshot dock regression tests passed')
