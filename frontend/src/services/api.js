import * as AppBindings from '../../wailsjs/go/app/App'
import {
  CancelRunningTask,
  GetQuestionBankStatus,
  SelectQuestionBankOutput,
  ListQuestionBankSessions,
  OpenQuestionBankSession,
  SaveQuestionBankSession,
  CancelQuestionBankOrganizing,
  CheckScreenCapturePermission,
  ClearResume,
  GetDomainCategories,
  GetInitStatus,
  GetMobileOutputInfo,
  GetScreenshotPreview,
  GetSettings,
  MoveWindow,
  OpenScreenCaptureSettings,
  ParseResume,
  RemoveFocus,
  RequestScreenCapturePermission,
  RestoreFocus,
  SaveImageToFile,
  ScrollContent,
  SelectResume,
  SetWindowAlwaysOnTop,
  StartRecordingKey,
  StopRecordingKey,
  ToggleClickThrough,
  ToggleVisibility,
  TriggerSolve,
  TriggerScreenshot,
  TriggerSend,
  RemoveScreenshot,
  ClearScreenshots,
  UpdateSettings,
} from '../../wailsjs/go/app/App'

import { Quit } from '../../wailsjs/runtime/runtime'

// 新接口与生成代码可能分批更新：优先使用 binding，缺失时回退到 Wails bridge。
function callApp(method, ...args) {
  const binding = AppBindings[method]
  if (typeof binding === 'function') return binding(...args)

  const bridgeMethod = globalThis.window?.go?.app?.App?.[method]
  if (typeof bridgeMethod === 'function') return bridgeMethod(...args)
  return Promise.reject(new Error(`Wails API ${method} 尚未生成`))
}

export const api = {
  getQuestionBankPreferences: () => callApp('GetQuestionBankPreferences'),
  updateQuestionBankPreferences: (preferences) => callApp('UpdateQuestionBankPreferences', preferences),
  getQuestionBankStatus: () => GetQuestionBankStatus(),
  selectQuestionBankOutput: () => SelectQuestionBankOutput(),
  splitQuestionBankSegment: () => callApp('SplitQuestionBankSegment'),
  discardQuestionBankSegment: (confirmed) => callApp('DiscardQuestionBankSegment', confirmed),
  exportCurrentQuestionBankSegment: () => callApp('ExportCurrentQuestionBankSegment'),
  exportQuestionBankSession: (id) => callApp('ExportQuestionBankSession', id),
  getQuestionBankExportDraft: (id) => callApp('GetQuestionBankExportDraft', id),
  selectQuestionBankExportDirectory: (current) => callApp('SelectQuestionBankExportDirectory', current),
  exportQuestionBankSegment: (request) => callApp('ExportQuestionBankSegment', request),
  listQuestionBankSessions: () => ListQuestionBankSessions(),
  openQuestionBankSession: (id) => OpenQuestionBankSession(id),
  saveQuestionBankSession: () => SaveQuestionBankSession(),
  cancelQuestionBankOrganizing: () => CancelQuestionBankOrganizing(),
  getSettings: () => GetSettings(),
  syncSettings: (json) => UpdateSettings(json),
  updateSettings: (json) => UpdateSettings(json),

  getModels: (apiKey, baseURL) => callApp('GetModelsForConnection', apiKey, baseURL),
  testConnection: (apiKey, baseURL, model) => callApp('TestConnectionForConnection', apiKey, baseURL, model),

  triggerSolve: () => TriggerSolve(),
  triggerScreenshot: () => TriggerScreenshot(),
  triggerSend: () => TriggerSend(),
  removeScreenshot: (index) => RemoveScreenshot(index),
  clearScreenshots: () => ClearScreenshots(),
  cancelTask: () => CancelRunningTask(),

  startRecordingKey: (action) => StartRecordingKey(action),
  stopRecordingKey: () => StopRecordingKey(),

  selectResume: () => SelectResume(),
  clearResume: () => ClearResume(),
  parseResume: () => ParseResume(),

  restoreFocus: () => RestoreFocus(),
  removeFocus: () => RemoveFocus(),
  moveWindow: (x, y) => MoveWindow(x, y),
  scrollContent: (dir) => ScrollContent(dir),
  setAlwaysOnTop: (v) => SetWindowAlwaysOnTop(v),
  toggleVisibility: () => ToggleVisibility(),
  toggleClickThrough: () => ToggleClickThrough(),
  quit: () => Quit(),

  getInitStatus: () => GetInitStatus(),
  getDomainCategories: () => GetDomainCategories(),
  getMobileOutputInfo: () => GetMobileOutputInfo(),
  retryMobileOutput: () => callApp('RetryMobileOutput'),

  getScreenshotPreview: (q, s, g, n, m) => GetScreenshotPreview(q, s, g, n, m),
  checkScreenCapturePermission: () => CheckScreenCapturePermission(),
  requestScreenCapturePermission: () => RequestScreenCapturePermission(),
  openScreenCaptureSettings: () => OpenScreenCaptureSettings(),

  saveImageToFile: (b64) => SaveImageToFile(b64),
}
