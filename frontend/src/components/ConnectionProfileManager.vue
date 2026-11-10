<template>
  <section class="connection-profile-manager" aria-label="连接记录管理">
    <div class="profile-heading-row">
      <label for="connection-profile-select">已配置连接</label>
      <button
        type="button"
        class="btn-secondary profile-new-button"
        :disabled="loading || formalProfileCount >= 20"
        @click="emit('new')"
      >新建连接</button>
    </div>

    <select
      id="connection-profile-select"
      class="profile-select"
      :value="selectedId"
      :disabled="loading || !profiles.length"
      @change="emit('select', $event.target.value)"
    >
      <option v-if="selectedId === ''" value="">新连接（尚未保存）</option>
      <option v-for="profile in profiles" :key="profile.id" :value="profile.id">
        {{ profileOptionText(profile) }}
      </option>
    </select>

    <div class="profile-name-row">
      <div class="profile-name-field">
        <label for="connection-profile-name">记录名称</label>
        <input
          id="connection-profile-name"
          :value="profileName"
          type="text"
          maxlength="64"
          autocomplete="off"
          :disabled="loading || !loaded"
          :placeholder="automaticNamePreview"
          @input="emit('update:profileName', $event.target.value)"
        />
        <p v-if="profileName === ''" class="profile-hint">保存后由后端确认名称：{{ automaticNamePreview }}</p>
      </div>
      <button
        type="button"
        class="btn-danger profile-delete-button"
        :disabled="deleteDisabled"
        :title="deleteDisabledReason"
        @click="emit('request-delete', selectedId)"
      >删除连接</button>
    </div>

    <p v-if="formalProfileCount >= 20" class="profile-hint warning" role="status">已达到 20 条连接记录上限</p>

    <div v-if="pendingDeleteId === selectedId && !deleteDisabled" class="delete-confirmation" role="alert">
      <p>确认删除“{{ selectedProfile?.name }}”？此操作不会调用模型接口。</p>
      <div class="delete-confirmation-actions">
        <button type="button" class="btn-danger" @click="emit('confirm-delete')">确认删除</button>
        <button type="button" class="btn-secondary" @click="emit('cancel-delete')">取消</button>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  profiles: { type: Array, default: () => [] },
  selectedId: { type: String, default: '' },
  activeId: { type: String, default: '' },
  profileName: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  loaded: { type: Boolean, default: false },
  pendingDeleteId: { type: String, default: '' },
  automaticNamePreview: { type: String, default: '未命名连接' },
})

const emit = defineEmits([
  'select', 'new', 'update:profileName',
  'request-delete', 'cancel-delete', 'confirm-delete',
])

const formalProfileCount = computed(() => props.profiles.filter(profile => profile.id !== 'legacy-current').length)
const selectedProfile = computed(() => props.profiles.find(profile => profile.id === props.selectedId) || null)
const deleteDisabled = computed(() => (
  props.loading || !props.loaded || !selectedProfile.value ||
  props.selectedId === props.activeId || props.selectedId === 'legacy-current'
))
const deleteDisabledReason = computed(() => {
  if (props.selectedId === 'legacy-current') return '旧连接保存为正式记录前不能删除'
  if (props.selectedId && props.selectedId === props.activeId) return '活动连接不能直接删除，请先保存激活另一条记录'
  return ''
})

function profileOptionText(profile) {
  const protocol = profile.apiProtocol === 'custom_json' ? '自定义模板' : 'OpenAI Chat'
  const address = profile.addressLabel || '地址未设置'
  const model = profile.model || '模型未设置'
  const key = profile.apiKeySet ? 'Key SET' : 'Key NOT SET'
  const active = profile.active ? ' · 活动' : ''
  return `${profile.name} · ${protocol} · ${address} · ${model} · ${key}${active}`
}
</script>

<style scoped>
.connection-profile-manager {
  display: grid;
  gap: var(--sp-3);
  padding: var(--sp-4);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--surface-card);
}
.profile-heading-row,
.profile-name-row,
.delete-confirmation-actions {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  flex-wrap: wrap;
}
.profile-heading-row { justify-content: space-between; }
.profile-heading-row label,
.profile-name-field label { color: var(--text-secondary); font-size: var(--text-sm); font-weight: 600; }
.profile-select,
.profile-name-field input {
  width: 100%;
  min-width: 0;
  min-height: 42px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-elevated);
  color: var(--text-primary);
  padding: 0 var(--sp-3);
}
.profile-name-field { flex: 1 1 280px; min-width: 0; display: grid; gap: var(--sp-2); }
.profile-delete-button { flex: 0 0 auto; align-self: end; }
.profile-hint { margin: 0; color: var(--text-tertiary); font-size: var(--text-xs); overflow-wrap: anywhere; }
.profile-hint.warning { color: var(--warning); }
.btn-danger {
  min-height: 38px;
  border: 1px solid var(--danger);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--danger);
  padding: 0 var(--sp-3);
  cursor: pointer;
}
.btn-danger:disabled { cursor: not-allowed; opacity: 0.45; }
.delete-confirmation { display: grid; gap: var(--sp-3); padding: var(--sp-3); border: 1px solid var(--danger); border-radius: var(--radius-md); }
.delete-confirmation p { margin: 0; color: var(--text-secondary); overflow-wrap: anywhere; }
@media (max-width: 560px) {
  .profile-name-row { align-items: stretch; }
  .profile-delete-button { width: 100%; }
}
</style>
