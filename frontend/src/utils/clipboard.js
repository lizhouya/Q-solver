import { ClipboardSetText } from '../../wailsjs/runtime/runtime'

export async function copyText(text) {
  if (typeof text !== 'string' || !text) return false
  try {
    if (window.runtime && await ClipboardSetText(text) === true) return true
  } catch { /* 继续尝试浏览器剪贴板。 */ }
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch { /* HTTP 同步端或权限限制时使用兼容路径。 */ }
  let textarea
  const focused = document.activeElement
  try {
    textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    return document.execCommand('copy') === true
  } catch {
    return false
  } finally {
    textarea?.remove()
    try { focused?.focus({ preventScroll: true }) } catch { /* 原焦点可能已离开页面。 */ }
  }
}

// 仅组合已校验的标签与答案，不解析 Markdown 或重新分配题号。
export function formatNumberedAnswers(items, plainText) {
  if (!plainText || typeof plainText !== 'string' || !Array.isArray(items) || !items.length) return ''
  if (items.some(item => !item || item.status !== 'ready' ||
    typeof item.label !== 'string' || !item.label.trim() || /[\r\n]/.test(item.label) || item.label.includes('待核对') ||
    typeof item.value !== 'string' || !item.value || /[\r\n]/.test(item.value))) return ''
  if (items.map(item => item.value).join('\n') !== plainText) return ''
  return items.map(item => `${item.label}. ${item.value}`).join('\n')
}
