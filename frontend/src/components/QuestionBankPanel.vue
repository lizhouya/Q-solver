<template>
  <Teleport to="body">
    <div v-if="bank.visible" class="qb-overlay" @keydown.esc="bank.close">
      <section ref="panel" class="qb-panel" role="dialog" aria-modal="true" aria-labelledby="qb-title" tabindex="-1">
        <header>
          <div>
            <h2 id="qb-title">笔试题库</h2>
            <p>第一次解题自动建立当前段；结束或导出后，下一次解题自动建立新段。</p>
          </div>
          <button class="qb-close" aria-label="关闭题库面板" @click="bank.close"><Icon name="x" :size="18" /></button>
        </header>

        <div class="qb-body">
          <div v-if="!bank.configured" class="qb-notice qb-warning">
            尚未配置默认题库目录。请在主设置的“题库”标签中选择目录。
          </div>

          <section class="qb-section" aria-labelledby="qb-current-title">
            <div class="qb-section-heading">
              <div>
                <h3 id="qb-current-title">当前 / 上一分段</h3>
                <p v-if="bank.status.session" class="qb-hint">
                  {{ bank.status.session.company || bank.status.session.type || '未指定单位' }}
                  <template v-if="bank.status.session.startedAt"> · {{ formatTime(bank.status.session.startedAt) }}</template>
                </p>
                <p v-else class="qb-hint">下一次解题时自动建立，无需手动开始。</p>
              </div>
              <div class="qb-state">
                <strong>{{ bank.status.session?.count || 0 }}</strong>
                <span>次提交 · {{ stateLabel }}</span>
              </div>
            </div>

            <div v-if="bank.status.unsaved" class="qb-inline-error">
              <span>当前段有内容尚未成功落盘。</span>
              <button :disabled="bank.working" @click="bank.retrySave">重试保存</button>
            </div>

            <div v-if="bank.status.organizing" class="qb-progress" role="status">
              <div class="qb-progress-copy">
                <span>{{ bank.status.operation === 'exporting' ? '正在导出原始记录' : '正在整理并生成建议' }}</span>
                <strong v-if="bank.status.operation !== 'exporting'">{{ bank.status.progress?.done || 0 }} / {{ bank.status.progress?.total || 0 }}</strong>
              </div>
              <div v-if="bank.status.operation !== 'exporting'" class="qb-progress-track"><span :style="{ width: progressWidth }"></span></div>
              <p v-if="bank.status.progress?.failed" class="qb-hint">失败 {{ bank.status.progress.failed }} 条；原始记录仍会保留。</p>
            </div>

            <div class="qb-actions">
              <button
                v-if="bank.canEndCurrent"
                class="qb-primary"
                :disabled="bank.busy"
                title="只封存当前段；下一次解题会自动建立新段"
                @click="bank.splitCurrent"
              >结束本段</button>
              <button v-if="!confirmDiscard" class="qb-danger" :disabled="!bank.canDiscardCurrent || bank.busy" @click="confirmDiscard = true">清空本段</button>
              <template v-else>
                <span class="qb-confirm-text">将永久删除当前分段的题库记录、答案和截图文件，无法恢复。已导出的文件不受影响。确认？</span>
                <button class="qb-danger qb-danger-solid" :disabled="bank.busy" @click="discardCurrent">永久删除</button>
                <button :disabled="bank.busy" @click="confirmDiscard = false">取消</button>
              </template>
              <button v-if="bank.status.organizing && bank.status.operation === 'organizing'" :disabled="bank.working" @click="bank.cancelOrganizing">取消整理</button>
            </div>
            <p class="qb-hint">“结束本段”会保留本地记录并清理解题界面，不导出；答案旁的“导出题库”会先打开确认窗口，导出成功后也会自动封段。</p>
          </section>

          <section v-if="bank.status.lastExport?.directory" class="qb-result" aria-labelledby="qb-result-title">
            <strong id="qb-result-title">最近导出结果</strong>
            <code class="qb-path">{{ bank.status.lastExport.directory }}</code>
            <p>
              <template v-if="bank.status.lastExport.records !== undefined">{{ bank.status.lastExport.records }} 次原始提交</template>
              <template v-if="bank.status.lastExport.questions !== undefined"> · {{ bank.status.lastExport.questions }} 道整理题目</template>
            </p>
          </section>

          <p v-if="bank.error" class="qb-error" role="alert">{{ bank.error }}</p>
          <p v-if="bank.status.message" class="qb-message" role="status">{{ bank.status.message }}</p>

          <section class="qb-section qb-history" aria-labelledby="qb-history-title">
            <div class="qb-section-heading">
              <div>
                <h3 id="qb-history-title">最近 {{ bank.sessionLimit }} 个分段</h3>
                <p class="qb-hint">点击分段可查看归档；“继续此分段”会把它恢复到截图解题主界面，并暂停当前活动段。</p>
              </div>
              <button :disabled="bank.busy" @click="bank.loadSessions">刷新</button>
            </div>
            <p v-if="bank.sessionHidden" class="qb-hidden-note">
              另有 {{ bank.sessionHidden }} 个更早分段仅保留在题库文件夹中；不会自动导出、整理或删除。
            </p>
            <div v-if="bank.sessions.length" class="qb-session-list">
              <div v-for="session in bank.sessions" :key="session.id" class="qb-session-row">
                <button
                  type="button"
                  class="qb-session"
                  :class="{ active: session.id === bank.status.session?.id, selected: session.id === bank.selectedSessionId }"
                  :aria-pressed="session.id === bank.selectedSessionId"
                  @click="browseSession(session.id)"
                >
                  <span>
                    <strong>{{ sessionTitle(session) }}</strong>
                    <small>{{ formatTime(session.startedAt) }}</small>
                  </span>
                  <span class="qb-session-meta">
                    <em v-if="session.id === bank.status.session?.id">后台当前段</em>
                    <em v-if="session.id === bank.selectedSessionId && bank.detailLoading">正在读取</em>
                    <em v-else-if="session.id === bank.sessionDetail?.id">正在浏览</em>
                    <em v-else-if="session.id === bank.selectedSessionId && bank.detailError">读取失败</em>
                    {{ session.count || 0 }} 次提交 · {{ stateName(session.state) }}
                  </span>
                </button>
                <div class="qb-session-actions">
                  <button
                    v-if="['ended', 'exported', 'paused'].includes(session.state)"
                    class="qb-history-continue"
                    :disabled="bank.busy || Boolean(bank.continueSessionWorking)"
                    :aria-label="`继续 ${sessionTitle(session)}`"
                    @click.stop="bank.continueSession(session.id)"
                  >{{ bank.continueSessionWorking === session.id ? '切换中…' : '继续此分段' }}</button>
                  <button
                    v-if="Number(session.count || 0) > 0 && ['ended', 'exported'].includes(session.state)"
                    class="qb-history-export"
                    :disabled="bank.busy || Boolean(bank.continueSessionWorking)"
                    :aria-label="`重新导出 ${sessionTitle(session)}`"
                    @click.stop="bank.exportSession(session.id)"
                  >重新导出…</button>
                </div>
              </div>
            </div>
            <p v-else class="qb-empty">暂无历史分段。</p>
          </section>

          <section v-if="bank.selectedSessionId" ref="detailPanel" class="qb-section qb-detail" aria-labelledby="qb-detail-title" tabindex="-1">
            <div class="qb-section-heading">
              <div>
                <h3 id="qb-detail-title">分段详情</h3>
                <p class="qb-hint">只读归档视图，与左侧实时解题历史相互独立。</p>
              </div>
              <button type="button" @click="bank.clearSessionDetail">关闭详情</button>
            </div>
            <p v-if="bank.detailLoading" class="qb-empty" role="status">正在读取完整记录…</p>
            <p v-else-if="bank.detailError" class="qb-error" role="alert">{{ bank.detailError }}</p>
            <template v-else-if="bank.sessionDetail">
              <div class="qb-detail-meta">
                <strong>{{ sessionTitle(bank.sessionDetail) }}</strong>
                <span>{{ stateName(bank.sessionDetail.state) }} · {{ bank.sessionDetail.count }} 次提交</span>
                <span v-if="bank.sessionDetail.company">单位：{{ bank.sessionDetail.company }}</span>
                <span v-if="bank.sessionDetail.issuer">出题方：{{ bank.sessionDetail.issuer }}</span>
                <span v-if="bank.sessionDetail.type">题型：{{ bank.sessionDetail.type }}</span>
                <span v-if="bank.sessionDetail.state === 'discarded'">已清空，仅保留本地历史；不可恢复、整理或导出。</span>
              </div>
              <p v-if="!bank.sessionDetail.records?.length" class="qb-empty qb-archive-empty">
                该分段没有提交记录，因此没有可显示的题目或答案。分段文件仍保留在题库目录中。
              </p>
              <article v-for="(record, recordIndex) in bank.sessionDetail.records" :key="record.id" class="qb-record">
                <header class="qb-record-header">
                  <strong>提交 {{ recordIndex + 1 }}</strong>
                  <span>{{ stateName(record.status) }} · {{ formatTime(record.createdAt) }}</span>
                </header>
                <p class="qb-record-context">模型：{{ record.model || '未知' }} · 场景：{{ record.scene || '未知' }}</p>
                <div v-if="record.images?.length" class="qb-record-images">
                  <div v-for="image in record.images" :key="image.index" class="qb-record-image">
                    <img
                      v-if="imageState(record.id, image.index).data"
                      :src="imageState(record.id, image.index).data"
                      :alt="`提交 ${recordIndex + 1} 截图 ${image.index + 1}`"
                    />
                    <button
                      v-else
                      type="button"
                      :disabled="imageState(record.id, image.index).loading"
                      @click="bank.loadRecordImage(record.id, image.index)"
                    >{{ imageState(record.id, image.index).loading ? '读取中…' : `查看截图 ${image.index + 1}` }}</button>
                    <small v-if="imageState(record.id, image.index).error">{{ imageState(record.id, image.index).error }}</small>
                  </div>
                </div>
                <h4>完整原答案</h4>
                <pre>{{ record.answer || '（无答案）' }}</pre>
                <p v-if="record.organizeError" class="qb-inline-note">整理状态：{{ record.organizeError }}</p>
                <div v-if="record.metadataSuggestion" class="qb-suggestion-inline">
                  <strong>模型元数据建议（待核对）</strong>
                  <span>单位：{{ record.metadataSuggestion.company || '未识别' }}</span>
                  <span>出题方：{{ record.metadataSuggestion.issuer || '未识别' }}</span>
                  <span>题型：{{ record.metadataSuggestion.type || '未识别' }}</span>
                  <small>{{ record.metadataSuggestion.reviewNote }}</small>
                </div>
                <div v-if="record.questions?.length" class="qb-questions">
                  <h4>已有整理结果（待核对）</h4>
                  <section v-for="(question, questionIndex) in record.questions" :key="questionIndex" class="qb-question">
                    <strong>{{ question.title }}</strong>
                    <p>{{ question.stem }}</p>
                    <p>分类：{{ question.category }} / {{ question.type }}</p>
                    <p>参考答案：{{ question.referenceAnswer || '待核对' }}</p>
                    <p v-if="question.knowledgePoints?.length">知识点：{{ question.knowledgePoints.join('、') }}</p>
                    <p v-if="question.keyPoints?.length">解析要点：{{ question.keyPoints.join('；') }}</p>
                    <small>{{ question.reviewNote }}</small>
                  </section>
                </div>
              </article>
            </template>
          </section>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { useQuestionBankStore } from '../stores/questionbank'

const bank = useQuestionBankStore()
const panel = ref(null)
const detailPanel = ref(null)
const confirmDiscard = ref(false)

const stateNames = {
  active: '记录中',
  collecting: '记录中',
  paused: '待恢复',
  ended: '已结束',
  sealed: '已结束',
  exported: '已导出',
  discarded: '已清空',
}

const stateName = (state) => stateNames[state] || state || '待记录'
const stateLabel = computed(() => bank.status.unsaved ? '未保存' : stateName(bank.status.session?.state))
const progressWidth = computed(() => {
  const total = Number(bank.status.progress?.total || 0)
  const done = Number(bank.status.progress?.done || 0)
  return total > 0 ? `${Math.min(100, Math.round((done / total) * 100))}%` : '0%'
})
const formatTime = (value) => value ? new Date(value).toLocaleString() : '时间未知'
const sessionTitle = (session) => session?.name || [session?.company, session?.issuer, session?.type].filter(Boolean).join('_') || '笔试题库'
const imageState = (recordId, index) => bank.imageCache[`${recordId}:${index}`] || { loading: false, data: '', error: '' }

async function browseSession(id) {
  const selection = bank.selectSession(id)
  await nextTick()
  if (bank.selectedSessionId === id) {
    detailPanel.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    detailPanel.value?.focus({ preventScroll: true })
  }
  return selection
}

async function discardCurrent() {
  const success = await bank.discardCurrent(true)
  if (success) confirmDiscard.value = false
}

watch(() => bank.status.session?.id, () => { confirmDiscard.value = false })
watch(() => bank.visible, async (visible) => {
  if (visible) {
    confirmDiscard.value = false
    await nextTick()
    panel.value?.focus()
  }
})
</script>

<style scoped>
.qb-overlay { position: fixed; inset: 0; z-index: 1100; background: var(--surface-overlay, rgba(0,0,0,.5)); display: flex; align-items: center; justify-content: center; padding: 1rem; pointer-events: auto; }
.qb-panel { width: min(700px, 100%); max-height: 88vh; display: flex; flex-direction: column; background: var(--surface-elevated); color: var(--text-primary); border: 1px solid var(--border-default); border-radius: var(--radius-lg); box-shadow: var(--shadow-xl); overflow: hidden; outline: none; }
header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; padding: 1.2rem; border-bottom: 1px solid var(--border-default); }
h2, h3 { margin: 0; }
h2 { font-size: 1.15rem; margin-bottom: .35rem; }
h3 { font-size: .95rem; }
p { margin: .4rem 0; line-height: 1.6; }
header p, .qb-hint { font-size: .8rem; color: var(--text-secondary); }
.qb-body { padding: 0 1.2rem 1rem; overflow-y: auto; min-height: 0; }
.qb-notice, .qb-result { padding: .75rem; background: var(--surface-card); border: 1px solid var(--border-default); border-radius: var(--radius-md); font-size: .8rem; margin: 1rem 0; }
.qb-result strong { display: block; }
.qb-section { padding: 1rem 0; border-bottom: 1px solid var(--border-default); }
.qb-section-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
.qb-state { flex-shrink: 0; display: flex; align-items: baseline; gap: .35rem; padding: .45rem .65rem; border-radius: var(--radius-md); background: var(--surface-card); color: var(--text-secondary); font-size: .75rem; }
.qb-state strong { color: var(--accent); font-size: 1.05rem; }
.qb-actions { display: flex; align-items: center; gap: .55rem; flex-wrap: wrap; margin-top: .85rem; }
button { background: var(--surface-card); color: var(--text-primary); border: 1px solid var(--border-default); padding: .55rem .8rem; border-radius: var(--radius-md); cursor: pointer; font-size: .8rem; }
button:hover:not(:disabled) { background: var(--surface-card-hover); }
button:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
button:disabled { opacity: .5; cursor: not-allowed; }
button.qb-primary { background: var(--accent); color: white; border-color: var(--accent); }
button.qb-danger { color: var(--color-error); }
button.qb-danger-solid { background: var(--error-bg); border-color: var(--error-border); }
.qb-close { display: flex; padding: .3rem; }
.qb-path { display: block; overflow-wrap: anywhere; white-space: pre-wrap; font-size: .78rem; margin-top: .4rem; }
.qb-warning { border-color: var(--warning-border); color: var(--color-warning); background: var(--warning-bg); }
.qb-inline-error, .qb-error, .qb-message { padding: .7rem; margin-top: .8rem; border-radius: var(--radius-md); font-size: .8rem; overflow-wrap: anywhere; }
.qb-inline-error { display: flex; align-items: center; justify-content: space-between; gap: .75rem; background: var(--error-bg); border: 1px solid var(--error-border); color: var(--color-error); }
.qb-error { background: var(--error-bg); border: 1px solid var(--error-border); color: var(--color-error); }
.qb-message { background: var(--surface-card); border: 1px solid var(--border-default); color: var(--text-secondary); }
.qb-progress { margin-top: .8rem; padding: .75rem; border: 1px solid var(--accent-border); border-radius: var(--radius-md); background: var(--accent-muted); }
.qb-progress-copy { display: flex; justify-content: space-between; gap: 1rem; font-size: .8rem; }
.qb-progress-track { height: 5px; overflow: hidden; margin-top: .55rem; border-radius: var(--radius-full); background: var(--border-default); }
.qb-progress-track span { display: block; height: 100%; border-radius: inherit; background: var(--accent); transition: width .2s ease; }
.qb-confirm-text { font-size: .78rem; color: var(--color-warning); }
.qb-history { border-bottom: 0; }
.qb-session-list { display: flex; flex-direction: column; gap: .5rem; margin-top: .75rem; }
.qb-session-row { display: grid; grid-template-columns: minmax(0, 1fr) 7.5rem; align-items: stretch; gap: .5rem; }
.qb-session-actions { min-width: 0; display: flex; flex-direction: column; align-items: stretch; justify-content: center; gap: .4rem; }
.qb-session-actions button { width: 100%; }
.qb-session { min-width: 0; display: flex; flex-direction: column; align-items: stretch; justify-content: center; gap: .35rem; padding: .55rem .8rem; border: 1px solid var(--border-default); border-radius: var(--radius-md); background: var(--surface-card); }
.qb-history-export, .qb-history-continue { white-space: nowrap; }
.qb-history-continue { color: white; background: var(--accent); border-color: var(--accent); }
.qb-session.active { border-color: var(--accent); background: var(--accent-muted); }
.qb-session > span:first-child { width: 100%; min-width: 0; display: flex; flex-direction: column; gap: .2rem; }
.qb-session strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.qb-session small, .qb-session-meta { color: var(--text-muted); font-size: .72rem; }
.qb-session-meta { max-width: 100%; align-self: flex-start; overflow-wrap: anywhere; }
.qb-empty { padding: 1rem; text-align: center; color: var(--text-muted); font-size: .8rem; }
.qb-hidden-note { padding: .6rem .7rem; border-radius: var(--radius-md); background: var(--warning-bg); color: var(--color-warning); font-size: .75rem; }
.qb-session { width: 100%; text-align: left; }
.qb-session.selected { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-muted); }
.qb-session-meta em { display: inline-block; margin-right: .3rem; padding: .08rem .3rem; border-radius: var(--radius-full); background: var(--accent-muted); color: var(--accent); font-style: normal; }
.qb-detail { border-top: 1px solid var(--border-default); scroll-margin-top: 1rem; outline: none; }
.qb-archive-empty { margin-top: .75rem; border: 1px dashed var(--border-default); border-radius: var(--radius-md); }
.qb-detail-meta { display: flex; flex-wrap: wrap; gap: .4rem .8rem; margin-top: .75rem; padding: .75rem; border: 1px solid var(--border-default); border-radius: var(--radius-md); background: var(--surface-card); font-size: .76rem; }
.qb-detail-meta strong { width: 100%; font-size: .9rem; }
.qb-record { margin-top: .75rem; padding: .8rem; border: 1px solid var(--border-default); border-radius: var(--radius-md); background: var(--surface-card); }
.qb-record-header { padding: 0 0 .55rem; border: 0; border-bottom: 1px solid var(--border-subtle); font-size: .77rem; }
.qb-record-header span, .qb-record-context { color: var(--text-muted); font-size: .72rem; }
.qb-record h4 { margin: .8rem 0 .4rem; font-size: .78rem; }
.qb-record pre { max-width: 100%; max-height: 22rem; margin: 0; padding: .7rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; border-radius: var(--radius-sm); background: var(--surface-input); color: var(--text-primary); font: .75rem/1.6 ui-monospace, monospace; }
.qb-record-images { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: .55rem; margin-top: .7rem; }
.qb-record-image { display: flex; flex-direction: column; gap: .3rem; align-items: flex-start; }
.qb-record-image img { width: 100%; max-height: 260px; object-fit: contain; border-radius: var(--radius-sm); background: var(--surface-input); }
.qb-record-image small { color: var(--color-error); overflow-wrap: anywhere; }
.qb-inline-note, .qb-suggestion-inline { margin-top: .65rem; padding: .55rem; border-radius: var(--radius-sm); background: var(--warning-bg); color: var(--color-warning); font-size: .72rem; }
.qb-suggestion-inline { display: flex; flex-wrap: wrap; gap: .25rem .7rem; }
.qb-suggestion-inline strong, .qb-suggestion-inline small { width: 100%; }
.qb-question { margin-top: .5rem; padding: .6rem; border-left: 3px solid var(--accent); background: var(--surface-input); font-size: .74rem; }
.qb-question p { margin: .3rem 0; white-space: pre-wrap; overflow-wrap: anywhere; }
.qb-question small { color: var(--text-muted); }
@media (max-width: 560px) {
  .qb-section-heading, .qb-inline-error { align-items: stretch; flex-direction: column; }
  .qb-session-row { grid-template-columns: 1fr; }
  .qb-session-actions { width: 100%; }
  .qb-state { align-self: flex-start; }
}
</style>
