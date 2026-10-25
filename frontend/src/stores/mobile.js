import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import { api } from '../services/api'
import { useUIStore } from './ui'

const emptyInfo = () => ({
  available: false,
  url: '',
  port: 0,
  status: 'not_started',
  retryable: false,
  stableLinkEnabled: true,
  canClear: false,
  canResetLink: false,
})

function normalizeInfo(value) {
  const available = value?.available === true
  return {
    ...emptyInfo(),
    ...(value || {}),
    available,
    url: available ? String(value?.url || '') : '',
    port: available ? Number(value?.port || 0) : 0,
    status: value?.status || (available ? 'ready' : 'not_started'),
    retryable: value?.retryable === true,
    stableLinkEnabled: value?.stableLinkEnabled !== false,
    canClear: value?.canClear === true,
    canResetLink: value?.canResetLink === true,
  }
}

const actionCopy = {
  clear: {
    title: '清空同步端显示？',
    description: '同步页面会回到待命状态，并清除页面内的答案、错误与历史。桌面答案、题库、截图和模型上下文不会删除。',
    confirmLabel: '确认清空',
  },
  reset: {
    title: '重置同步链接？',
    description: '当前链接和已连接的同步页面会失效，新链接从空白状态开始。请重新复制并发送新地址。',
    confirmLabel: '重置链接',
  },
  stable: {
    title: '切换固定链接模式？',
    description: '切换会立即更换同步链接并清空同步页面。关闭后，每次完整启动都会生成新的临时链接；同一进程内重试不会更换。',
    confirmLabel: '确认切换',
  },
}

export const useMobileStore = defineStore('mobile', () => {
  const ui = useUIStore()
  const info = reactive(emptyInfo())
  const loading = ref(false)
  const readError = ref(false)
  const mutating = ref(false)
  const pendingAction = ref(null)
  const lastCode = ref('')
  let requestVersion = 0

  const busy = computed(() => loading.value || mutating.value)

  function applyInfo(value) {
    Object.assign(info, normalizeInfo(value))
    readError.value = false
  }

  async function refresh() {
    if (mutating.value) return false
    const version = ++requestVersion
    loading.value = true
    readError.value = false
    try {
      const value = await api.getMobileOutputInfo()
      if (version !== requestVersion) return false
      applyInfo(value)
      return true
    } catch (_) {
      if (version === requestVersion) readError.value = true
      return false
    } finally {
      if (version === requestVersion) loading.value = false
    }
  }

  async function retry() {
    if (busy.value || pendingAction.value) return false
    const version = ++requestVersion
    mutating.value = true
    readError.value = false
    try {
      const value = await api.retryMobileOutput()
      if (version !== requestVersion) return false
      applyInfo(value)
      ui.showToast(value?.available ? 'Mobile 服务已恢复' : 'Mobile 服务仍不可用，请检查端口占用', value?.available ? 'success' : 'warning')
      return value?.available === true
    } catch (_) {
      if (version === requestVersion) {
        readError.value = true
        ui.showToast('重试 Mobile 服务失败', 'error')
      }
      return false
    } finally {
      if (version === requestVersion) mutating.value = false
    }
  }

  function setPending(type, extra = {}) {
    if (busy.value || pendingAction.value) return false
    pendingAction.value = { type, ...actionCopy[type], ...extra }
    return true
  }

  function requestClear() {
    if (!info.canClear) return false
    return setPending('clear')
  }

  function requestReset() {
    if (!info.canResetLink) return false
    return setPending('reset')
  }

  function requestStableMode(enabled) {
    if (!info.canResetLink || enabled === info.stableLinkEnabled) return false
    return setPending('stable', {
      enabled,
      title: enabled ? '启用固定同步链接？' : '关闭固定同步链接？',
      confirmLabel: enabled ? '启用并换链接' : '关闭并换链接',
    })
  }

  function cancelPendingAction() {
    if (mutating.value) return false
    pendingAction.value = null
    return true
  }

  function successMessage(action, code) {
    if (code === 'already_current') return '同步链接模式未变化'
    if (action.type === 'clear') return '同步端清屏状态已提交并广播'
    if (action.type === 'reset') return '同步链接已重置，请复制新地址'
    return action.enabled ? '固定同步链接已启用' : '固定同步链接已关闭'
  }

  async function confirmPendingAction() {
    const action = pendingAction.value
    if (!action || busy.value) return false

    const version = ++requestVersion
    loading.value = false
    mutating.value = true
    lastCode.value = ''
    try {
      let result
      if (action.type === 'clear') result = await api.clearMobileDisplay(true)
      else if (action.type === 'reset') result = await api.resetMobileOutputLink(true)
      else result = await api.setMobileStableLink(action.enabled, true)

      if (version !== requestVersion) return false
      lastCode.value = result?.code || ''
      if (result?.success !== true) {
        ui.showToast('Mobile 操作未完成，请稍后重试', 'error')
        return false
      }
      if (result.info) applyInfo(result.info)
      ui.showToast(successMessage(action, result.code), 'success', 3500)
      return true
    } catch (_) {
      if (version === requestVersion) ui.showToast('Mobile 操作失败，原链接与状态保持不变', 'error')
      return false
    } finally {
      if (version === requestVersion) {
        mutating.value = false
        pendingAction.value = null
      }
    }
  }

  return {
    info,
    loading,
    readError,
    mutating,
    pendingAction,
    lastCode,
    busy,
    refresh,
    retry,
    requestClear,
    requestReset,
    requestStableMode,
    cancelPendingAction,
    confirmPendingAction,
  }
})
