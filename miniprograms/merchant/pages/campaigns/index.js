const api = require('../../utils/api')
const format = require('../../utils/format')

Page({
  data: {
    campaigns: [],
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
    api.get('/v1/merchant/campaigns').then((items) => {
      const campaigns = format.list(items).map(format.campaign)
      this.setData({ campaigns, loading: false, empty: campaigns.length === 0 })
      if (fromPullDown) wx.stopPullDownRefresh()
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '预售活动加载失败' })
      if (fromPullDown) wx.stopPullDownRefresh()
    })
  }
})
