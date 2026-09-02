Page({
  data: {
    records: [],
    loading: true,
    error: '',
    empty: false,
    showForm: false,
    submitting: false,
    severityOptions: ['低', '中', '高', '紧急'],
    severityIndex: 0,
    form: {
      title: '',
      description: ''
    }
  },

  onShow() {
    this.load()
  },

  onPullDownRefresh() {
    this.load(true)
  },

  load(fromPullDown) {
    this.setData({
      records: [],
      loading: false,
      error: '',
      empty: true
    })
    if (fromPullDown) wx.stopPullDownRefresh()
  },

  showCreate() {
    wx.showModal({
      title: '食品安全接口占位',
      content: 'Go API 当前版本尚未开放食品安全事件接口，页面已保留记录入口和字段设计。',
      showCancel: false
    })
  },

  hideCreate() {
    this.setData({ showForm: false })
  },

  onInput(event) {
    this.setData({
      [`form.${event.currentTarget.dataset.field}`]: event.detail.value
    })
  },

  onSeverityChange(event) {
    this.setData({ severityIndex: Number(event.detail.value) })
  },

  submit() {
    wx.showModal({
      title: '食品安全接口占位',
      content: '当前版本暂不提交到后端，正式接入后将保存事件、批次和证据链。',
      showCancel: false
    })
  }
})
