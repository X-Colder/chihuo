var request = require('../../utils/request')
var taskUtils = require('../../utils/task')

Page({
  data: {
    task: null,
    loading: true,
    submitting: false,
    isMock: false,
    error: '',
    issueVisible: false,
    issueCategory: '商家出餐延迟',
    issueDescription: '',
    issueCategories: ['商家出餐延迟', '顾客无法联系', '地址无法送达', '餐品或包装破损', '其他异常']
  },

  onLoad: function (options) {
    this.taskId = options && options.id ? decodeURIComponent(options.id) : ''
    this.loadTask()
  },

  loadTask: function () {
    var page = this
    if (!this.taskId) {
      this.setData({
        loading: false,
        error: '缺少任务编号，请返回任务列表重新进入。'
      })
      return
    }

    this.setData({ loading: true, error: '' })
    Promise.all([
      request.get('/v1/rider/tasks'),
      request.get('/v1/rider/tasks/queue')
    ])
      .then(function (responses) {
        var tasks = (responses[0] || []).concat(responses[1] || [])
        var rawTask = tasks.find(function (item) {
          return item && item.id === page.taskId
        })
        if (!rawTask) {
          throw new Error('任务不存在')
        }
        page.setData({
          task: taskUtils.normalizeTask(rawTask),
          loading: false,
          isMock: false
        })
      })
      .catch(function () {
        var mockTask = taskUtils.findMockTask(page.taskId)
        page.setData({
          task: mockTask,
          loading: false,
          isMock: true,
          error: mockTask ? '当前使用演示数据，状态操作会先在本地展示。' : '没有找到这个任务。'
        })
      })
  },

  onBack: function () {
    wx.navigateBack()
  },

  onPrimaryAction: function () {
    var task = this.data.task
    if (!task || !task.actionType || this.data.submitting) {
      return
    }

    if (task.actionType === 'advance' && task.nextStatus === 'COMPLETED') {
      var page = this
      wx.showModal({
        title: '确认送达',
        content: '请确认餐品已交付给顾客或放置在约定位置。',
        confirmText: '确认送达',
        success: function (result) {
          if (result.confirm) {
            page.submitStatus(task.nextStatus)
          }
        }
      })
      return
    }

    this.submitStatus(task.nextStatus)
  },

  submitStatus: function (nextStatus) {
    var page = this
    var task = this.data.task
    if (!task) {
      return
    }

    this.setData({ submitting: true })
    var action = task.actionType === 'claim'
      ? request.post('/v1/rider/tasks/' + task.id + '/claim')
      : request.patch('/v1/rider/tasks/' + task.id, { status: nextStatus })

    action
      .then(function (updated) {
        page.updateTask(updated, false)
        wx.showToast({ title: '状态已更新', icon: 'success' })
      })
      .catch(function () {
        var localTask = Object.assign({}, task, {
          rawStatus: nextStatus,
          status: nextStatus
        })
        page.updateTask(localTask, true)
        wx.showToast({ title: '演示状态已更新', icon: 'none' })
      })
      .then(function () {
        page.setData({ submitting: false })
        page.markListForRefresh()
      })
  },

  updateTask: function (task, isMock) {
    this.setData({
      task: taskUtils.normalizeTask(task),
      isMock: isMock,
      error: isMock ? '接口暂不可用，当前仅展示本地状态。' : ''
    })
  },

  markListForRefresh: function () {
    var pages = getCurrentPages()
    var previous = pages.length > 1 ? pages[pages.length - 2] : null
    if (previous) {
      previous.pendingTask = this.data.task
      previous.shouldRefresh = !this.data.isMock
    }
  },

  onOpenIssue: function () {
    if (!this.data.task || this.data.submitting) {
      return
    }
    this.setData({
      issueVisible: true,
      issueDescription: ''
    })
  },

  onCloseIssue: function () {
    if (!this.data.submitting) {
      this.setData({ issueVisible: false })
    }
  },

  onIssueCategory: function (event) {
    this.setData({
      issueCategory: event.currentTarget.dataset.category
    })
  },

  onIssueInput: function (event) {
    this.setData({
      issueDescription: event.detail.value
    })
  },

  onSubmitIssue: function () {
    var page = this
    var task = this.data.task
    if (!task || this.data.submitting) {
      return
    }

    var description = (this.data.issueDescription || '').trim()
    var note = this.data.issueCategory + '：' + (description || '骑手提交异常，请平台协助处理。')
    this.setData({ submitting: true })
    request.patch('/v1/rider/tasks/' + task.id, {
      status: 'CANCELLED',
      note: note
    })
      .then(function (updated) {
        page.updateTask(updated, false)
        wx.showToast({ title: '异常已上报', icon: 'success' })
      })
      .catch(function () {
        page.updateTask(Object.assign({}, task, {
          rawStatus: 'CANCELLED',
          status: 'CANCELLED',
          note: note,
          notes: note
        }), true)
        wx.showToast({ title: '异常已记录', icon: 'none' })
      })
      .then(function () {
        page.setData({
          issueVisible: false,
          submitting: false
        })
        page.markListForRefresh()
      })
  },

  noop: function () {}
})
