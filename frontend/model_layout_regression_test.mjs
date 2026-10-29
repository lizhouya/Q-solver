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

const modelSelect = fs.readFileSync(new URL('./src/components/ModelSelect.vue', import.meta.url), 'utf8')
assert.ok(modelSelect.includes('@wheel.stop'), 'model dropdown must stop wheel propagation')
assert.ok(modelSelect.includes('@touchmove.stop'), 'model dropdown must stop touch scrolling propagation')
const dropdownPortal = modelSelect.match(/\.model-dropdown-portal\s*\{([^}]*)\}/)?.[1] || ''
assert.match(dropdownPortal, /overscroll-behavior:\s*contain/, 'model dropdown must contain boundary scroll chaining')
assert.match(dropdownPortal, /touch-action:\s*pan-y/, 'model dropdown must preserve native vertical touch scrolling')
assert.ok(modelSelect.includes("'open-change'"), 'model select must expose dropdown open state')
assert.ok(modelSelect.includes("emit('open-change', value)"), 'model select must notify every open-state transition')
assert.ok(modelSelect.includes('watch(() => props.active'), 'inactive model tab must close the teleported dropdown')

assert.ok(modal.includes("'model-dropdown-open': modelDropdownOpen"), 'model tab must bind the dropdown scroll-lock class')
assert.ok(modal.includes('@open-change="modelDropdownOpen = $event"'), 'settings modal must receive dropdown state changes')
assert.ok(modal.includes(`:active="ui.showSettings && ui.activeTab === 'model'"`), 'settings modal must deactivate the dropdown when hidden or navigating away')
const lockedModelTab = modal.match(/\.model-tab\.model-dropdown-open\s*\{([^}]*)\}/)?.[1] || ''
assert.match(lockedModelTab, /overflow-y:\s*hidden/, 'open model dropdown must lock outer model-page scrolling')
assert.ok(modal.includes('modelDropdownOpen.value = false'), 'settings modal must reset the scroll lock on navigation or close')

console.log('Model layout regression tests passed')
