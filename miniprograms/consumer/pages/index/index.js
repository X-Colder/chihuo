const api = require('../../utils/api')
const format = require('../../utils/format')

Page({
  data: {
    demands: [],
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
    api.get('/v1/demands').then((items) => {
      const demands = format.list(items).map(format.demand)
      this.setData({ demands, loading: false, empty: demands.length === 0 })
      if (fromPullDown) wx.stopPullDownRefresh()
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '需求加载失败' })
      if (fromPullDown) wx.stopPullDownRefresh()
    })
  },

  openDemand(event) {
    wx.navigateTo({
      url: `/pages/demand/detail?id=${event.currentTarget.dataset.id}`
    })
  },

  goPublish() {
    wx.navigateTo({ url: '/pages/publish/index' })
  }
})
