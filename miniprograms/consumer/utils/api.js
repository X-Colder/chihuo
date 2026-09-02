const TOKEN_KEY = 'chihuo.consumer.access-token'
const USER_KEY = 'chihuo.consumer.session-user'

function getConfig() {
  const app = getApp()
  return (app && app.globalData && app.globalData.config) || {}
}

function baseUrl() {
  return String(getConfig().API_BASE_URL || '').replace(/\/$/, '')
}

function toError(body, statusCode) {
  const message = body && body.error && body.error.message
  return new Error(message || `请求失败（${statusCode || '网络错误'}）`)
}

function readSession() {
  return {
    token: wx.getStorageSync(TOKEN_KEY) || '',
    user: wx.getStorageSync(USER_KEY) || null
  }
}

function clearSession() {
  wx.removeStorageSync(TOKEN_KEY)
  wx.removeStorageSync(USER_KEY)
}

function saveSession(data) {
  if (!data || !data.token) {
    throw new Error('开发登录响应缺少 token')
  }
  const rawUser = data.user || {}
  const user = Object.assign({}, rawUser, {
    merchantId: rawUser.merchantId || rawUser.merchant_id || ''
  })
  wx.setStorageSync(TOKEN_KEY, data.token)
  wx.setStorageSync(USER_KEY, user)
  return Object.assign({}, data, { user })
}

function wxLoginCode() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: (result) => {
        if (result.code) {
          resolve(result.code)
        } else {
          reject(new Error('微信开发登录未返回 code'))
        }
      },
      fail: (error) => reject(new Error(error.errMsg || '微信登录调用失败'))
    })
  })
}

function devLogin(force) {
  const existing = readSession()
  if (!force && existing.token) return Promise.resolve(existing)

  const config = getConfig()
  const loginPath = config.LOGIN_MODE === 'wechat'
    ? (config.LOGIN_PATH || '/v1/auth/wechat-login')
    : (config.DEV_LOGIN_PATH || '/v1/auth/dev/wechat-login')
  return wxLoginCode().then((code) => new Promise((resolve, reject) => {
    wx.request({
      url: `${baseUrl()}${loginPath}`,
      method: 'POST',
      data: {
        code,
        name: config.DEV_NAME || '开发消费者',
        role: config.DEV_ROLE || 'CONSUMER'
      },
      header: {
        'content-type': 'application/json'
      },
      success: (response) => {
        if (response.statusCode >= 200 && response.statusCode < 300) {
          try {
            const body = response.data || {}
            resolve(saveSession(body.data || body))
          } catch (error) {
            reject(error)
          }
          return
        }
        reject(toError(response.data, response.statusCode))
      },
      fail: (error) => reject(new Error(error.errMsg || '无法连接 Go API'))
    })
  })).then((data) => ({
    token: data.token,
    user: data.user || null
  }))
}

function request(path, options, canRetry) {
  const requestOptions = options || {}
  const retry = canRetry !== false
  return devLogin(false).then((session) => new Promise((resolve, reject) => {
    const headers = Object.assign({
      'content-type': 'application/json',
      'X-Request-ID': `wx-${Date.now()}-${Math.random().toString(16).slice(2)}`
    }, requestOptions.header || {})
    if (session.token) headers.Authorization = `Bearer ${session.token}`

    wx.request({
      url: `${baseUrl()}${path}`,
      method: requestOptions.method || 'GET',
      data: requestOptions.data,
      header: headers,
      success: (response) => {
        if (response.statusCode === 401 && retry) {
          clearSession()
          request(path, requestOptions, false).then(resolve).catch(reject)
          return
        }
        if (response.statusCode >= 200 && response.statusCode < 300) {
          const body = response.data || {}
          resolve(body.data !== undefined ? body.data : body)
          return
        }
        reject(toError(response.data, response.statusCode))
      },
      fail: (error) => reject(new Error(error.errMsg || '网络请求失败'))
    })
  }))
}

module.exports = {
  get(path, options) {
    return request(path, Object.assign({}, options, { method: 'GET' }))
  },
  post(path, data, options) {
    return request(path, Object.assign({}, options, { method: 'POST', data }))
  },
  getSession: readSession,
  devLogin,
  clearSession
}
