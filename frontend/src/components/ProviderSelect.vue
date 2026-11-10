<template>
  <div class="provider-page">
    <section class="form-card">
      <div class="form-item">
        <label class="field-label"><span>API 模式</span><span class="field-meta">Protocol</span></label>
        <select class="control" :value="apiProtocol" @change="changeProtocol($event.target.value)">
          <option value="openai_chat">OpenAI 兼容 Chat Completions</option>
          <option value="custom_json">自定义请求模板</option>
        </select>
        <p class="field-hint">兼容模式支持流式输出；模板模式等待完整 JSON 响应后显示答案。</p>
      </div>

      <div class="form-item">
        <label class="field-label" for="connection-url"><span>请求地址</span></label>
        <input id="connection-url" class="control mono" type="text" :value="baseURL" @input="emit('update:baseURL', $event.target.value)" placeholder="https://api.example.com/v1" />
        <p class="field-hint">兼容模式填写 API root 或完整 /chat/completions 地址；模板模式填写完整推理 URL。</p>
      </div>

      <div class="form-item">
        <label class="field-label" for="connection-key"><span>API Key（可选）</span></label>
        <input id="connection-key" class="control" type="password" autocomplete="off" :value="apiKey" @input="emit('update:apiKey', $event.target.value)" placeholder="本地无鉴权接口可留空" />
      </div>

      <details class="advanced-card">
        <summary>高级设置</summary>
        <div v-if="apiProtocol === 'openai_chat'" class="advanced-content">
      <div class="form-item">
        <label class="field-label"><span>快捷预设</span><span class="field-meta">Preset</span></label>
        <select class="control" @change="selectPreset($event.target.value)">
          <option value="">选择后仅填充草稿，不会自动联网</option>
          <option v-for="preset in connectionPresets" :key="preset.id" :value="preset.id">{{ preset.label }}</option>
        </select>
        <p class="field-hint">CCR 本地网关只是 OpenAI Chat 兼容预设，不再是独立连接模式。</p>
      </div>

      <div class="form-item">
        <label class="field-label"><span>Endpoint Path</span><span class="field-meta">Inference</span></label>
        <input class="control mono" type="text" :value="endpointPath" @input="emit('update:endpointPath', $event.target.value)" :placeholder="defaultEndpoint" />
        <p class="field-hint">只接受以 / 开头的相对路径；模型目录仍使用标准 /models。</p>
      </div>

      <div v-if="apiProtocol === 'openai_chat'" class="form-item">
        <label class="field-label"><span>Response Format</span><span class="field-meta">Chat only</span></label>
        <select class="control" :value="responseFormat" @change="emit('update:responseFormat', $event.target.value)">
          <option value="text">text</option>
          <option value="json_object">json_object</option>
        </select>
        <p class="field-hint">json_object 会要求兼容端返回 JSON；其他协议固定使用 text。</p>
      </div>

        </div>
        <div v-if="apiProtocol === 'custom_json'" class="form-item advanced-content">
          <label class="field-label" for="requestTemplate">请求模板 JSON</label>
          <textarea id="requestTemplate" class="control mono" :value="requestTemplate" @input="emit('update:requestTemplate', $event.target.value)" spellcheck="false" rows="7"></textarea>
          <label class="field-label" for="responseJSONPointer">响应 JSON Pointer</label>
          <input id="responseJSONPointer" class="control mono" :value="responseJSONPointer" @input="emit('update:responseJSONPointer', $event.target.value)" />
          <label class="field-label" for="authHeader">鉴权 Header</label>
          <input id="authHeader" class="control mono" :value="authHeader" @input="emit('update:authHeader', $event.target.value)" />
          <label class="field-label" for="authPrefix">鉴权前缀（保留末尾空格；x-api-key 通常留空）</label>
          <input id="authPrefix" class="control mono" :value="authPrefix" @input="emit('update:authPrefix', $event.target.value)" />
          <p class="field-hint" v-pre>占位符必须独立作为 JSON 值：{{model}}、{{openai_messages}}、{{responses_input}}、{{anthropic_messages}}、{{anthropic_system}}、{{system_prompt}}、{{user_text}}、{{images_data_urls}}、{{images_base64}}。不执行脚本；默认响应路径 /choices/0/message/content，非标准服务须按其文档修改。密钥仅进入鉴权 Header。</p>
        </div>

        <div class="form-item advanced-content">
          <label class="field-label"><span>额外 Headers JSON</span><span class="field-meta">Encrypted at rest</span></label>
          <textarea class="control headers-editor mono" :value="extraHeadersText" @input="emit('update:extraHeadersText', $event.target.value)" spellcheck="false" placeholder='{"X-Route":"route-a"}'></textarea>
          <p class="field-hint">值必须是字符串。禁止 Host、Content-Length 等传输控制 Header；敏感值不会回显在保存结果中。</p>
        </div>
      </details>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  apiKey: { type: String, default: '' },
  baseURL: { type: String, default: '' },
  apiProtocol: { type: String, default: 'openai_chat' },
  endpointPath: { type: String, default: '/chat/completions' },
  responseFormat: { type: String, default: 'text' },
  extraHeadersText: { type: String, default: '{}' },
  requestTemplate: { type: String, default: '' },
  responseJSONPointer: { type: String, default: '' },
  authHeader: { type: String, default: '' },
  authPrefix: { type: String, default: '' },
  defaultCustomTemplate: { type: String, required: true },
})

const emit = defineEmits([
  'update:apiKey', 'update:baseURL', 'update:apiProtocol', 'update:endpointPath',
  'update:responseFormat', 'update:extraHeadersText', 'select-preset',
  'update:requestTemplate', 'update:responseJSONPointer', 'update:authHeader', 'update:authPrefix',
])

const connectionPresets = [
  { id: 'openai', label: 'OpenAI 官方', baseURL: 'https://api.openai.com/v1', apiProtocol: 'openai_chat', endpointPath: '/chat/completions', responseFormat: 'text' },
  { id: 'ccr', label: 'CCR 本地网关', baseURL: 'http://127.0.0.1:3456/v1', apiProtocol: 'openai_chat', endpointPath: '/chat/completions', responseFormat: 'text' },
]

const defaultEndpoint = computed(() => {
  if (props.apiProtocol === 'openai_responses') return '/responses'
  if (props.apiProtocol === 'anthropic_messages') return '/v1/messages'
  return '/chat/completions'
})
const isReady = computed(() => Boolean(props.baseURL.trim()))

function selectPreset(id) {
  const preset = connectionPresets.find(item => item.id === id)
  if (preset) emit('select-preset', { ...preset })
}

function changeProtocol(value) {
  emit('update:apiProtocol', value)
  if (value === 'custom_json') {
    if (!props.requestTemplate) emit('update:requestTemplate', props.defaultCustomTemplate)
    if (!props.responseJSONPointer) emit('update:responseJSONPointer', '/choices/0/message/content')
    if (!props.authHeader) {
      emit('update:authHeader', 'Authorization')
      emit('update:authPrefix', 'Bearer ')
    }
  }
}
</script>

<style scoped>
.provider-page { min-width: 0; display: grid; gap: var(--sp-4); }
.eyebrow { color: var(--accent); font: var(--weight-bold) var(--text-xs)/1 var(--font-mono); letter-spacing: .08em; }
h3 { margin: var(--sp-2) 0; color: var(--text-primary); font-size: var(--text-xl); }
.ready-badge { flex: none; padding: 5px 9px; border: 1px solid var(--warning-border); border-radius: var(--radius-full); background: var(--warning-bg); color: var(--color-warning); font-size: var(--text-xs); }
.ready-badge.ready { border-color: var(--success-border); background: var(--success-bg); color: var(--color-success); }
.form-card { min-width: 0; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); display: grid; gap: var(--sp-4); padding: var(--sp-4); }
.form-item { min-width: 0; display: grid; gap: var(--sp-2); }
.field-label { display: flex; justify-content: space-between; gap: var(--sp-3); color: var(--text-secondary); font-size: var(--text-sm); font-weight: var(--weight-semibold); }
.field-meta { color: var(--text-tertiary); font-family: var(--font-mono); font-size: var(--text-xs); font-weight: var(--weight-normal); }
.control { min-width: 0; width: 100%; min-height: 44px; box-sizing: border-box; padding: 0 var(--sp-3); border: 1px solid var(--border-default); border-radius: var(--radius-md); outline: none; background: var(--surface-input); color: var(--text-primary); font-size: var(--text-sm); }
.control:focus { border-color: var(--accent); box-shadow: var(--shadow-glow); }
.mono { font-family: var(--font-mono); }
.field-hint { margin: 0; color: var(--text-muted); font-size: var(--text-xs); line-height: 1.55; overflow-wrap: anywhere; }
.advanced-card { min-width: 0; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); background: var(--surface-card); }
.advanced-card summary { padding: var(--sp-3); color: var(--text-secondary); font-size: var(--text-sm); font-weight: var(--weight-semibold); cursor: pointer; }
.advanced-content { display: grid; gap: var(--sp-3); min-width: 0; padding: 0 var(--sp-3) var(--sp-3); }
.headers-editor { min-height: 112px; padding-block: var(--sp-3); resize: vertical; line-height: 1.5; }
textarea.control { padding-block: var(--sp-3); resize: vertical; }
</style>
