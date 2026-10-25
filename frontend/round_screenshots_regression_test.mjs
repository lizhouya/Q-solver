import assert from 'node:assert/strict'
import fs from 'node:fs'

const screenshots = fs.readFileSync(new URL('./src/components/RoundScreenshots.vue', import.meta.url), 'utf8')
const solveView = fs.readFileSync(new URL('./src/components/SolveView.vue', import.meta.url), 'utf8')

// 展开状态必须与图片缓存解耦，默认集合为空，并按 Session/Record 作用域重置。
assert.ok(screenshots.includes("import { ref, watch } from 'vue'"))
assert.ok(screenshots.includes('const expandedImages = ref(new Set())'))
assert.ok(screenshots.includes('scopeKey: { type: String, required: true }'))
assert.ok(screenshots.includes('watch(() => props.scopeKey'))
assert.ok(screenshots.includes('expandedImages.value = new Set()'))

// 每张截图使用可访问的独立开关，展开后按钮或图片都能收起。
assert.ok(screenshots.includes(':aria-expanded="isExpanded(image)"'))
assert.ok(screenshots.includes('显示截图（'))
assert.ok(screenshots.includes('隐藏截图（'))
assert.ok(screenshots.includes('@click="toggleImage(image)"'))
assert.ok(screenshots.includes('@click="collapseImage(image)"'))
assert.ok(screenshots.includes(':aria-label="`隐藏题目截图 ${image.index + 1}`"'))
assert.match(screenshots, /:focus-visible/)

// 只有展开且未缓存、未加载时才继续调用既有 load 事件。
assert.match(screenshots, /if\s*\(!image\.data\s*&&\s*!image\.loading\)\s*\{\s*emit\('load',\s*props\.round,\s*image\)/)
assert.equal(/image\.data\s*=/.test(screenshots), false)
assert.equal(/image\.loading\s*=/.test(screenshots), false)
assert.equal(/image\.error\s*=/.test(screenshots), false)
assert.equal(/image\.requestVersion\s*=/.test(screenshots), false)
assert.equal(/round\.userScreenshot\s*=/.test(screenshots), false)

// SolveView 只传显示作用域，图片加载仍复用 solution.loadRoundImage。
assert.ok(solveView.includes(':scope-key="roundScopeKey(round, idx)"'))
assert.ok(solveView.includes('@load="solution.loadRoundImage"'))
assert.ok(solveView.includes('solution.currentHistoryItem'))
assert.ok(solveView.includes('round.recordId || round.localId || idx'))

console.log('Round screenshots regression tests passed')
