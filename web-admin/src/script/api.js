// API 封装

import axios from 'axios'
import { useStore } from './store'
import { shouldResetAuth, toApiError } from './error-codes'

const API_BASE_URL = `${window.location.origin}/api/v1`
const DEFAULT_REQUEST_TIMEOUT_MS = 60000
const AI_REQUEST_TIMEOUT_MS = 180000

class ApiService {
  constructor() {
    this.baseURL = API_BASE_URL
    
    // 创建axios实例
    this.api = axios.create({
      baseURL: this.baseURL,
      timeout: DEFAULT_REQUEST_TIMEOUT_MS,
      headers: {
        'Content-Type': 'application/json'
      }
    })
    
    // 请求拦截器
    this.api.interceptors.request.use(
      config => {
        const store = useStore()
        if (store.token) {
          config.headers.Authorization = `Bearer ${store.token}`
        }
        return config
      },
      error => {
        return Promise.reject(error)
      }
    )
    
    // 响应拦截器
    this.api.interceptors.response.use(
      response => {
        return response
      },
      error => {
        const apiError = toApiError(error)
        if (shouldResetAuth(apiError.code)) {
          this.removeToken()
          if (window.location.pathname !== '/login') {
            window.location.href = '/login'
          }
        }
        return Promise.reject(apiError)
      }
    )
  }

  setToken(token) {
    const store = useStore()
    store.token = token
  }

  removeToken() {
    const store = useStore()
    store.clearUser()
  }

  // 登录
  async login(username, password, captcha = '', captchaId = '') {
    const payload = {
      username,
      password
    }
    if (String(captcha || '').trim()) {
      payload.captcha = String(captcha || '').trim()
      if (String(captchaId || '').trim()) {
        payload.captcha_id = String(captchaId || '').trim()
      }
    }
    const data = await this.api.post('/login', payload)

    const token = data?.data?.data?.token || ''
    this.setToken(token)
    return data
  }

  async getCaptcha() {
    return this.api.get('/captcha')
  }

  // 获取用户信息
  async getUserInfo() {
    return this.api.get('/user/info')
  }

  // 登出
  async logout() {
    await this.api.post('/logout')
    this.removeToken()
  }

  // App 管理
  async listApps(params) {
    return this.api.get('/apps/list', { params })
  }

  async createApp(data) {
    return this.api.post('/apps/create', data)
  }

  async updateApp(appId, data) {
    return this.api.put('/apps/update', { app_id: appId, ...data })
  }

  async deleteApp(appId) {
    return this.api.delete('/apps/delete', {
      params: { app_id: appId }
    })
  }

  // 客服管理
  async listStaff(params) {
    return this.api.get('/user/list', { params })
  }

  async createStaff(data) {
    return this.api.post('/user/create', data)
  }

  async updateStaff(data) {
    return this.api.put('/user/update', data)
  }

  async deleteStaff(id) {
    return this.api.delete('/user/delete', { params: { id } })
  }

  async batchSetStaffActive(ids, active) {
    return this.api.post('/user/batch-active', { ids, active })
  }

  // 用户状态管理
  async setUserStatus(status) {
    return this.api.post('/user/status', { status })
  }

  async getUserStatus() {
    return this.api.get('/user/status')
  }

  // 用户资料
  async updateProfile(data) {
    return this.api.put('/user/profile', data)
  }

  async changePassword(currentPassword, newPassword) {
    return this.api.post('/user/password', {
      current_password: currentPassword,
      new_password: newPassword
    })
  }

  // 会话工作台
  async listSessions(params) {
    return this.api.get('/sessions/list', { params })
  }

  async getSessionMessages(params) {
    return this.api.get('/sessions/messages', { params })
  }

  async acceptSession(sid) {
    return this.api.post('/sessions/accept', { sid })
  }

  async transferSession(sid, toAgentName) {
    return this.api.post('/sessions/transfer', { sid, to_agent_name: toAgentName })
  }

  async closeSession(sid) {
    return this.api.post('/sessions/close', { sid })
  }

  async readSession(sid) {
    return this.api.post('/sessions/read', { sid })
  }

  async followUpSession(sid) {
    return this.api.post('/sessions/follow-up', { sid })
  }

  async listSessionAgents(params) {
    return this.api.get('/sessions/agents', { params })
  }

  // 快捷回复
  async listQuickReplies(params) {
    return this.api.get('/quick-replies/list', { params })
  }

  async createQuickReply(data) {
    return this.api.post('/quick-replies/create', data)
  }

  async updateQuickReply(data) {
    return this.api.put('/quick-replies/update', data)
  }

  async deleteQuickReply(id) {
    return this.api.delete('/quick-replies/delete', { params: { id } })
  }

  async useQuickReply(id) {
    return this.api.post('/quick-replies/use', { id })
  }

  // 审计
  async listAuditLogs(params) {
    return this.api.get('/audit/list', { params })
  }

  // 管理面板
  async getDashboard(params) {
    return this.api.get('/panel/dashboard', { params })
  }

  async listVisitors(params) {
    return this.api.get('/panel/visitors', { params })
  }

  async listUserStats(params) {
    return this.api.get('/panel/user-stats', { params })
  }

  async getSystemSettings() {
    return this.api.get('/panel/settings')
  }

  async updateSystemSettings(data) {
    return this.api.put('/panel/settings', data)
  }

  async getProfileSummary() {
    return this.api.get('/panel/profile-summary')
  }

  async exportSessions(params) {
    return this.api.get('/panel/export/sessions', {
      params,
      responseType: 'blob'
    })
  }

  async rateSession(payloadOrSID, score, comment = '') {
    const payload = typeof payloadOrSID === 'object' && payloadOrSID !== null
      ? payloadOrSID
      : { sid: payloadOrSID, score, comment }

    return this.api.post('/sessions/rate', {
      sid: payload.sid,
      app_id: payload.app_id || payload.appId || '',
      visitor_id: payload.visitor_id || payload.visitorId || '',
      score: payload.score,
      comment: payload.comment || ''
    })
  }

  async uploadFile(appId, file, contentType = '') {
    const formData = new FormData()
    formData.append('app_id', appId)
    if (contentType) {
      formData.append('content_type', contentType)
    }
    formData.append('file', file)
    return this.api.post('/upload-auth', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  }

  // 坐席个人设置
  async getAgentSettings() {
    return this.api.get('/agent/settings')
  }

  async updateAgentSettings(data) {
    return this.api.put('/agent/settings', data)
  }

  async getAgentAIBotSettings() {
    return this.api.get('/agent/ai-bot-settings')
  }

  async updateAgentAIBotSettings(data) {
    return this.api.put('/agent/ai-bot-settings', data)
  }

  async getAgentSensitiveWords() {
    return this.api.get('/agent/sensitive-words')
  }

  async updateAgentSensitiveWords(data) {
    return this.api.put('/agent/sensitive-words', data)
  }

  // AI 建议
  async suggestAIReply(sid = '', query = '', appId = '') {
    return this.api.post('/ai/suggest', { sid, query, app_id: appId }, { timeout: AI_REQUEST_TIMEOUT_MS })
  }

  async testAIBot(appId = '', query = '') {
    return this.api.post('/ai/bot-test', { app_id: appId, query }, { timeout: AI_REQUEST_TIMEOUT_MS })
  }

  // 新知识库工作区
  async listKnowledgeBases(params = {}) {
    return this.api.get('/knowledge-bases/list', { params })
  }

  async createKnowledgeBase(data) {
    return this.api.post('/knowledge-bases/create', data)
  }

  async updateKnowledgeBase(data) {
    return this.api.put('/knowledge-bases/update', data)
  }

  async deleteKnowledgeBase(id) {
    return this.api.delete('/knowledge-bases/delete', { params: { id } })
  }

  async listKnowledgeDocuments(baseID, params = {}) {
    return this.api.get(`/knowledge-bases/${baseID}/documents`, { params })
  }

  async uploadKnowledgeDocument(baseID, file) {
    const formData = new FormData()
    formData.append('file', file)
    return this.api.post(`/knowledge-bases/${baseID}/documents/upload`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      timeout: 10 * 60 * 1000
    })
  }

  async reindexKnowledgeDocument(baseID, documentID) {
    return this.api.post(`/knowledge-bases/${baseID}/documents/reindex`, { document_id: documentID })
  }

  async deleteKnowledgeDocument(baseID, documentID) {
    return this.api.delete(`/knowledge-bases/${baseID}/documents/${documentID}`)
  }

  async listKnowledgeChunks(baseID, params = {}) {
    return this.api.get(`/knowledge-bases/${baseID}/chunks`, { params })
  }

  async updateKnowledgeChunk(baseID, chunkID, content) {
    return this.api.put(`/knowledge-bases/${baseID}/chunks/${chunkID}`, { content })
  }

  async deleteKnowledgeChunk(baseID, chunkID) {
    return this.api.delete(`/knowledge-bases/${baseID}/chunks/${chunkID}`)
  }

  async knowledgeRetrieveTest(baseID, query, topK = 5, timeoutSec = 120) {
    const payload = { query, top_k: topK }
    const timeoutNum = Number(timeoutSec || 120)
    const timeoutMs = Math.max(15000, Math.min(900000, (Number.isFinite(timeoutNum) ? timeoutNum : 120) * 1000 + 10000))
    return this.api.post(`/knowledge-bases/${baseID}/retrieve-test`, payload, { timeout: timeoutMs })
  }

  async knowledgeQATest(baseID, query, topK = 5, timeoutSec = 120) {
    const payload = { query, top_k: topK }
    const timeoutNum = Number(timeoutSec || 120)
    const timeoutMs = Math.max(15000, Math.min(900000, (Number.isFinite(timeoutNum) ? timeoutNum : 120) * 1000 + 10000))
    return this.api.post(`/knowledge-bases/${baseID}/qa-test`, payload, { timeout: timeoutMs })
  }

  async listAPIModels(modelType) {
    const params = modelType ? { model_type: modelType } : {}
    return this.api.get('/admin/api-models', { params })
  }

  async getAPIModel(id) {
    return this.api.get(`/admin/api-models/${id}`)
  }

  async createAPIModel(data) {
    return this.api.post('/admin/api-models', data, { timeout: AI_REQUEST_TIMEOUT_MS })
  }

  async updateAPIModel(id, data) {
    return this.api.put(`/admin/api-models/${id}`, data, { timeout: AI_REQUEST_TIMEOUT_MS })
  }

  async deleteAPIModel(id) {
    return this.api.delete(`/admin/api-models/${id}`)
  }

  async testAPIModel(id) {
    return this.api.post(`/admin/api-models/${id}/test`, {}, { timeout: 180000 })
  }

  async setDefaultAPIModel(id) {
    return this.api.post(`/admin/api-models/${id}/set-default`)
  }

  async triggerRebuild(id) {
    return this.api.post(`/admin/api-models/${id}/rebuild`)
  }

  async getRebuildStatus(id) {
    return this.api.get(`/admin/api-models/${id}/rebuild`)
  }

  async saveKnowledgeFeedback(baseID, data) {
    return this.api.post(`/knowledge-bases/${baseID}/feedback`, data)
  }

  async checkKnowledgeBackend() {
    return this.api.get('/knowledge-bases/healthz')
  }

  // FAQ
  async listFAQ(params) {
    return this.api.get('/faq/list', { params })
  }

  async createFAQ(data) {
    return this.api.post('/faq/create', data)
  }

  async updateFAQ(data) {
    return this.api.put('/faq/update', data)
  }

  async deleteFAQ(id) {
    return this.api.delete('/faq/delete', { params: { id } })
  }

  // 应用 API 密钥
  async listAppAPIKeys(appId) {
    return this.api.get('/app-api-keys/list', { params: { app_id: appId } })
  }

  async createAppAPIKey(data) {
    return this.api.post('/app-api-keys/create', data)
  }

  async rotateAppAPIKey(id) {
    return this.api.post('/app-api-keys/rotate', { id })
  }

  async setAppAPIKeyEnabled(id, enabled) {
    return this.api.post('/app-api-keys/set-enabled', { id, enabled })
  }

  async deleteAppAPIKey(id) {
    return this.api.delete('/app-api-keys/delete', { params: { id } })
  }

  // 企业微信 - 管理员配置
  async getWecomConfig() {
    return this.api.get('/admin/wecom/config')
  }

  async saveWecomConfig(data) {
    return this.api.post('/admin/wecom/config', data)
  }

  async testWecomConnection(data) {
    return this.api.post('/admin/wecom/test', data)
  }

  // 企业微信 - 客服绑定
  async getWecomQrcode() {
    return this.api.get('/agent/notification/channels/wecom/qrcode')
  }

  async getWecomBindStatus(state) {
    return this.api.get('/agent/notification/channels/wecom/bind-status', { params: { state } })
  }

  async getWecomBindInfo() {
    return this.api.get('/agent/notification/channels/wecom/bind-info')
  }

  async unbindWecom() {
    return this.api.post('/agent/notification/channels/wecom/unbind')
  }

  // 通知系统 - 管理员配置
  async getNotificationChannels() {
    return this.api.get('/admin/notification/channels')
  }

  async getNotificationChannelConfig(channel) {
    return this.api.get(`/admin/notification/channels/${channel}`)
  }

  async saveNotificationChannelConfig(channel, data) {
    return this.api.post(`/admin/notification/channels/${channel}`, data)
  }

  async testNotificationChannel(channel, data) {
    return this.api.post(`/admin/notification/channels/${channel}/test`, data)
  }

  // 通知系统 - 客服绑定
  async getNotificationBindQrcode(channel) {
    return this.api.get(`/agent/notification/channels/${channel}/qrcode`)
  }

  async getNotificationBindStatus(channel, state) {
    return this.api.get(`/agent/notification/channels/${channel}/bind-status`, { params: { state } })
  }

  async getNotificationBindInfo(channel) {
    return this.api.get(`/agent/notification/channels/${channel}/bind-info`)
  }

  async unbindNotification(channel) {
    return this.api.post(`/agent/notification/channels/${channel}/unbind`)
  }
}

export default new ApiService()
