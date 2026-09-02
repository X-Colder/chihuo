export type ConsumerView = 'overview' | 'demands' | 'publish' | 'presale' | 'orders'

export type Demand = {
  id: string
  title: string
  category: string
  location: string
  schedule: string
  target: number
  joined: number
  budget: string
  status: '聚合中' | '即将开售' | '已达标'
  tags: string[]
  description: string
  merchantHint: string
}

export type Product = {
  id: string
  title: string
  merchant: string
  price: number
  originalPrice: number
  grams: number
  stock: number
  delivery: string
  tags: string[]
  accent: string
  demandId: string
}

export type ConsumerOrder = {
  id: string
  product: string
  merchant: string
  quantity: number
  amount: number
  status: '待成团' | '制作中' | '配送中' | '已完成'
  time: string
  address: string
}

export type ConsumerSnapshot = {
  demands: Demand[]
  products: Product[]
  orders: ConsumerOrder[]
}
