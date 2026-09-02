const api = require('../../utils/api')

const severityValues = ['LOW', 'MEDIUM', 'HIGH', 'CRITICAL']
const severityLabels = ['低', '中', '高', '紧急']

function mapRecord(item) {
  const statusMap = {
    REPORTED: ['待处理', 'status-warn'],
    CONTAINED: ['已控制', 'status-warn'],
    INVESTIGATING: ['调查中', 'status-info'],
    CONFIRMED: ['已确认', 'status-danger'],
    RECALLING: ['召回中', 'status-danger'],
    COMPENSATING: ['赔偿中', 'status-info'],
    RESOLVED: ['已解决', 'status-success'],
    CLOSED: ['已结案', 'status-success'],
    DISMISSED: ['已驳回', 'status-muted']
  }
  const status = statusMap[item.status] || [item.status || '未知', 'status-muted']
  const severityIndex = Math.max(0, severityValues.indexOf(item.severity))
  return Object.assign({}, item, {
    createdAt: item.created_at || item.reported_at || '',
    severityText: severityLabels[severityIndex],
    status: status[0],
    statusClass: status[1]
  })
}

Page({
  data: {
    records: [],
    loading: true,
    error: '',
    empty: false,
    showForm: false,
    submitting: false,
    severityOptions: severityLabels,
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
    this.setData({ loading: true, error: '', empty: false })
    api.get('/v1/merchant/safety/incidents').then((items) => {
      const records = (Array.isArray(items) ? items : []).map(mapRecord)
      this.setData({ records, loading: false, empty: records.length === 0 })
      if (fromPullDown) wx.stopPullDownRefresh()
    }).catch((error) => {
      this.setData({ loading: false, error: error.message || '安全记录加载失败' })
      if (fromPullDown) wx.stopPullDownRefresh()
    })
  },

  showCreate() {
    this.setData({
      showForm: true,
      form: { title: '', description: '' },
      severityIndex: 0
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
    const title = this.data.form.title.trim()
    const description = this.data.form.description.trim()
    if (!title || !description) {
      wx.showToast({ title: '请填写标题和情况说明', icon: 'none' })
      return
    }
    this.setData({ submitting: true })
    api.post('/v1/merchant/safety/incidents', {
      category: 'MERCHANT_REPORT',
      severity: severityValues[this.data.severityIndex],
      title,
      description,
      batch_ids: [],
      order_ids: [],
      evidence_ids: []
    }).then(() => {
      this.setData({ submitting: false, showForm: false })
      wx.showToast({ title: '已提交平台', icon: 'success' })
      this.load()
    }).catch((error) => {
      this.setData({ submitting: false })
      wx.showToast({ title: error.message || '提交失败', icon: 'none' })
    })
  }
})
