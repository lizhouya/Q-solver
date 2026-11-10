import assert from 'node:assert/strict'
import fs from 'node:fs'

const modal = fs.readFileSync(new URL('./src/components/SettingsModal.vue', import.meta.url), 'utf8')
const modelTab = modal.match(/\.api-tab-pane\s*\{([^}]*)\}/)?.[1] || ''
assert.match(modelTab, /display:\s*block/, 'unified model/API tab must use normal document flow')
assert.match(modelTab, /overflow-y:\s*auto/, 'unified model/API tab must be the vertical scroll container')
assert.match(modelTab, /overflow-x:\s*hidden/, 'unified model/API tab must suppress horizontal overflow')
assert.equal(/overflow:\s*hidden/.test(modelTab), false, 'unified model/API tab must not clip its contents')

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
assert.ok(modal.includes(`:active="ui.showSettings && ui.activeTab === 'api'"`), 'settings modal must deactivate the dropdown when hidden or navigating away')
const lockedModelTab = modal.match(/\.api-tab-pane\.model-dropdown-open\s*\{([^}]*)\}/)?.[1] || ''
assert.match(lockedModelTab, /overflow-y:\s*hidden/, 'open model dropdown must lock unified page scrolling')
assert.ok(modal.includes('modelDropdownOpen.value = false'), 'settings modal must reset the scroll lock on navigation or close')

const apiPane = modal.slice(modal.indexOf(`v-show="ui.activeTab === 'api'"`), modal.indexOf(`v-show="ui.activeTab === 'scene'"`))
assert.ok(modal.includes(`v-show="ui.activeTab === 'scene'"`), 'scene must have an independent pane')
assert.ok(modal.includes('class="tab-pane scene-tab-pane"'))
assert.equal(apiPane.includes('<DomainSelector'), false, 'API pane must not contain scene selection')
const scenePane = modal.slice(modal.indexOf(`v-show="ui.activeTab === 'scene'"`), modal.indexOf('<div class="modal-footer">'))
assert.ok(scenePane.includes('<DomainSelector'))
assert.equal(scenePane.includes('model-dropdown-open'), false, 'model dropdown must not lock scene scrolling')
const actions = modal.match(/\.connection-actions\s*\{([^}]*)\}/)?.[1] || ''
assert.match(actions, /grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\)/)
assert.match(modal, /class="btn-primary connection-save-action"/)
assert.match(modal.match(/\.connection-actions \.connection-save-action\s*\{([^}]*)\}/)?.[1] || '', /grid-column:\s*1 \/ -1/)
const actionButton = modal.match(/\.connection-actions button\s*\{([^}]*)\}/)?.[1] || ''
assert.match(actionButton, /min-width:\s*0/)
assert.match(actionButton, /white-space:\s*normal/)
assert.match(modal, /@media \(max-width: 420px\)[\s\S]*?\.connection-actions\s*\{\s*grid-template-columns:\s*1fr/)
assert.match(modal, /@container \(max-width: 420px\)[\s\S]*?\.connection-actions\s*\{\s*grid-template-columns:\s*1fr/, 'narrow modal, not only viewport, must use one column')
assert.match(modal.match(/\.modal-content\s*\{([^}]*)\}/)?.[1] || '', /container-type:\s*inline-size/)
for (const selector of ['modal-body', 'tab-pane', 'connection-workspace']) {
  const rule = modal.match(new RegExp(String.raw`\.${selector}\s*\{([^}]*)\}`))?.[1] || ''
  assert.match(rule, /min-width:\s*0/, `${selector} must allow children to shrink`)
}
assert.match(modal, /:deep\(\.model-select-container\)/, 'model child intrinsic width must be constrained')
const footer = modal.match(/\.modal-footer\s*\{([^}]*)\}/)?.[1] || ''
assert.equal(/position:\s*(absolute|fixed)/.test(footer), false)
assert.match(footer, /flex-shrink:\s*0/)
const body = modal.match(/\.modal-body\s*\{([^}]*)\}/)?.[1] || ''
assert.match(body, /min-height:\s*0/)
assert.match(modal.match(/\.tab-pane\s*\{([^}]*)\}/)?.[1] || '', /overflow-y:\s*auto/)
const modalContent = modal.match(/\.modal-content\s*\{([^}]*)\}/)?.[1] || ''
assert.match(modalContent, /width:\s*720px/, 'settings modal must be wide enough to show all tabs without scrolling')
const tabs = modal.match(/\.tabs\s*\{([^}]*)\}/)?.[1] || ''
assert.match(tabs, /flex-wrap:\s*wrap/, 'tabs must wrap only when the window is genuinely narrow')
assert.match(tabs, /overflow:\s*visible/, 'tabs must not use a horizontal scrolling strip')
assert.equal(/overflow-x:\s*auto/.test(tabs), false, 'tabs must not use horizontal scrolling')
assert.match(modal.match(/\.tab\s*\{([^}]*)\}/)?.[1] || '', /flex:\s*0 0 auto/)
assert.match(modal.match(/\.close-btn\s*\{([^}]*)\}/)?.[1] || '', /flex-shrink:\s*0/)
assert.ok(modal.includes(`ui.activeTab = 'output'">显示</button>`), 'output tab label must be renamed to 显示')
assert.equal(modal.includes('scrollIntoView'), false, 'tab navigation must not trigger horizontal scrolling')
assert.equal(modal.includes('tabsElement'), false, 'scroll-only tab refs must be removed')
assert.equal(modal.includes('nextTick'), false, 'scroll-only nextTick dependency must be removed')
assert.match(modal, /<button[^>]*class="tab"/, 'tabs must be keyboard operable')
const categories = modal.match(/\.scene-tab-pane :deep\(\.category-tabs\)\s*\{([^}]*)\}/)?.[1] || ''
assert.match(categories, /flex-wrap:\s*wrap/, 'all scene categories must remain visible without a hidden horizontal scrollbar')
assert.match(categories, /overflow:\s*visible/)
assert.match(modal, /\.scene-tab-pane :deep\(\.item-title\)/)
assert.match(modal, /\.scene-tab-pane :deep\(\.item-desc\)[^{]*\{[^}]*white-space:\s*normal/, 'scene descriptions must not be ellipsized')

const provider = fs.readFileSync(new URL('./src/components/ProviderSelect.vue', import.meta.url), 'utf8')
for (const selector of ['provider-page', 'form-card', 'form-item', 'control', 'advanced-card']) {
  const rule = provider.match(new RegExp(String.raw`\.${selector}\s*\{([^}]*)\}`))?.[1] || ''
  assert.match(rule, /min-width:\s*0/, `${selector} must shrink instead of relying on clipping`)
}
console.log('Model layout regression tests passed')
