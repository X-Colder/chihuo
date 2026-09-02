export type TaskStatus = '待取餐' | '待配送' | '配送中' | '已完成' | '异常待处理'

export type RiderTask = {
  id: string
  orderNo: string
  storeName: string
  pickupAddress: string
  dropoffAddress: string
  customer: string
  phone: string
  distance: string
  fee: number
  status: TaskStatus
  pickupWindow: string
  itemSummary: string
  notes: string
  createdAt: string
  updatedAt: string
}

export const mockTasks: RiderTask[] = [
  {
    id: 'task-240901-01',
    orderNo: 'CH202609020184',
    storeName: '谷仓轻食 · 科技园店',
    pickupAddress: '南山区科技园科兴科学园 B3 栋一层',
    dropoffAddress: '南山区高新南七道 12 号 A 座 1603',
    customer: '林女士',
    phone: '138****1268',
    distance: '2.8 km',
    fee: 5,
    status: '待取餐',
    pickupWindow: '11:20 - 11:40',
    itemSummary: '低油鸡腿饭 × 2，350g / 份',
    notes: '餐盒请保持水平，顾客不接受香菜。',
    createdAt: '09:46',
    updatedAt: '10:18',
  },
  {
    id: 'task-240901-02',
    orderNo: 'CH202609020173',
    storeName: '小满家常菜 · 后海店',
    pickupAddress: '南山区后海大道 2088 号一层',
    dropoffAddress: '南山区海德三道 168 号 2 栋 1208',
    customer: '周先生',
    phone: '159****0341',
    distance: '4.6 km',
    fee: 7,
    status: '待配送',
    pickupWindow: '11:05 - 11:25',
    itemSummary: '定量番茄牛腩饭 × 1，450g',
    notes: '到达后请放前台，订单尾号 173。',
    createdAt: '09:22',
    updatedAt: '10:06',
  },
  {
    id: 'task-240901-03',
    orderNo: 'CH202609020166',
    storeName: '一碗好汤 · 南山店',
    pickupAddress: '南山区粤海街道创业路 28 号',
    dropoffAddress: '南山区文心五路 33 号 1 栋 901',
    customer: '陈女士',
    phone: '186****7822',
    distance: '3.2 km',
    fee: 6,
    status: '配送中',
    pickupWindow: '10:45 - 11:10',
    itemSummary: '少盐冬瓜排骨汤 × 1，600ml',
    notes: '汤品易洒，请使用保温袋固定。',
    createdAt: '08:58',
    updatedAt: '10:52',
  },
  {
    id: 'task-240901-04',
    orderNo: 'CH202609020142',
    storeName: '谷仓轻食 · 科技园店',
    pickupAddress: '南山区科技园科兴科学园 B3 栋一层',
    dropoffAddress: '南山区高新南环路 10 号 C 座 704',
    customer: '王先生',
    phone: '137****9910',
    distance: '1.9 km',
    fee: 5,
    status: '已完成',
    pickupWindow: '10:05 - 10:25',
    itemSummary: '低脂牛肉藜麦碗 × 1，400g',
    notes: '已放置于前台冷藏柜。',
    createdAt: '08:20',
    updatedAt: '10:38',
  },
  {
    id: 'task-240901-05',
    orderNo: 'CH202609020131',
    storeName: '小满家常菜 · 后海店',
    pickupAddress: '南山区后海大道 2088 号一层',
    dropoffAddress: '南山区中心路 3001 号 1 栋 1802',
    customer: '赵女士',
    phone: '151****0448',
    distance: '5.1 km',
    fee: 8,
    status: '异常待处理',
    pickupWindow: '09:40 - 10:00',
    itemSummary: '清淡虾仁蒸蛋 × 1，320g',
    notes: '顾客反馈外包装破损，等待平台处理。',
    createdAt: '08:05',
    updatedAt: '10:27',
  },
]

export const statusMeta: Record<TaskStatus, { tone: string; label: string }> = {
  待取餐: { tone: 'amber', label: '待取餐' },
  待配送: { tone: 'blue', label: '待配送' },
  配送中: { tone: 'teal', label: '配送中' },
  已完成: { tone: 'green', label: '已完成' },
  异常待处理: { tone: 'red', label: '异常' },
}
