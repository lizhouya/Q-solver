/**
 * Solution Store — 解题结果、流式输出、历史记录
 */
import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'
import { api } from '../services/api'

export const useSolutionStore = defineStore('solution', () => {
  const history = ref([])
  const activeHistoryIndex = ref(0)
  const selectedRoundTarget = ref(null)
  const streamTarget = ref(null)
  const legacyStreamTarget = ref(null)
  const isLoading = ref(false)
  const isAppending = ref(false)
  const shouldOverwriteHistory = ref(false)

  // Thinking chain
  const isThinking = ref(false)
  const thinkingStatusText = ref('Thinking Process')
  const thinkingExpanded = ref(true)
  const isThinkingStalled = ref(false)

  // Stream state (not reactive — internal)
  let streamBuffer = ''
  let thinkingBuffer = ''
  let thinkingStartTime = 0
  let stallTimer = null
  let pendingUserScreenshot = ''
  let localHistorySequence = 0
  let localRoundSequence = 0

  // 流式渲染：节流 + 全量渲染
  let renderRafId = null
  let renderDirty = false
  const streamingHtml = ref('')

  // Error state
  const errorState = reactive({
    show: false,
    icon: '⚠️',
    title: '出错了',
    desc: '发生了一个未知错误',
    rawError: '',
    showDetails: false,
  })

  // ---- Helpers ----

  function renderMarkdown(md) {
    if (!md) return ''
    return renderMarkdownWithLatex(md)
  }

  function nextLocalHistoryId() {
    localHistorySequence += 1
    return `legacy-session-${localHistorySequence}`
  }

  function nextLocalRoundId() {
    localRoundSequence += 1
    return `legacy-round-${localRoundSequence}`
  }

  function formatRoundTime(value, fallback = '') {
    if (!value) return fallback
    const parsed = new Date(value)
    return Number.isNaN(parsed.getTime()) ? fallback : parsed.toLocaleTimeString()
  }

  function getRoundSummary(round) {
    const text = round?.aiResponse || ''
    return text.substring(0, 30).replace(/\n/g, ' ') + '...'
  }

  function getSummary(item) {
    if (!item) return ''
    if (!item.rounds?.length) return item.summary || ''
    return getRoundSummary(item.rounds[item.rounds.length - 1])
  }

  function getRoundsCount(item) {
    return item?.rounds?.length || 1
  }

  function getFullContent(item) {
    if (!item) return ''
    if (!item.rounds?.length) return item.full || ''
    return item.rounds.map(r => r.aiResponse || '').join('\n\n---\n\n')
  }

  function createLegacyRound(userScreenshot) {
    return {
      localId: nextLocalRoundId(),
      status: 'running',
      time: new Date().toLocaleTimeString(),
      userScreenshot: userScreenshot || '',
      thinking: '',
      thinkingStatus: 'Thinking Process',
      thinkingDuration: 0,
      aiResponse: '',
      answerItems: [],
      answerCopyText: '',
      error: null,
    }
  }

  function createHistoryItem(userScreenshot) {
    return {
      localId: nextLocalHistoryId(),
      time: new Date().toLocaleTimeString(),
      rounds: [createLegacyRound(userScreenshot)],
    }
  }

  function addRoundToItem(item, userScreenshot) {
    if (!item.rounds) item.rounds = []
    item.rounds.push(createLegacyRound(userScreenshot))
    return getCurrentRound(item)
  }

  function getCurrentRound(item) {
    if (!item?.rounds?.length) return null
    return item.rounds[item.rounds.length - 1]
  }

  function resetStreamState() {
    streamBuffer = ''
    thinkingBuffer = ''
    thinkingStartTime = 0
    isThinking.value = false
    thinkingStatusText.value = 'Thinking Process'
    thinkingExpanded.value = true
    isThinkingStalled.value = false
    if (renderRafId) { cancelAnimationFrame(renderRafId); renderRafId = null }
    renderDirty = false
    streamingHtml.value = ''
    if (stallTimer) { clearTimeout(stallTimer); stallTimer = null }
  }

  function normalizeArchivedRound(record) {
    return {
      recordId: record.id,
      time: formatRoundTime(record.createdAt),
      status: record.status,
      model: record.model || '',
      scene: record.scene || '',
      keepContext: record.keepContext,
      images: (record.images || []).map(image => ({
        index: image.index,
        mime: image.mime || '',
        data: '',
        loading: false,
        error: '',
        requestVersion: 0,
      })),
      userScreenshot: '',
      thinking: '',
      thinkingStatus: 'Thinking Process',
      thinkingDuration: 0,
      aiResponse: record.answer || '',
      answerItems: record.status === 'completed' ? (record.answerItems || []) : [],
      answerCopyText: record.status === 'completed' ? (record.answerCopyText || '') : '',
      error: null,
    }
  }

  function roundTarget(item, round) {
    if (item?.sessionId && round?.recordId) {
      return {
        kind: 'typed',
        sessionId: item.sessionId,
        epoch: Number(item.epoch || 0),
        recordId: round.recordId,
      }
    }
    if (!item?.localId || !round?.localId) return null
    return { kind: 'legacy', itemId: item.localId, roundId: round.localId }
  }

  function roundMatchesTarget(item, round, target) {
    if (!item || !round || !target) return false
    if (target.kind === 'typed') {
      return item.sessionId === target.sessionId &&
        Number(item.epoch) === Number(target.epoch) &&
        round.recordId === target.recordId
    }
    return target.kind === 'legacy' &&
      item.localId === target.itemId &&
      round.localId === target.roundId
  }

  function locateRoundTarget(target) {
    if (!target) return null
    for (let sessionIndex = 0; sessionIndex < history.value.length; sessionIndex += 1) {
      const item = history.value[sessionIndex]
      const rounds = item.rounds || []
      for (let roundIndex = 0; roundIndex < rounds.length; roundIndex += 1) {
        const round = rounds[roundIndex]
        if (roundMatchesTarget(item, round, target)) {
          return { item, round, sessionIndex, roundIndex }
        }
      }
    }
    return null
  }

  function selectRound(item, round) {
    const sessionIndex = history.value.indexOf(item)
    if (sessionIndex < 0 || !item?.rounds?.includes(round)) return false
    const target = roundTarget(item, round)
    if (!target) return false
    activeHistoryIndex.value = sessionIndex
    selectedRoundTarget.value = target
    return true
  }

  function selectLatestRound(index) {
    const item = history.value[index]
    if (!item) return false
    activeHistoryIndex.value = index
    const round = getCurrentRound(item)
    if (!round) {
      selectedRoundTarget.value = null
      return true
    }
    return selectRound(item, round)
  }

  function activateQuestionBankSession(continuation) {
    const detail = continuation?.detail
    if (!detail?.id) return false
    resetStreamState()
    streamTarget.value = null
    legacyStreamTarget.value = null
    selectedRoundTarget.value = null
    isLoading.value = false
    isAppending.value = false
    shouldOverwriteHistory.value = false
    history.value.forEach(item => { item.isWritable = false })
    const item = {
      id: `session:${detail.id}`,
      sessionId: detail.id,
      epoch: Number(continuation.epoch || 0),
      source: 'questionbank',
      isWritable: true,
      contextQuality: continuation.contextQuality || '',
      contextWarning: continuation.contextWarning || '',
      name: detail.name || [detail.company, detail.issuer, detail.type].filter(Boolean).join('_') || '笔试题库',
      state: detail.state,
      time: detail.startedAt ? new Date(detail.startedAt).toLocaleTimeString() : new Date().toLocaleTimeString(),
      rounds: (detail.records || []).map(normalizeArchivedRound),
    }
    history.value = [item, ...history.value.filter(existing => existing.sessionId !== detail.id)]
    selectLatestRound(0)
    return true
  }

  function locateTargetRound(target = streamTarget.value) {
    if (!target) return null
    const item = history.value.find(candidate => (
      candidate.sessionId === target.sessionId && Number(candidate.epoch) === Number(target.epoch)
    ))
    if (!item) return null
    const round = item.rounds?.find(candidate => candidate.recordId === target.recordId)
    return round ? { item, round } : null
  }

  function beginDesktopRequest(event) {
    resetStreamState()
    legacyStreamTarget.value = null
    errorState.show = false
    history.value.forEach(item => { item.isWritable = false })
    let item = history.value.find(candidate => candidate.sessionId === event.sessionId)
    if (!item) {
      item = {
        id: `session:${event.sessionId}`,
        sessionId: event.sessionId,
        source: event.sessionId === 'live' ? 'live' : 'questionbank',
        rounds: [],
        time: new Date().toLocaleTimeString(),
      }
    }
    item.epoch = Number(event.epoch || 0)
    item.isWritable = true
    item.state = 'active'
    let round = item.rounds.find(candidate => candidate.recordId === event.recordId)
    if (!round) {
      round = {
        recordId: event.recordId,
        time: new Date().toLocaleTimeString(),
        status: 'running',
        images: (event.images || []).map((data, index) => ({ index, mime: '', data, loading: false, error: '', requestVersion: 0 })),
        userScreenshot: event.images?.[0] || '',
        thinking: '',
        thinkingStatus: 'Thinking Process',
        thinkingDuration: 0,
        aiResponse: '',
      answerItems: [],
      answerCopyText: '',
        error: null,
      }
      item.rounds.push(round)
    }
    history.value = [item, ...history.value.filter(candidate => candidate !== item)]
    const activeItem = history.value[0]
    const activeRound = activeItem.rounds.find(candidate => candidate.recordId === event.recordId)
    streamTarget.value = { epoch: activeItem.epoch, sessionId: activeItem.sessionId, recordId: event.recordId }
    selectRound(activeItem, activeRound)
    isAppending.value = activeItem.rounds.length > 1
    isLoading.value = !isAppending.value
    return activeRound
  }

  function desktopEventMatches(event) {
    const target = streamTarget.value
    return Boolean(target && event &&
      Number(event.epoch) === Number(target.epoch) &&
      event.sessionId === target.sessionId &&
      event.recordId === target.recordId)
  }

  function handleDesktopSolveEvent(event) {
    if (!event?.event || !event.sessionId || !event.recordId) return false
    if (event.event === 'request-start') {
      beginDesktopRequest(event)
      return true
    }
    if (!desktopEventMatches(event)) return false
    const located = locateTargetRound()
    if (!located) return false
    const { round } = located
    const text = event.text || ''
    switch (event.event) {
      case 'solution-stream-start':
        resetStreamState()
        break
      case 'solution-stream-thinking':
        if (!isThinking.value) thinkingStartTime = Date.now()
        isThinking.value = true
        isLoading.value = false
        isAppending.value = false
        thinkingBuffer += text
        round.thinking = thinkingBuffer
        round.thinkingStatus = thinkingStatusText.value
        break
      case 'solution-stream-chunk':
        isLoading.value = false
        isAppending.value = false
        isThinking.value = false
        streamBuffer += text
        round.aiResponse = streamBuffer
        streamingHtml.value = renderMarkdownWithLatex(streamBuffer)
        break
      case 'solution':
        round.answerItems = event.answerItems || []
        round.answerCopyText = event.answerCopyText || ''
        round.aiResponse = text || round.aiResponse
        round.status = 'completed'
        resetStreamState()
        isLoading.value = false
        isAppending.value = false
        streamTarget.value = null
        break
      case 'solution-error':
        round.answerItems = []
        round.answerCopyText = ''
        round.status = text.includes('canceled') ? 'cancelled' : 'failed'
        round.error = { title: '请求出错', desc: text || '未知错误', icon: 'x-circle' }
        resetStreamState()
        isLoading.value = false
        isAppending.value = false
        streamTarget.value = null
        break
      case 'hidden':
        round.status = event.success ? 'completed' : 'failed'
        resetStreamState()
        isLoading.value = false
        isAppending.value = false
        streamTarget.value = null
        break
      default:
        return false
    }
    scrollContentToBottom()
    return true
  }

  async function loadRoundImage(round, image) {
    if (!round?.recordId || !image || image.data || image.loading) return false
    const item = history.value.find(candidate => candidate.rounds?.includes(round))
    if (!item?.sessionId) return false
    const version = Number(image.requestVersion || 0) + 1
    image.requestVersion = version
    image.loading = true
    image.error = ''
    try {
      const data = await api.getQuestionBankRecordImage(item.sessionId, round.recordId, image.index)
      if (!history.value.includes(item) || image.requestVersion !== version) return false
      image.data = data
      if (!round.userScreenshot && image.index === 0) round.userScreenshot = data
      return true
    } catch (error) {
      if (image.requestVersion === version) image.error = error?.message || String(error)
      return false
    } finally {
      if (image.requestVersion === version) image.loading = false
    }
  }

  function getThinkingPreview(thinking) {
    if (!thinking) return ''
    const lines = thinking.split('\n').filter(l => l.trim())
    if (lines.length === 0) return 'Just started...'
    const lastLine = lines[lines.length - 1]
    return lastLine.length > 80 ? lastLine.substring(0, 80) + '...' : lastLine
  }

  // ---- Computed ----
  const historyEntries = computed(() => history.value.flatMap((item, sessionIndex) => {
    const rounds = item.rounds || []
    const entries = []
    for (let roundIndex = rounds.length - 1; roundIndex >= 0; roundIndex -= 1) {
      const round = rounds[roundIndex]
      const typed = Boolean(item.sessionId && round.recordId)
      entries.push({
        key: typed
          ? `typed:${item.sessionId}:${round.recordId}`
          : `legacy:${item.localId}:${round.localId}`,
        sessionIndex,
        roundIndex,
        sessionId: item.sessionId || '',
        epoch: Number(item.epoch || 0),
        recordId: round.recordId || '',
        parent: item,
        round,
        roundLabel: `第 ${roundIndex + 1} 轮`,
        time: round.time || item.time || '',
        summary: getRoundSummary(round),
      })
    }
    return entries
  }))
  const selectedRoundLocation = computed(() => locateRoundTarget(selectedRoundTarget.value))
  const currentRounds = computed(() => {
    const located = selectedRoundLocation.value
    return located ? [located.round] : []
  })
  const currentHistoryItem = computed(() => (
    selectedRoundLocation.value?.item || history.value[activeHistoryIndex.value] || null
  ))
  const isViewingStreamTarget = computed(() => {
    const selected = selectedRoundTarget.value
    const typed = streamTarget.value
    if (selected?.kind === 'typed' && typed) {
      return selected.sessionId === typed.sessionId &&
        Number(selected.epoch) === Number(typed.epoch) &&
        selected.recordId === typed.recordId
    }
    const legacy = legacyStreamTarget.value
    return Boolean(selected?.kind === 'legacy' && legacy &&
      selected.itemId === legacy.itemId && selected.roundId === legacy.roundId)
  })

  // ---- Core actions ----

  function isHistoryEntrySelected(entry) {
    return roundMatchesTarget(entry?.parent, entry?.round, selectedRoundTarget.value)
  }

  function selectHistoryEntry(entry) {
    if (!entry?.parent || !entry?.round) return false
    return selectRound(entry.parent, entry.round)
  }

  function selectHistory(idx) {
    return selectLatestRound(idx)
  }

  function handleStreamStart(keepContext) {
    streamBuffer = ''
    thinkingBuffer = ''
    thinkingStartTime = 0
    isThinking.value = false
    thinkingStatusText.value = 'Thinking Process'
    thinkingExpanded.value = true
    isThinkingStalled.value = false
    streamTarget.value = null
    if (renderRafId) { cancelAnimationFrame(renderRafId); renderRafId = null }
    renderDirty = false
    streamingHtml.value = ''
    if (stallTimer) clearTimeout(stallTimer)

    if (keepContext && history.value.length > 0 && !shouldOverwriteHistory.value) {
      const currentItem = history.value[0]
      const lastRound = getCurrentRound(currentItem)
      if (lastRound && lastRound.error) {
        lastRound.userScreenshot = pendingUserScreenshot || lastRound.userScreenshot
        lastRound.time = new Date().toLocaleTimeString()
        lastRound.thinking = ''
        lastRound.thinkingStatus = 'Thinking Process'
        lastRound.aiResponse = ''
        lastRound.error = null
      } else {
        addRoundToItem(currentItem, pendingUserScreenshot)
      }
    } else {
      if (shouldOverwriteHistory.value && history.value.length > 0) {
        history.value[0] = createHistoryItem(pendingUserScreenshot)
        shouldOverwriteHistory.value = false
      } else {
        history.value.unshift(createHistoryItem(pendingUserScreenshot))
      }
    }

    const item = history.value[0]
    const round = getCurrentRound(item)
    legacyStreamTarget.value = roundTarget(item, round)
    selectRound(item, round)
    pendingUserScreenshot = ''
  }

  function handleStreamChunk(token) {
    if (isLoading.value) isLoading.value = false
    if (isAppending.value) isAppending.value = false
    if (isThinking.value) {
      isThinking.value = false
      thinkingExpanded.value = false
      if (stallTimer) clearTimeout(stallTimer)
    }
    streamBuffer += token

    const located = locateRoundTarget(legacyStreamTarget.value)
    if (located) located.round.aiResponse = streamBuffer

    // 用 rAF 节流渲染，避免每个 token 都触发 DOM 更新
    if (!renderDirty) {
      renderDirty = true
      renderRafId = requestAnimationFrame(() => {
        renderDirty = false
        renderRafId = null
        streamingHtml.value = renderMarkdownWithLatex(streamBuffer)
        scrollContentToBottom()
      })
    }
  }

  function handleThinkingChunk(token) {
    if (isLoading.value) isLoading.value = false
    if (isAppending.value) isAppending.value = false
    if (!isThinking.value) {
      thinkingStartTime = Date.now()
      thinkingExpanded.value = true
    }
    isThinking.value = true
    isThinkingStalled.value = false
    if (stallTimer) clearTimeout(stallTimer)
    stallTimer = setTimeout(() => {
      if (isThinking.value) isThinkingStalled.value = true
    }, 1000)

    thinkingBuffer += token

    if (token.length > 2) {
      if (token.match(/search|web|google/i)) thinkingStatusText.value = 'Searching Internet...'
      else if (token.match(/code|function|import|class/i)) thinkingStatusText.value = 'Generating Code...'
      else if (token.match(/error|bug|fix/i)) thinkingStatusText.value = 'Debugging...'
      else if (token.match(/analyse|analyze|review/i)) thinkingStatusText.value = 'Analyzing...'
    }

    const located = locateRoundTarget(legacyStreamTarget.value)
    if (located) {
      located.round.thinking = thinkingBuffer
      located.round.thinkingStatus = thinkingStatusText.value
    }
    scrollContentToBottom()
  }

  function handleSolution(data) {
    isLoading.value = false
    isAppending.value = false
    isThinking.value = false
    if (stallTimer) clearTimeout(stallTimer)
    thinkingStartTime = 0
    if (renderRafId) { cancelAnimationFrame(renderRafId); renderRafId = null }
    renderDirty = false
    streamingHtml.value = ''
    const located = locateRoundTarget(legacyStreamTarget.value)
    if (located?.round) {
      located.round.aiResponse = data || located.round.aiResponse
      located.round.status = 'completed'
    }
    legacyStreamTarget.value = null
  }

  function handleInlineError(errorInfo) {
    const located = locateRoundTarget(legacyStreamTarget.value)
    if (located?.round) {
      located.round.status = 'failed'
      located.round.answerItems = []
      located.round.answerCopyText = ''
      located.round.error = {
        title: errorInfo.title || '请求出错',
        desc: errorInfo.desc || '未知错误',
        icon: errorInfo.icon || '❌',
      }
      isLoading.value = false
      isAppending.value = false
      isThinking.value = false
      legacyStreamTarget.value = null
      return true
    }
    return false
  }

  function clearInlineError() {
    const located = locateRoundTarget(legacyStreamTarget.value) || locateRoundTarget(selectedRoundTarget.value)
    if (located?.round) located.round.error = null
  }

  function setStreamBuffer(val) { streamBuffer = val }
  function setUserScreenshot(screenshot) { pendingUserScreenshot = screenshot }
  function resetLiveHistory() {
    if (renderRafId) {
      cancelAnimationFrame(renderRafId)
      renderRafId = null
    }
    if (stallTimer) {
      clearTimeout(stallTimer)
      stallTimer = null
    }
    history.value = []
    activeHistoryIndex.value = 0
    selectedRoundTarget.value = null
    streamTarget.value = null
    legacyStreamTarget.value = null
    isLoading.value = false
    isAppending.value = false
    shouldOverwriteHistory.value = false
    isThinking.value = false
    thinkingStatusText.value = 'Thinking Process'
    thinkingExpanded.value = true
    isThinkingStalled.value = false
    streamBuffer = ''
    thinkingBuffer = ''
    thinkingStartTime = 0
    pendingUserScreenshot = ''
    renderDirty = false
    streamingHtml.value = ''
    Object.assign(errorState, {
      show: false,
      icon: '⚠️',
      title: '出错了',
      desc: '发生了一个未知错误',
      rawError: '',
      showDetails: false,
    })
  }


  function deleteHistory(index) {
    if (index < 0 || index >= history.value.length) return false
    const removedItem = history.value[index]
    const selectedBeforeDelete = locateRoundTarget(selectedRoundTarget.value)
    const removesSelected = selectedBeforeDelete?.item === removedItem
    const typedTarget = streamTarget.value
    const removesTypedStream = Boolean(typedTarget && removedItem.sessionId === typedTarget.sessionId &&
      Number(removedItem.epoch) === Number(typedTarget.epoch))
    const legacyTarget = legacyStreamTarget.value
    const removesLegacyStream = Boolean(legacyTarget && removedItem.localId === legacyTarget.itemId)

    history.value.splice(index, 1)

    if (removesTypedStream || removesLegacyStream) {
      resetStreamState()
      streamTarget.value = null
      legacyStreamTarget.value = null
      isLoading.value = false
      isAppending.value = false
    }

    if (history.value.length === 0) {
      activeHistoryIndex.value = 0
      selectedRoundTarget.value = null
      return true
    }

    const selectedAfterDelete = locateRoundTarget(selectedRoundTarget.value)
    if (removesSelected || !selectedAfterDelete) {
      selectLatestRound(Math.min(index, history.value.length - 1))
    } else {
      activeHistoryIndex.value = selectedAfterDelete.sessionIndex
    }
    return true
  }

  function scrollContentToBottom() {
    const el = document.getElementById('content')
    if (el) el.scrollTop = el.scrollHeight
  }

  // ---- Export image (migrated from useSolution) ----

  function createRoundCard(round, roundIndex, totalRounds) {
    const card = document.createElement('div')
    card.style.cssText = `display:flex;gap:24px;align-items:stretch;margin-bottom:${roundIndex < totalRounds - 1 ? '24px' : '0'};padding-bottom:${roundIndex < totalRounds - 1 ? '24px' : '0'};border-bottom:${roundIndex < totalRounds - 1 ? '1px dashed #cbd5e1' : 'none'};`
    // Left panel - user input
    const leftPanel = document.createElement('div')
    leftPanel.style.cssText = 'flex:0 0 240px;background:white;border-radius:12px;padding:16px;box-shadow:0 2px 4px rgba(0,0,0,0.05);border:1px solid #e2e8f0;'
    const userHeader = document.createElement('div')
    userHeader.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:12px;padding-bottom:10px;border-bottom:1px solid #f1f5f9;'
    userHeader.innerHTML = `<div style="width:28px;height:28px;border-radius:50%;background:linear-gradient(135deg,#6366f1,#4f46e5);display:flex;align-items:center;justify-content:center;color:white;font-size:12px;">👤</div><div><div style="font-weight:600;font-size:12px;color:#334155;">问题 ${roundIndex + 1}</div></div>`
    leftPanel.appendChild(userHeader)
    if (round.userScreenshot) {
      const imgC = document.createElement('div')
      imgC.innerHTML = `<img src="${round.userScreenshot}" style="width:100%;border-radius:6px;border:1px solid #e2e8f0;" />`
      leftPanel.appendChild(imgC)
    } else {
      const ph = document.createElement('div')
      ph.style.cssText = 'padding:20px;text-align:center;color:#94a3b8;font-size:12px;background:#f8fafc;border-radius:6px;'
      ph.textContent = '无截图'
      leftPanel.appendChild(ph)
    }
    card.appendChild(leftPanel)

    // Right panel - AI response
    const rightPanel = document.createElement('div')
    rightPanel.style.cssText = 'flex:1;background:white;border-radius:12px;padding:16px;box-shadow:0 2px 4px rgba(0,0,0,0.05);border:1px solid #e2e8f0;overflow:hidden;'
    const aiHeader = document.createElement('div')
    aiHeader.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:12px;padding-bottom:10px;border-bottom:1px solid #f1f5f9;'
    aiHeader.innerHTML = `<div style="width:28px;height:28px;border-radius:50%;background:linear-gradient(135deg,#6366f1,#818cf8);display:flex;align-items:center;justify-content:center;color:white;font-size:12px;">🤖</div><div><div style="font-weight:600;font-size:12px;color:#334155;">AI 回复</div></div>`
    rightPanel.appendChild(aiHeader)
    const aiContent = document.createElement('div')
    aiContent.style.cssText = 'font-size:13px;line-height:1.6;color:#334155;'
    aiContent.innerHTML = renderMarkdown(round.aiResponse || '')
    rightPanel.appendChild(aiContent)
    card.appendChild(rightPanel)
    return card
  }

  async function exportImage(index) {
    const item = history.value[index]
    if (!item) return
    const rounds = item.rounds || []
    if (rounds.length === 0) return
    try {
      const { default: html2canvas } = await import('html2canvas')
      const container = document.createElement('div')
      container.style.cssText = 'position:fixed;left:-9999px;top:0;width:900px;padding:28px;background:linear-gradient(135deg,#f8fafc 0%,#e2e8f0 100%);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#1e293b;border-radius:16px;'
      if (rounds.length > 1) {
        const title = document.createElement('div')
        title.style.cssText = 'font-size:14px;font-weight:600;color:#64748b;margin-bottom:20px;padding-bottom:12px;border-bottom:1px solid #cbd5e1;'
        title.textContent = `共 ${rounds.length} 轮对话`
        container.appendChild(title)
      }
      rounds.forEach((round, idx) => container.appendChild(createRoundCard(round, idx, rounds.length)))
      const footer = document.createElement('div')
      footer.style.cssText = 'display:flex;justify-content:space-between;align-items:center;margin-top:20px;padding-top:14px;border-top:1px solid #cbd5e1;font-size:11px;color:#64748b;'
      footer.innerHTML = `<div style="display:flex;align-items:center;gap:6px;"><span style="font-weight:600;">Q-Solver</span></div><div>${new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })}</div>`
      container.appendChild(footer)
      document.body.appendChild(container)
      const canvas = await html2canvas(container, { backgroundColor: null, scale: 2, useCORS: true, logging: false })
      document.body.removeChild(container)
      const b64 = canvas.toDataURL('image/png')
      await api.saveImageToFile(b64)
    } catch (e) {
      console.error('导出图片失败:', e)
    }
  }

  return {
    history, activeHistoryIndex, selectedRoundTarget, streamTarget, isLoading, isAppending, shouldOverwriteHistory,
    isThinking, thinkingStatusText, thinkingExpanded, isThinkingStalled,
    streamingHtml, errorState, historyEntries, currentRounds, currentHistoryItem, isViewingStreamTarget,
    renderMarkdown, getSummary, getRoundSummary, getRoundsCount, getFullContent, getThinkingPreview,
    selectHistory, selectHistoryEntry, isHistoryEntrySelected,
    handleStreamStart, handleStreamChunk, handleThinkingChunk,
    handleSolution, handleInlineError, clearInlineError,
    activateQuestionBankSession, handleDesktopSolveEvent, loadRoundImage,
    setStreamBuffer, setUserScreenshot, resetLiveHistory, deleteHistory, exportImage,
  }
})
