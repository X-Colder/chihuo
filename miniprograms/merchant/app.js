App({
  globalData: {
    config: {
      // Replace this with the reachable Go API address for the current environment.
      API_BASE_URL: 'http://127.0.0.1:4000',
      DEV_LOGIN_PATH: '/v1/auth/dev/wechat-login',
      DEV_ROLE: 'MERCHANT',
      DEV_NAME: '开发商家',
      DEV_MERCHANT_NAME: '吃货开发演示餐厅'
    }
  },

  onLaunch() {
    wx.getSystemInfo({
      success: (info) => {
        this.globalData.windowWidth = info.windowWidth
        this.globalData.statusBarHeight = info.statusBarHeight
      }
    })
  }
})
