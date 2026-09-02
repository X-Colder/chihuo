var request = require('../../utils/request')
var taskUtils = require('../../utils/task')

Page({
  data: {
    tasks: [],
    visibleTasks: [],
    filters: [],
    activeFilter: 'ALL',
    loading: true,
    refreshing: false,
    error: '',
    isMock: false,
    empty: false,
    activeCount: 0,
    completedCount: 0,
    income: '0.00'
  },

  onLoad: function () {
    this.loadTasks(false)
  },

  onShow: function () {
    if (this.pendingTask) {
      this.applyTasks(
        this.data.tasks.map(function (task) {
          return task.id === this.pendingTask.id ? this.pendingTask : task
        }, this),
        this.data.isMock
      )
      this.pendingTask = null
    }
    if (this.hasLoaded && this.shouldRefresh) {
      this.shouldRefresh = false
      this.loadTasks(true)
    }
  },

  onPullDownRefresh: function () {
    this.loadTasks(true)
  },

  loadTasks: function (silent) {
    var page = this
    if (silent) {
      this.setData({ refreshing: true, error: '' })
    } else {
      this.setData({ loading: true, error: '' })
    }

    Promise.all([
      request.get('/v1/rider/tasks'),
      request.get('/v1/rider/tasks/queue')
    ])
      .then(function (responses) {
        var tasks = page.mergeTasks(responses[0], responses[1])
        page.applyTasks(tasks, false)
      })
      .catch(function () {
        page.applyTasks(taskUtils.getMockTasks(), true)
        page.setData({ error: '当前使用演示数据，接口恢复后可查看真实任务。' })
      })
      .then(function () {
        page.hasLoaded = true
        page.setData({
          loading: false,
          refreshing: false,
          empty: page.data.visibleTasks.length === 0
        })
        wx.stopPullDownRefresh()
      })
  },

  mergeTasks: function (assigned, queue) {
    var all = []
    var seen = {}
    ;(assigned || []).concat(queue || []).forEach(function (item) {
      if (!item || !item.id || seen[item.id]) {
        return
      }
      seen[item.id] = true
      all.push(taskUtils.normalizeTask(item))
    })
    return all
  },

  applyTasks: function (tasks, isMock) {
    var normalized = (tasks || []).map(function (task) {
      return task.rawStatus ? task : taskUtils.normalizeTask(task)
    })
    var filters = this.buildFilters(normalized)
    var activeCount = normalized.filter(function (task) {
      return task.isActive
    }).length
    var completed = normalized.filter(function (task) {
      return task.isCompleted
    })
    var income = completed.reduce(function (sum, task) {
      return sum + Number(task.fee || 0)
    }, 0)

    this.setData({
      tasks: normalized,
      filters: filters,
      isMock: isMock,
      activeCount: activeCount,
      completedCount: completed.length,
      income: income.toFixed(2)
    })
    this.applyFilter(this.data.activeFilter, normalized, filters)
  },

  buildFilters: function (tasks) {
    var count = function (predicate) {
      return tasks.filter(predicate).length
    }
    return [
      { key: 'ALL', label: '全部', count: tasks.length },
      { key: 'AVAILABLE', label: '待领取', count: count(function (task) { return task.isAvailable }) },
      { key: 'ACTIVE', label: '进行中', count: count(function (task) { return task.isActive }) },
      { key: 'COMPLETED', label: '已完成', count: count(function (task) { return task.isCompleted }) },
      { key: 'CANCELLED', label: '异常', count: count(function (task) { return task.isCancelled }) }
    ]
  },

  applyFilter: function (filter, tasks, filters) {
    var source = tasks || this.data.tasks
    var result = source
    if (filter === 'AVAILABLE') {
      result = source.filter(function (task) { return task.isAvailable })
    } else if (filter === 'ACTIVE') {
      result = source.filter(function (task) { return task.isActive })
    } else if (filter === 'COMPLETED') {
      result = source.filter(function (task) { return task.isCompleted })
    } else if (filter === 'CANCELLED') {
      result = source.filter(function (task) { return task.isCancelled })
    }
    this.setData({
      activeFilter: filter,
      visibleTasks: result,
      empty: !this.data.loading && result.length === 0,
      filters: filters || this.data.filters
    })
  },

  onFilterTap: function (event) {
    this.applyFilter(event.currentTarget.dataset.filter)
  },

  onTaskTap: function (event) {
    var id = event.currentTarget.dataset.id
    if (!id) {
      return
    }
    wx.navigateTo({
      url: '/pages/task-detail/index?id=' + encodeURIComponent(id)
    })
  },

  onRetry: function () {
    this.loadTasks(false)
  }
})
