import assert from 'node:assert/strict'
import fs from 'node:fs'

const helperSource = fs.readFileSync(
  new URL('./src/services/modelBridgeTimeout.js', import.meta.url),
  'utf8',
)
const helperModule = await import(
  `data:text/javascript;base64,${Buffer.from(helperSource).toString('base64')}`
)
const {
  MODEL_BRIDGE_TIMEOUT_MS,
  withModelBridgeTimeout,
} = helperModule

assert.equal(MODEL_BRIDGE_TIMEOUT_MS, 20_000, 'model bridge 默认等待上限必须为 20 秒')

const resolvedValue = { models: ['fixture-model'] }
assert.strictEqual(
  await withModelBridgeTimeout(Promise.resolve(resolvedValue), 'catalog', 50),
  resolvedValue,
  '正常 resolve 必须原样传递结果',
)

const originalError = new Error('synthetic bridge rejection')
let rejectedError
try {
  await withModelBridgeTimeout(Promise.reject(originalError), 'probe', 50)
} catch (error) {
  rejectedError = error
}
assert.strictEqual(rejectedError, originalError, '正常 reject 必须原样传递原因')

for (const operation of ['catalog', 'probe']) {
  await assert.rejects(
    withModelBridgeTimeout(new Promise(() => {}), operation, 5),
    (error) => {
      assert.equal(error.code, 'frontend_wait_timeout')
      assert.equal(error.operation, operation)
      assert.equal(
        error.message,
        '等待后端响应超时；界面已停止等待，后台请求不保证已取消',
      )
      assert.equal(
        Object.hasOwn(error, 'requestSent'),
        false,
        '前端等待超时不得伪造 requestSent=false',
      )
      assert.equal(error.message.includes('后端已取消'), false)
      assert.equal(error.message.includes('零费用'), false)
      return true
    },
  )
}

const apiSource = fs.readFileSync(new URL('./src/services/api.js', import.meta.url), 'utf8')
assert.ok(
  apiSource.includes("import { withModelBridgeTimeout } from './modelBridgeTimeout'"),
  'API wrapper 必须导入 model bridge deadline helper',
)
assert.ok(
  apiSource.includes("listModelsForDraft: (connection) => withModelBridgeTimeout(callApp('ListModelsForDraft', connection), 'catalog')"),
  '模型目录 bridge 必须使用 catalog deadline',
)
assert.ok(
  apiSource.includes("probeModelConnection: (request) => withModelBridgeTimeout(callApp('ProbeModelConnection', request), 'probe')"),
  '模型 probe bridge 必须使用 probe deadline',
)
assert.equal(
  (apiSource.match(/withModelBridgeTimeout\(callApp\(/g) || []).length,
  2,
  '只允许包装模型目录与 probe 两个 bridge 调用',
)

console.log('Model bridge timeout regression tests passed')
