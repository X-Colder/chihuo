const api = require('../../utils/api')

function splitText(value) {
  return String(value || '').split(/[,，]/).map((item) => item.trim()).filter(Boolean)
}

Page({
  data: {
    demandId: '',
    demandTitle: '需求报价',
    submitting: false,
    oilOptions: ['未知', '低油', '适中油', '高油'],
    saltOptions: ['未知', '少盐', '适中盐', '高盐'],
    oilIndex: 1,
    saltIndex: 1,
    form: {
      price: '25',
      capacity: '30',
      weightGrams: '350',
      productionTime: '11:00',
      ingredientsText: '',
      allergensText: '无',
      shelfLifeMinutes: '180',
      storageInstructions: '建议两小时内食用，配送过程中保持密封。',
      notes: ''
    }
  },

  onLoad(options) {
    this.setData({
      demandId: options.demandId || '',
      demandTitle: options.title ? decodeURIComponent(options.title) : '需求报价'
    })
  },

  onInput(event) {
    this.setData({
      [`form.${event.currentTarget.dataset.field}`]: event.detail.value
    })
  },

  onOilChange(event) {
    this.setData({ oilIndex: Number(event.detail.value) })
  },

  onSaltChange(event) {
    this.setData({ saltIndex: Number(event.detail.value) })
  },

  submit() {
    const form = this.data.form
    if (!this.data.demandId || !form.price || !form.capacity || !form.weightGrams || !form.productionTime || !form.storageInstructions) {
      wx.showToast({ title: '请完整填写报价信息', icon: 'none' })
      return
    }
    const ingredients = splitText(form.ingredientsText)
    if (!ingredients.length) {
      wx.showToast({ title: '请填写主要食材', icon: 'none' })
      return
    }
    const allergens = splitText(form.allergensText).filter((item) => item !== '无')
    const levels = ['UNKNOWN', 'LOW', 'MEDIUM', 'HIGH']
    const payload = {
      demand_id: this.data.demandId,
      unit_price_cents: Math.round(Number(form.price) * 100),
      production_capacity: Number(form.capacity),
      weight_grams: Number(form.weightGrams),
      ingredients,
      allergens,
      oil_level: levels[this.data.oilIndex],
      salt_level: levels[this.data.saltIndex],
      production_time: form.productionTime,
      shelf_life_minutes: Number(form.shelfLifeMinutes),
      storage_instructions: form.storageInstructions.trim(),
      notes: form.notes.trim() || undefined
    }
    if (!payload.unit_price_cents || !payload.production_capacity || !payload.weight_grams || !payload.shelf_life_minutes) {
      wx.showToast({ title: '价格、产能、重量和保质时长必须大于 0', icon: 'none' })
      return
    }

    this.setData({ submitting: true })
    api.post('/v1/merchant/offers', payload).then(() => {
      wx.showToast({ title: '报价已提交', icon: 'success' })
      setTimeout(() => wx.navigateBack(), 500)
    }).catch((error) => {
      wx.showToast({ title: error.message || '报价提交失败', icon: 'none' })
    }).finally(() => {
      this.setData({ submitting: false })
    })
  }
})
