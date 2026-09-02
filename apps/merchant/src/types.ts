export type MerchantView = 'dashboard' | 'demand-hall' | 'quotes' | 'production' | 'qualification' | 'safety'

export type MerchantDemand = {
  id: string
  title: string
  category: string
  location: string
  schedule: string
  joined: number
  target: number
  budget: string
  deadline: string
  tags: string[]
  fit: '高匹配' | '可尝试' | '需评估'
  fitNote: string
  status: '待报价' | '已报价' | '已转预售'
}

export type MerchantQuote = {
  id: string
  demandId: string
  demandTitle: string
  submittedAt: string
  price: number
  grams: number
  minOrder: number
  status: '审核中' | '已入选' | '未入选'
  note: string
}

export type ProductionOrder = {
  id: string
  product: string
  location: string
  quantity: number
  paid: number
  status: '待生产' | '制作中' | '待交接' | '已完成'
  window: string
  batch: string
  note: string
}

export type Qualification = {
  name: string
  status: '已通过' | '待复核' | '待补充'
  expires: string
  updated: string
}

export type SafetyRecord = {
  id: string
  type: '日常检查' | '投诉跟进' | '批次留档' | '整改记录'
  title: string
  date: string
  owner: string
  status: '已完成' | '处理中' | '待补充'
  detail: string
}

export type MerchantSnapshot = {
  demands: MerchantDemand[]
  quotes: MerchantQuote[]
  productionOrders: ProductionOrder[]
  qualifications: Qualification[]
  safetyRecords: SafetyRecord[]
}
