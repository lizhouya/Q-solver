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
                <span>{{ bank.status.operation === 'exporting' ? '正在导出原始记录' : '正在整理并导出' }}</span>
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
                <span class="qb-confirm-text">仅标记清空，原始文件不会物理删除。确认？</span>
                <button class="qb-danger qb-danger-solid" :disabled="bank.busy" @click="discardCurrent">确认清空</button>
                <button :disabled="bank.busy" @click="confirmDiscard = false">取消</button>
              </template>
              <button v-if="bank.status.organizing && bank.status.operation === 'organizing'" :disabled="bank.working" @click="bank.cancelOrganizing">取消整理</button>
            </div>
            <p class="qb-hint">“结束本段”只封存，不导出；答案旁的“导出题库”会先打开确认窗口，导出成功后也会自动封段。</p>
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
                <h3 id="qb-history-title">历史分段</h3>
                <p class="qb-hint">已封存或已导出的分段可重新打开同一导出窗口，不会切换当前可写段。</p>
              </div>
              <button :disabled="bank.busy" @click="bank.loadSessions">刷新</button>
            </div>
            <div v-if="bank.sessions.length" class="qb-session-list">
              <div v-for="session in bank.sessions" :key="session.id" class="qb-session-row">
                <div class="qb-session" :class="{ active: session.id === bank.status.session?.id }">
                  <span>
                    <strong>{{ session.company || session.type || '笔试题库' }}</strong>
                    <small>{{ formatTime(session.startedAt) }}</small>
                  </span>
                  <span class="qb-session-meta">{{ session.count || 0 }} 次提交 · {{ stateName(session.state) }}</span>
                </div>
                <button
                  v-if="['ended', 'exported'].includes(session.state)"
                  class="qb-history-export"
                  :disabled="bank.busy"
                  :aria-label="`重新导出 ${session.company || session.type || '笔试题库'}`"
                  @click="bank.exportSession(session.id)"
                >重新导出…</button>
              </div>
            </div>
            <p v-else class="qb-empty">暂无历史分段。</p>
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
.qb-session-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .5rem; }
.qb-session { min-width: 0; display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: .55rem .8rem; border: 1px solid var(--border-default); border-radius: var(--radius-md); background: var(--surface-card); }
.qb-history-export { white-space: nowrap; }
.qb-session.active { border-color: var(--accent); background: var(--accent-muted); }
.qb-session > span:first-child { min-width: 0; display: flex; flex-direction: column; gap: .2rem; }
.qb-session strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.qb-session small, .qb-session-meta { color: var(--text-muted); font-size: .72rem; }
.qb-session-meta { flex-shrink: 0; }
.qb-empty { padding: 1rem; text-align: center; color: var(--text-muted); font-size: .8rem; }
@media (max-width: 560px) {
  .qb-section-heading, .qb-session, .qb-inline-error { align-items: stretch; flex-direction: column; }
  .qb-session-row { grid-template-columns: 1fr; }
  .qb-history-export { justify-self: start; }
  .qb-state { align-self: flex-start; }
  .qb-session-meta { align-self: flex-start; }
}
</style>
