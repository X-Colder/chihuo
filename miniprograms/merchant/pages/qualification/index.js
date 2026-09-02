function mapStatus(status) {
  if (status === 'APPROVED') return { text: '已通过', className: 'success' }
  if (status === 'REJECTED' || status === 'SUSPENDED') return { text: '需整改', className: 'muted-status' }
  return { text: '待复核', className: '' }
}

Page({
  data: {
    loading: true,
    error: '',
    profile: {
      id: '-',
      name: '加载中',
      statusText: '待复核',
      statusClass: ''
    },
    items: []
  },

  onShow() {
    this.load()
  },

  load() {
    const session = wx.getStorageSync('chihuo.merchant.session-user') || {}
    const status = mapStatus('PENDING')
    this.setData({
      loading: false,
      error: '',
      profile: {
        id: session.merchantId || session.merchant_id || '-',
        name: session.name ? `${session.name}的店` : '开发商家',
        statusText: status.text,
        statusClass: status.className
      },
      items: [
        {
          name: '营业执照',
          status: '待补充',
          statusClass: '',
          note: '主体名称和经营地址需要与实际门店一致'
        },
        {
          name: '食品经营许可证',
          status: '待复核',
          statusClass: '',
          note: '经营项目应覆盖实际提供的餐饮服务'
        },
        {
          name: '加工地址核验',
          status: '待现场核验',
          statusClass: '',
          note: '平台将按规则进行实质性核验'
        }
      ]
    })
  },

  showPlaceholder() {
    wx.showModal({
      title: '接入说明',
      content: '正式版本将接入证照上传、OCR 识别、官方信息比对和人工复核流程。',
      showCancel: false
    })
  }
})
