var STATUS_META = {
  UNASSIGNED: { text: '待领取', className: 'available' },
  ASSIGNED: { text: '待取餐', className: 'assigned' },
  PICKED_UP: { text: '待配送', className: 'pickup' },
  DELIVERING: { text: '配送中', className: 'delivering' },
  COMPLETED: { text: '已完成', className: 'completed' },
  CANCELLED: { text: '异常待处理', className: 'cancelled' }
}

var MOCK_TASKS = [
  {
    id: 'task-240901-01',
    orderId: 'CH202609020184',
    status: 'UNASSIGNED',
    pickupPoint: '南山区科技园科兴科学园 B3 栋一层',
    deliveryAddress: '南山区高新南七道 12 号 A 座 1603',
    note: '餐盒请保持水平，顾客不接受香菜。',
    storeName: '谷仓轻食 · 科技园店',
    customer: '林女士',
    phone: '138****1268',
    distance: '2.8 km',
    fee: 5,
    pickupWindow: '11:20 - 11:40',
    itemSummary: '低油鸡腿饭 × 2，350g / 份',
    createdAt: '09:46',
    updatedAt: '10:18'
  },
  {
    id: 'task-240901-02',
    orderId: 'CH202609020173',
    status: 'ASSIGNED',
    pickupPoint: '南山区后海大道 2088 号一层',
    deliveryAddress: '南山区海德三道 168 号 2 栋 1208',
    note: '到达后请放前台，订单尾号 173。',
    storeName: '小满家常菜 · 后海店',
    customer: '周先生',
    phone: '159****0341',
    distance: '4.6 km',
    fee: 7,
    pickupWindow: '11:05 - 11:25',
    itemSummary: '定量番茄牛腩饭 × 1，450g',
    createdAt: '09:22',
    updatedAt: '10:06'
  },
  {
    id: 'task-240901-03',
    orderId: 'CH202609020166',
    status: 'DELIVERING',
    pickupPoint: '南山区粤海街道创业路 28 号',
    deliveryAddress: '南山区文心五路 33 号 1 栋 901',
    note: '汤品易洒，请使用保温袋固定。',
    storeName: '一碗好汤 · 南山店',
    customer: '陈女士',
    phone: '186****7822',
    distance: '3.2 km',
    fee: 6,
    pickupWindow: '10:45 - 11:10',
    itemSummary: '少盐冬瓜排骨汤 × 1，600ml',
    createdAt: '08:58',
    updatedAt: '10:52'
  },
  {
    id: 'task-240901-04',
    orderId: 'CH202609020142',
    status: 'COMPLETED',
    pickupPoint: '南山区科技园科兴科学园 B3 栋一层',
    deliveryAddress: '南山区高新南环路 10 号 C 座 704',
    note: '已放置于前台冷藏柜。',
    storeName: '谷仓轻食 · 科技园店',
    customer: '王先生',
    phone: '137****9910',
    distance: '1.9 km',
    fee: 5,
    pickupWindow: '10:05 - 10:25',
    itemSummary: '低脂牛肉藜麦碗 × 1，400g',
    createdAt: '08:20',
    updatedAt: '10:38'
  },
  {
    id: 'task-240901-05',
    orderId: 'CH202609020131',
    status: 'CANCELLED',
    pickupPoint: '南山区后海大道 2088 号一层',
    deliveryAddress: '南山区中心路 3001 号 1 栋 1802',
    note: '顾客反馈外包装破损，等待平台处理。',
    storeName: '小满家常菜 · 后海店',
    customer: '赵女士',
    phone: '151****0448',
    distance: '5.1 km',
    fee: 8,
    pickupWindow: '09:40 - 10:00',
    itemSummary: '清淡虾仁蒸蛋 × 1，320g',
    createdAt: '08:05',
    updatedAt: '10:27'
  }
]

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function formatTime(value) {
  if (!value) {
    return '待确认'
  }
  if (/^\d{2}:\d{2}/.test(String(value))) {
    return String(value).slice(0, 5)
  }
  var date = new Date(value)
  if (isNaN(date.getTime())) {
    return String(value)
  }
  var hours = String(date.getHours()).padStart(2, '0')
  var minutes = String(date.getMinutes()).padStart(2, '0')
  return hours + ':' + minutes
}

function getAction(rawStatus) {
  var actions = {
    UNASSIGNED: { text: '领取任务', nextStatus: 'ASSIGNED', type: 'claim' },
    ASSIGNED: { text: '确认已取餐', nextStatus: 'PICKED_UP', type: 'advance' },
    PICKED_UP: { text: '开始配送', nextStatus: 'DELIVERING', type: 'advance' },
    DELIVERING: { text: '确认送达', nextStatus: 'COMPLETED', type: 'advance' }
  }
  return actions[rawStatus] || { text: '', nextStatus: '', type: '' }
}

function normalizeTask(task) {
  var rawStatus = task.status || task.rawStatus || 'UNASSIGNED'
  var meta = STATUS_META[rawStatus] || STATUS_META.UNASSIGNED
  var action = getAction(rawStatus)

  return {
    id: task.id,
    orderId: task.orderId || task.orderNo || '待生成',
    orderNo: task.orderNo || task.orderId || '待生成',
    orderTail: String(task.orderNo || task.orderId || '待生成').slice(-4),
    rawStatus: rawStatus,
    statusText: meta.text,
    statusClass: meta.className,
    pickupAddress: task.pickupAddress || task.pickupPoint || '待确认',
    dropoffAddress: task.dropoffAddress || task.deliveryAddress || '待确认',
    storeName: task.storeName || '合作商家',
    customer: task.customer || '消费者',
    phone: task.phone || '',
    distance: task.distance || '待计算',
    fee: Number(task.fee || 5),
    pickupWindow: task.pickupWindow || '集中配送时段',
    itemSummary: task.itemSummary || '预售餐品',
    notes: task.notes || task.note || '请核对餐品和批次信息。',
    createdAt: formatTime(task.createdAt),
    updatedAt: formatTime(task.updatedAt),
    actionText: action.text,
    actionType: action.type,
    nextStatus: action.nextStatus,
    isAvailable: rawStatus === 'UNASSIGNED',
    isActive: ['ASSIGNED', 'PICKED_UP', 'DELIVERING'].indexOf(rawStatus) !== -1,
    isCompleted: rawStatus === 'COMPLETED',
    isCancelled: rawStatus === 'CANCELLED'
  }
}

function getMockTasks() {
  return clone(MOCK_TASKS).map(normalizeTask)
}

function findMockTask(id) {
  var task = MOCK_TASKS.find(function (item) {
    return item.id === id
  })
  return task ? normalizeTask(clone(task)) : null
}

function getStatusMeta(status) {
  return STATUS_META[status] || STATUS_META.UNASSIGNED
}

module.exports = {
  STATUS_META: STATUS_META,
  normalizeTask: normalizeTask,
  getMockTasks: getMockTasks,
  findMockTask: findMockTask,
  getStatusMeta: getStatusMeta,
  getAction: getAction
}
