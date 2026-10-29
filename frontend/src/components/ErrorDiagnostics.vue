<template>
  <details class="error-diagnostics">
    <summary>诊断详情</summary>
    <p v-if="!detail" class="diagnostic-note">暂无信息：此错误未提供结构化诊断。</p>
    <dl v-else>
      <dt>错误类别</dt><dd>{{ detail.code === 'unknown' ? '未知（unknown）' : detail.code }}</dd>
      <dt>失败阶段</dt><dd>{{ detail.stage === 'unknown' ? '未知' : detail.stage }}</dd>
      <dt>HTTP 状态码</dt><dd>{{ detail.httpStatus ?? '未知' }}</dd>
      <dt>实际请求协议</dt><dd>{{ detail.protocol === 'unknown' ? '未知' : detail.protocol }}</dd>
      <dt>是否建议重试</dt><dd>{{ detail.retryable === true ? '可手动重试' : detail.retryable === false ? '不建议直接重试，请先检查配置或请求' : '未知，请先定位原因' }}</dd>
    </dl>
    <p class="diagnostic-note">仅反映当前调用边界；网关错误不等于最终上游原因。不会自动重试，重试可能产生费用。详情未写入题库，重启后不保留。</p>
  </details>
</template>

<script setup>
defineProps({ detail: { type: Object, default: null } })
</script>

<style scoped>
.error-diagnostics { margin: var(--sp-3) 0; max-width: 100%; text-align: left; font-size: var(--text-xs); color: var(--text-secondary); }
summary { cursor: pointer; color: var(--text-primary); }
dl { display: grid; grid-template-columns: minmax(6rem, auto) minmax(0, 1fr); gap: var(--sp-2); }
dt { color: var(--text-tertiary); }
dd { margin: 0; overflow-wrap: anywhere; }
.diagnostic-note { line-height: 1.6; overflow-wrap: anywhere; }
</style>
