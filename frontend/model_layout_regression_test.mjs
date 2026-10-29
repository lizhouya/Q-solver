import assert from 'node:assert/strict'
import fs from 'node:fs'

const modal = fs.readFileSync(new URL('./src/components/SettingsModal.vue', import.meta.url), 'utf8')
const modelTab = modal.match(/\.model-tab\s*\{([^}]*)\}/)?.[1] || ''
assert.match(modelTab, /display:\s*block/, 'model tab must use normal document flow')
assert.match(modelTab, /overflow-y:\s*auto/, 'model tab must be the vertical scroll container')
assert.match(modelTab, /overflow-x:\s*hidden/, 'model tab must suppress horizontal overflow')
assert.equal(/overflow:\s*hidden/.test(modelTab), false, 'model tab must not clip its contents')

const domainGroup = modal.match(/\.domain-group\s*\{([^}]*)\}/)?.[1] || ''
assert.equal(/flex:\s*1/.test(domainGroup), false, 'domain group must not depend on leftover flex height')
assert.equal(modal.includes(':deep(.domain-grid)'), false, 'parent must not target the nonexistent domain-grid class')
assert.ok(modal.includes(':deep(.domain-list)'), 'parent must target the actual domain-list class')
assert.match(modal.match(/:deep\(\.domain-list\)\s*\{([^}]*)\}/)?.[1] || '', /overflow:\s*visible/, 'domain list must not create a nested clipping scroll area')

console.log('Model layout regression tests passed')
