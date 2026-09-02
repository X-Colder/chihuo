const api = require('../../utils/api')
const format = require('../../utils/format')

Page({
  data: {
    campaigns: [],
    loading: true,
    error: '',
    empty: false,
    showOrder: false,
    selected: null,
    submitting: false,
    orderTotal: '¥0.00',
    orderForm: {
      quantity: '1',
      contactName: '',
      contactPhone: '',
      deliveryAddress: ''
    }
  },

  onShow() {
    this.load()
  },

  onPullDownRefresh() {
    this.load(true)
  },

  load(fromPullDown) {
    this.setData({ loading: true, error: '', empty: false })
    api.get('/v1/campaigns').then((items) => {
      const campaigns = format.list(items).map(format.campaign)
      this.setData({ campaigns, loading: false, empty: campaigns.length === 0 })
      if (fromPullDown) wx.stopPullDownRefresh()
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '预售商品加载失败' })
      if (fromPullDown) wx.stopPullDownRefresh()
    })
  },

  showOrderForm(event) {
    const selected = this.data.campaigns.find((item) => item.id === event.currentTarget.dataset.id)
    if (!selected || selected.stock < 1) {
      wx.showToast({ title: '该商品暂时没有余量', icon: 'none' })
      return
    }
    this.setData({
      selected,
      showOrder: true,
      orderTotal: selected.price
    })
  },

  hideOrderForm() {
    this.setData({ showOrder: false })
  },

  onOrderInput(event) {
    const field = event.currentTarget.dataset.field
    const value = event.detail.value
    const next = Object.assign({}, this.data.orderForm, { [field]: value })
    const quantity = Math.max(1, Number(next.quantity) || 1)
    const total = (this.data.selected ? this.data.selected.unitPriceCents * quantity : 0)
    this.setData({ orderForm: next, orderTotal: format.money(total) })
  },

  placeOrder() {
    const form = this.data.orderForm
    const campaign = this.data.selected
    if (!campaign) return
    if (!form.contactName || !form.contactPhone || !form.deliveryAddress) {
      wx.showToast({ title: '请完整填写收餐信息', icon: 'none' })
      return
    }
    const quantity = Number(form.quantity)
    if (!quantity || quantity < 1 || quantity > campaign.stock) {
      wx.showToast({ title: `数量需在 1-${campaign.stock} 份之间`, icon: 'none' })
      return
    }

    this.setData({ submitting: true })
    api.post(`/v1/campaigns/${campaign.id}/orders`, {
      quantity,
      delivery_address: form.deliveryAddress.trim(),
      contact_name: form.contactName.trim(),
      contact_phone: form.contactPhone.trim()
    }).then(() => {
      this.setData({ showOrder: false, submitting: false })
      wx.showToast({ title: '订单已提交', icon: 'success' })
      setTimeout(() => wx.switchTab({ url: '/pages/orders/index' }), 500)
    }).catch((error) => {
      this.setData({ submitting: false })
      wx.showToast({ title: error.message || '下单失败', icon: 'none' })
    })
  }
})
