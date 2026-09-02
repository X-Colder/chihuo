var DEFAULT_API_BASE_URL = 'http://localhost:4000'
var TOKEN_KEY = 'chihuo.rider.token'

function getAppConfig() {
  var app = null
  try {
    app = getApp()
  } catch (error) {
    app = null
  }

  var storedBaseUrl = wx.getStorageSync('chihuo.rider.apiBaseUrl')
  var appBaseUrl = app && app.globalData
    ? (app.globalData.API_BASE_URL || app.globalData.apiBaseUrl || '')
    : ''
  var baseUrl = storedBaseUrl || appBaseUrl || DEFAULT_API_BASE_URL

  return {
    baseUrl: String(baseUrl).replace(/\/+$/, ''),
    tokenKey: app && app.globalData && app.globalData.tokenKey ? app.globalData.tokenKey : TOKEN_KEY
  }
}

function getApiBaseUrl() {
  return getAppConfig().baseUrl
}

function setApiBaseUrl(baseUrl) {
  var normalized = String(baseUrl || '').trim().replace(/\/+$/, '')
  if (!normalized) {
    wx.removeStorageSync('chihuo.rider.apiBaseUrl')
    return DEFAULT_API_BASE_URL
  }
  wx.setStorageSync('chihuo.rider.apiBaseUrl', normalized)
  return normalized
}

function getToken() {
  return wx.getStorageSync(getAppConfig().tokenKey) || ''
}

function setToken(token) {
  wx.setStorageSync(getAppConfig().tokenKey, token)
}

function clearToken() {
  wx.removeStorageSync(getAppConfig().tokenKey)
}

function getPayload(data) {
  if (data && Object.prototype.hasOwnProperty.call(data, 'data')) {
    return data.data
  }
  return data
}

function createError(message, statusCode, payload) {
  var error = new Error(message || '网络请求失败')
  error.statusCode = statusCode || 0
  error.payload = payload
  return error
}

function ensureToken() {
  var token = getToken()
  if (token) {
    return Promise.resolve(token)
  }

  return new Promise(function (resolve, reject) {
    wx.login({
      success: function (loginResult) {
        if (!loginResult.code) {
          reject(createError('微信登录未返回 code'))
          return
        }
        wx.request({
          url: getApiBaseUrl() + '/v1/auth/dev/wechat-login',
          method: 'POST',
          data: {
            code: loginResult.code,
            name: '演示骑手',
            role: 'RIDER'
          },
          header: {
            'content-type': 'application/json'
          },
          timeout: 8000,
          success: function (response) {
            var payload = getPayload(response.data)
            if (response.statusCode >= 200 && response.statusCode < 300 && payload && payload.token) {
              setToken(payload.token)
              resolve(payload.token)
              return
            }
            reject(createError('登录失败', response.statusCode, response.data))
          },
          fail: function (error) {
            reject(createError(error.errMsg || '登录请求失败'))
          }
        })
      },
      fail: function (error) {
        reject(createError(error.errMsg || '微信登录调用失败'))
      }
    })
  })
}

function request(options) {
  var requestOptions = options || {}

  return ensureToken().then(function (token) {
    return new Promise(function (resolve, reject) {
      var headers = Object.assign(
        {
          'content-type': 'application/json',
          Authorization: 'Bearer ' + token
        },
        requestOptions.header || {}
      )

      wx.request({
        url: getApiBaseUrl() + requestOptions.url,
        method: requestOptions.method || 'GET',
        data: requestOptions.data || {},
        header: headers,
        timeout: requestOptions.timeout || 8000,
        success: function (response) {
          var payload = getPayload(response.data)
          if (response.statusCode >= 200 && response.statusCode < 300) {
            resolve(payload)
            return
          }
          if (response.statusCode === 401) {
            clearToken()
          }
          var message = response.data && response.data.error && response.data.error.message
          reject(createError(message || '请求失败', response.statusCode, response.data))
        },
        fail: function (error) {
          reject(createError(error.errMsg || '网络请求失败'))
        }
      })
    })
  })
}

function get(url, data) {
  return request({
    url: url,
    method: 'GET',
    data: data || {}
  })
}

function post(url, data) {
  return request({
    url: url,
    method: 'POST',
    data: data || {}
  })
}

function patch(url, data) {
  return request({
    url: url,
    method: 'PATCH',
    data: data || {}
  })
}

module.exports = {
  request: request,
  get: get,
  post: post,
  patch: patch,
  getToken: getToken,
  setToken: setToken,
  clearToken: clearToken,
  getApiBaseUrl: getApiBaseUrl,
  setApiBaseUrl: setApiBaseUrl
}
