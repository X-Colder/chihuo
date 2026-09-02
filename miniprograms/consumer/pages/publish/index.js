const api = require('../../utils/api')

function today() {
  const date = new Date()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

function splitText(value) {
  return String(value || '').split(/[,，]/).map((item) => item.trim()).filter(Boolean)
}

Page({
  data: {
    submitting: false,
    form: {
      title: '',
      category: '工作餐',
      serviceArea: '',
      servingDate: today(),
      servingTime: '11:30',
      budgetMin: '20',
      budgetMax: '30',
      weightMinGrams: '300',
      weightMaxGrams: '400',
      minimumMembers: '10',
      maximumMembers: '100',
      hardConstraintsText: '',
      preferencesText: '',
      notes: ''
    }
  },

  onInput(event) {
    this.setData({
      [`form.${event.currentTarget.dataset.field}`]: event.detail.value
    })
  },

  onPickerChange(event) {
    this.setData({
      [`form.${event.currentTarget.dataset.field}`]: event.detail.value
    })
  },

  submit() {
    const form = this.data.form
    const required = [
      ['title', '请填写餐品名称'],
      ['serviceArea', '请填写配送区域'],
      ['servingDate', '请选择用餐日期'],
      ['servingTime', '请填写用餐时间']
    ]
    const missing = required.find((item) => !String(form[item[0]] || '').trim())
    if (missing) {
      wx.showToast({ title: missing[1], icon: 'none' })
      return
    }

    const payload = {
      title: form.title.trim(),
      category: form.category.trim() || '餐饮',
      service_area: form.serviceArea.trim(),
      serving_date: form.servingDate,
      serving_time: form.servingTime,
      budget_min_cents: Math.round(Number(form.budgetMin) * 100),
      budget_max_cents: Math.round(Number(form.budgetMax) * 100),
      quantity: 1,
      weight_min_grams: Number(form.weightMinGrams),
      weight_max_grams: Number(form.weightMaxGrams),
      hard_constraints: splitText(form.hardConstraintsText),
      preferences: splitText(form.preferencesText),
      notes: form.notes.trim() || undefined,
      minimum_members: Number(form.minimumMembers),
      maximum_members: Number(form.maximumMembers)
    }

    if (payload.budget_min_cents <= 0 || payload.budget_min_cents > payload.budget_max_cents) {
      wx.showToast({ title: '请检查预算范围', icon: 'none' })
      return
    }
    if (payload.weight_min_grams <= 0 || payload.weight_min_grams > payload.weight_max_grams) {
      wx.showToast({ title: '请检查重量范围', icon: 'none' })
      return
    }
    if (payload.minimum_members <= 0 || payload.minimum_members > payload.maximum_members) {
      wx.showToast({ title: '请检查成团人数', icon: 'none' })
      return
    }

    this.setData({ submitting: true })
    api.post('/v1/demands', payload).then((result) => {
      const demand = result && result.demand ? result.demand : result
      wx.showToast({ title: '需求已提交', icon: 'success' })
      setTimeout(() => {
        if (demand && demand.id) {
          wx.redirectTo({ url: `/pages/demand/detail?id=${demand.id}` })
        } else {
          wx.switchTab({ url: '/pages/index/index' })
        }
      }, 500)
    }).catch((error) => {
      wx.showToast({ title: error.message || '提交失败', icon: 'none' })
    }).finally(() => {
      this.setData({ submitting: false })
    })
  }
})
