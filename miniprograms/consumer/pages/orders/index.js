const api = require('../../utils/api')
const format = require('../../utils/format')

Page({
  data: {
    orders: [],
    loading: true,
    error: '',
    empty: false
  },

  onShow() {
    this.load()
  },

  onPullDownRefresh() {
    this.load(true)
  },

  load(fromPullDown) {
    this.setData({ loading: true, error: '', empty: false })
    api.get('/v1/orders').then((items) => {
      const orders = format.list(items).map(format.order)
      this.setData({ orders, loading: false, empty: orders.length === 0 })
      if (fromPullDown) wx.stopPullDownRefresh()
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '订单加载失败' })
      if (fromPullDown) wx.stopPullDownRefresh()
    })
  },

  goCampaigns() {
    wx.switchTab({ url: '/pages/campaigns/index' })
  }
})
