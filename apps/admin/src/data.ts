export type AdminView = 'dashboard' | 'merchants' | 'review' | 'orders' | 'safety' | 'audit'
export type MerchantStatus = '待审核' | '已通过' | '需补充' | '已暂停'
export type ReviewStatus = '待审核' | '需补充' | '已通过' | '已驳回'
export type ReviewKind = '需求' | '商品'
export type OrderStatus = '配送中' | '已完成' | '售后中' | '已退款'
export type SafetySeverity = '高' | '中' | '低'
export type SafetyStatus = '待处理' | '调查中' | '已结案'

export type Merchant = {
  id: string
  name: string
  owner: string
  phone: string
  area: string
  category: string
  status: MerchantStatus
  licenseNo: string
  licenseExpire: string
  appliedAt: string
  products: number
  risk: '低' | '中' | '高'
  address: string
  documents: string[]
}

export type ReviewItem = {
  id: string
  kind: ReviewKind
  title: string
  merchant: string
  summary: string
  submittedAt: string
  status: ReviewStatus
  signals: string[]
  price?: string
  quantity?: string
}

export type AdminOrder = {
  id: string
  orderNo: string
  merchant: string
  customer: string
  amount: number
  status: OrderStatus
  issue?: string
  createdAt: string
  delivery: string
}

export type SafetyEvent = {
  id: string
  title: string
  merchant: string
  orderNo: string
  severity: SafetySeverity
  status: SafetyStatus
  reportedAt: string
  reporter: string
  summary: string
  actions: string[]
}

export type AuditLog = {
  id: string
  actor: string
  action: string
  target: string
  result: '成功' | '待复核' | '失败'
  ip: string
  createdAt: string
}

export const mockMerchants: Merchant[] = [
  {
    id: 'merchant-001',
    name: '谷仓轻食 · 科技园店',
    owner: '深圳谷仓餐饮有限公司',
    phone: '0755-8621-4088',
    area: '南山区 · 科技园',
    category: '餐饮服务经营者',
    status: '待审核',
    licenseNo: 'JY24403050018352',
    licenseExpire: '2027-04-19',
    appliedAt: '今天 09:42',
    products: 12,
    risk: '低',
    address: '深圳市南山区科兴科学园 B3 栋一层 101',
    documents: ['营业执照', '食品经营许可证', '后厨现场照片', '负责人身份证明'],
  },
  {
    id: 'merchant-002',
    name: '小满家常菜 · 后海店',
    owner: '深圳市小满餐饮管理有限公司',
    phone: '0755-8699-1026',
    area: '南山区 · 后海',
    category: '餐饮服务经营者',
    status: '需补充',
    licenseNo: 'JY24403050029746',
    licenseExpire: '2026-11-02',
    appliedAt: '昨天 16:20',
    products: 8,
    risk: '中',
    address: '深圳市南山区后海大道 2088 号一层',
    documents: ['营业执照', '食品经营许可证', '负责人身份证明'],
  },
  {
    id: 'merchant-003',
    name: '一碗好汤 · 南山店',
    owner: '深圳一碗好汤餐饮有限公司',
    phone: '0755-2673-8810',
    area: '南山区 · 粤海',
    category: '餐饮服务经营者',
    status: '已通过',
    licenseNo: 'JY24403050031468',
    licenseExpire: '2028-06-28',
    appliedAt: '2026-08-27',
    products: 16,
    risk: '低',
    address: '深圳市南山区创业路 28 号一层',
    documents: ['营业执照', '食品经营许可证', '后厨现场照片', '负责人身份证明'],
  },
  {
    id: 'merchant-004',
    name: '禾下厨房 · 福田店',
    owner: '深圳禾下厨房有限公司',
    phone: '0755-8382-6199',
    area: '福田区 · 车公庙',
    category: '餐饮服务经营者',
    status: '已暂停',
    licenseNo: 'JY24403040010117',
    licenseExpire: '2026-08-31',
    appliedAt: '2026-08-18',
    products: 4,
    risk: '高',
    address: '深圳市福田区泰然九路 12 号',
    documents: ['营业执照', '食品经营许可证'],
  },
]

export const mockReviews: ReviewItem[] = [
  {
    id: 'review-001',
    kind: '需求',
    title: '低油定量午餐 · 科技园工作日午餐',
    merchant: '谷仓轻食 · 科技园店',
    summary: '基础份 350g，少油，不含花生，11:30-12:30 配送。',
    submittedAt: '今天 10:06',
    status: '待审核',
    signals: ['34 人已确认', '预计客单 ¥24', '区域集中度 86%'],
    quantity: '34 份',
  },
  {
    id: 'review-002',
    kind: '商品',
    title: '少盐冬瓜排骨汤 · 600ml',
    merchant: '一碗好汤 · 南山店',
    summary: '冬瓜、排骨、玉米；少盐版本，冷藏配送，建议 4 小时内食用。',
    submittedAt: '今天 09:31',
    status: '待审核',
    signals: ['配料已填写', '批次规则完整', '支持明厨亮灶'],
    price: '¥18 / 份',
  },
  {
    id: 'review-003',
    kind: '商品',
    title: '低脂牛肉藜麦碗 · 400g',
    merchant: '谷仓轻食 · 科技园店',
    summary: '牛肉、藜麦、西兰花；独立酱汁，标注过敏原和称重范围。',
    submittedAt: '昨天 18:52',
    status: '待审核',
    signals: ['配料已填写', '图片待补充', '包装方式已确认'],
    price: '¥29 / 份',
  },
  {
    id: 'review-004',
    kind: '需求',
    title: '不含海鲜的家庭晚餐套餐',
    merchant: '用户共创 · 宝安中心',
    summary: '家庭 3-4 人份，预算 ¥86 以内，周五 18:00 前送达。',
    submittedAt: '昨天 15:18',
    status: '已通过',
    signals: ['22 人已确认', '已匹配 2 家商户', '预售已开启'],
    quantity: '18 组',
  },
]

export const mockOrders: AdminOrder[] = [
  { id: 'order-001', orderNo: 'CH202609020184', merchant: '谷仓轻食 · 科技园店', customer: '林女士', amount: 53, status: '配送中', createdAt: '10:18', delivery: '骑手 林师傅' },
  { id: 'order-002', orderNo: 'CH202609020173', merchant: '小满家常菜 · 后海店', customer: '周先生', amount: 31, status: '配送中', createdAt: '10:06', delivery: '骑手 待接单' },
  { id: 'order-003', orderNo: 'CH202609020166', merchant: '一碗好汤 · 南山店', customer: '陈女士', amount: 24, status: '售后中', issue: '顾客反馈汤盒外包装破损', createdAt: '09:52', delivery: '骑手 林师傅' },
  { id: 'order-004', orderNo: 'CH202609020142', merchant: '谷仓轻食 · 科技园店', customer: '王先生', amount: 34, status: '已完成', createdAt: '09:41', delivery: '骑手 林师傅' },
  { id: 'order-005', orderNo: 'CH202609020131', merchant: '小满家常菜 · 后海店', customer: '赵女士', amount: 28, status: '售后中', issue: '顾客反馈餐品与需求规格不符', createdAt: '09:26', delivery: '骑手 林师傅' },
  { id: 'order-006', orderNo: 'CH202609020118', merchant: '一碗好汤 · 南山店', customer: '李先生', amount: 19, status: '已退款', issue: '商家出餐超时，订单取消', createdAt: '09:03', delivery: '平台取消' },
]

export const mockSafetyEvents: SafetyEvent[] = [
  {
    id: 'safety-001',
    title: '疑似包装破损导致汤品渗漏',
    merchant: '一碗好汤 · 南山店',
    orderNo: 'CH202609020166',
    severity: '中',
    status: '调查中',
    reportedAt: '今天 10:01',
    reporter: '消费者 陈女士',
    summary: '消费者上传外包装照片，反馈汤盒边缘渗漏，暂未反馈身体不适。',
    actions: ['已暂停该订单售后结算', '已通知商家保留涉事批次', '等待商家补充包装记录'],
  },
  {
    id: 'safety-002',
    title: '商家食品经营许可证已过期',
    merchant: '禾下厨房 · 福田店',
    orderNo: '店铺级事件',
    severity: '高',
    status: '待处理',
    reportedAt: '今天 08:46',
    reporter: '资质自动巡检',
    summary: '系统比对发现许可证有效期为 2026 年 8 月 31 日，当前店铺仍有 4 个在售商品。',
    actions: ['已暂停店铺交易', '已隐藏全部商品', '等待商家提交新证照'],
  },
  {
    id: 'safety-003',
    title: '商品“低油鸡腿饭”配料信息缺少过敏原',
    merchant: '谷仓轻食 · 科技园店',
    orderNo: '商品级事件',
    severity: '低',
    status: '待处理',
    reportedAt: '昨天 17:26',
    reporter: '商品审核员 唐宁',
    summary: '商品描述提到芝麻酱，但过敏原字段未标注芝麻及其制品。',
    actions: ['已退回商品编辑', '已保留当前页面快照'],
  },
]

export const mockAuditLogs: AuditLog[] = [
  { id: 'log-001', actor: '唐宁 · 审核员', action: '退回商品审核', target: '低油鸡腿饭 · 谷仓轻食', result: '待复核', ip: '10.18.4.23', createdAt: '今天 10:16' },
  { id: 'log-002', actor: '系统 · 资质巡检', action: '暂停商家交易', target: '禾下厨房 · 福田店', result: '成功', ip: 'SYSTEM', createdAt: '今天 08:46' },
  { id: 'log-003', actor: '林雪 · 客服', action: '发起订单退款', target: 'CH202609020118', result: '成功', ip: '10.18.8.12', createdAt: '今天 09:07' },
  { id: 'log-004', actor: '周衡 · 管理员', action: '更新商家审核状态', target: '一碗好汤 · 南山店', result: '成功', ip: '10.18.1.6', createdAt: '昨天 16:34' },
  { id: 'log-005', actor: 'AI OCR · 证照识别', action: '识别营业执照字段', target: '谷仓轻食 · 科技园店', result: '待复核', ip: 'SYSTEM', createdAt: '昨天 15:52' },
]
