const api = require('../../utils/api')
const format = require('../../utils/format')

Page({
  data: {
    id: '',
    demand: null,
    loading: true,
    error: ''
  },

  onLoad(options) {
    this.setData({ id: options.id || '' })
    this.load()
  },

  load() {
    if (!this.data.id) {
      this.setData({ loading: false, error: '缺少需求编号' })
      return
    }
    api.get(`/v1/demands/${this.data.id}`).then((item) => {
      this.setData({ demand: format.demand(item), loading: false })
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '需求加载失败' })
    })
  },

  goQuote() {
    wx.navigateTo({
      url: `/pages/quote/index?demandId=${this.data.id}&title=${encodeURIComponent(this.data.demand.title)}`
    })
  }
})
