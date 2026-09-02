const api = require('../../utils/api')
const format = require('../../utils/format')

function splitText(value) {
  return String(value || '').split(/[,，]/).map((item) => item.trim()).filter(Boolean)
}

Page({
  data: {
    id: '',
    demand: null,
    loading: true,
    error: '',
    joining: false,
    quantity: '1',
    weightGrams: '',
    preferencesText: ''
  },

  onLoad(options) {
    this.setData({ id: options.id || '' })
    this.load()
  },

  onInput(event) {
    this.setData({
      [event.currentTarget.dataset.field]: event.detail.value
    })
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

  join() {
    const quantity = Number(this.data.quantity)
    if (!quantity || quantity < 1) {
      wx.showToast({ title: '请输入购买数量', icon: 'none' })
      return
    }
    this.setData({ joining: true })
    api.post(`/v1/demands/${this.data.id}/join`, {
      quantity,
      weight_grams: this.data.weightGrams ? Number(this.data.weightGrams) : undefined,
      preferences: splitText(this.data.preferencesText)
    }).then((result) => {
      const item = result && result.demand ? result.demand : result
      this.setData({ demand: format.demand(item) })
      wx.showToast({ title: '已加入需求', icon: 'success' })
    }).catch((error) => {
      wx.showToast({ title: error.message || '加入失败', icon: 'none' })
    }).finally(() => {
      this.setData({ joining: false })
    })
  }
})
