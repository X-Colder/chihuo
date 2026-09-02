const api = require('../../utils/api')
const format = require('../../utils/format')

Page({
  data: {
    demands: [],
    summary: { open: 0, quoted: 0, ready: 0 },
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
    Promise.all([
      api.get('/v1/merchant/demands'),
      api.get('/v1/merchant/offers')
    ]).then((results) => {
      const rawDemands = format.list(results[0])
      const offers = format.list(results[1])
      const quotedIds = {}
      offers.forEach((item) => {
        quotedIds[item.demandId] = true
      })
      const demands = rawDemands.map((item) => format.demand(item, quotedIds))
      this.setData({
        demands,
        summary: {
          open: demands.filter((item) => !item.quoted).length,
          quoted: offers.length,
          ready: rawDemands.filter((item) => item.status === 'READY').length
        },
        loading: false,
        empty: demands.length === 0
      })
      if (fromPullDown) wx.stopPullDownRefresh()
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '需求大厅加载失败' })
      if (fromPullDown) wx.stopPullDownRefresh()
    })
  },

  openDemand(event) {
    wx.navigateTo({
      url: `/pages/demand/detail?id=${event.currentTarget.dataset.id}`
    })
  }
})
