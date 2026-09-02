App({
  globalData: {
    // 本地开发保持 dev，生产发布改为 wechat 并配置真实微信 AppID/Secret。
    API_BASE_URL: 'http://localhost:4000',
    tokenKey: 'chihuo.rider.token',
    LOGIN_MODE: 'dev',
    LOGIN_PATH: '/v1/auth/wechat-login',
    DEV_LOGIN_PATH: '/v1/auth/dev/wechat-login'
  }
})
