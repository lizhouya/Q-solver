export const MODEL_BRIDGE_TIMEOUT_MS = 20_000

const MODEL_BRIDGE_TIMEOUT_MESSAGE = '等待后端响应超时；界面已停止等待，后台请求不保证已取消'

function createTimeoutError(operation) {
  const error = new Error(MODEL_BRIDGE_TIMEOUT_MESSAGE)
  error.code = 'frontend_wait_timeout'
  error.operation = operation
  return error
}

export function withModelBridgeTimeout(request, operation, timeoutMs = MODEL_BRIDGE_TIMEOUT_MS) {
  let timeoutID
  const timeout = new Promise((_, reject) => {
    timeoutID = setTimeout(() => reject(createTimeoutError(operation)), timeoutMs)
  })

  return Promise.race([Promise.resolve(request), timeout])
    .finally(() => clearTimeout(timeoutID))
}
