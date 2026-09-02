import { mockConsumerSnapshot } from '../mock'
import type { ConsumerOrder, ConsumerSnapshot, Demand, Product } from '../types'

const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:4000').replace(/\/$/, '')
const tokenKey = 'chihuo.consumer.demo-token'

type ApiEnvelope<T> = { data?: T; error?: { message?: string } }
type BackendDemand = {
  id: string
  title: string
  category: string
  serviceArea: string
  servingDate: string
  servingTime: string
  budgetMinCents: number
  budgetMaxCents: number
  hardConstraints: string[]
  preferences: string[]
  minimumMembers: number
  memberCount: number
  status: string
  notes?: string
}
type BackendCampaign = {
  id: string
  demandId: string
  title: string
  unitPriceCents: number
  currentOrders: number
  maximumOrders: number
  pickupPoint: string
  startsAt: string
  foodSpec: {
    weightGrams: number
    allergens: string[]
    oilLevel: string
    saltLevel: string
  }
}
type BackendOrder = {
  id: string
  campaignId: string
  status: string
  quantity: number
  totalCents: number
  deliveryAddress: string
  createdAt: string
}

async function ensureToken() {
  const existing = window.localStorage.getItem(tokenKey)
  if (existing) return existing
  const response = await fetch(`${apiBaseUrl}/v1/auth/demo-login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: '演示消费者', role: 'CONSUMER' }),
    signal: AbortSignal.timeout(1600),
  })
  if (!response.ok) throw new Error(`登录失败（${response.status}）`)
  const payload = (await response.json()) as ApiEnvelope<{ token: string }>
  if (!payload.data?.token) throw new Error('登录响应缺少 token')
  window.localStorage.setItem(tokenKey, payload.data.token)
  return payload.data.token
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await ensureToken()
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...(init?.headers ?? {}),
    },
    signal: AbortSignal.timeout(3000),
  })
  if (!response.ok) {
    if (response.status === 401) window.localStorage.removeItem(tokenKey)
    const payload = (await response.json().catch(() => ({}))) as ApiEnvelope<unknown>
    throw new Error(payload.error?.message ?? `接口请求失败（${response.status}）`)
  }
  const payload = (await response.json()) as ApiEnvelope<T>
  return (payload.data ?? payload) as T
}

function moneyRange(min: number, max: number) {
  return `¥${(min / 100).toFixed(0)}–${(max / 100).toFixed(0)}`
}

function mapDemand(item: BackendDemand): Demand {
  const status: Demand['status'] = item.status === 'READY' || item.status === 'CLOSED' ? '已达标' : '聚合中'
  return {
    id: item.id,
    title: item.title,
    category: item.category,
    location: item.serviceArea,
    schedule: `${item.servingDate.slice(5)} ${item.servingTime}`,
    target: item.minimumMembers,
    joined: item.memberCount,
    budget: moneyRange(item.budgetMinCents, item.budgetMaxCents),
    status,
    tags: [...item.hardConstraints, ...item.preferences],
    description: item.notes ?? '平台正在聚合相近需求，并将规格整理给附近商家。',
    merchantHint: status === '已达标' ? '需求已达标，正在等待商家提交最终规格和价格。' : '达到人数后会进入商家报价阶段。',
  }
}

function mapCampaign(item: BackendCampaign, index: number): Product {
  return {
    id: item.id,
    title: item.title,
    merchant: `合作商家 · ${item.pickupPoint}`,
    price: item.unitPriceCents / 100,
    originalPrice: item.unitPriceCents / 100,
    grams: item.foodSpec.weightGrams,
    stock: Math.max(0, item.maximumOrders - item.currentOrders),
    delivery: `${item.startsAt.slice(5, 10)} 集中配送`,
    tags: [
      item.foodSpec.oilLevel === 'LOW' ? '低油' : '',
      item.foodSpec.saltLevel === 'LOW' ? '少盐' : '',
      item.foodSpec.allergens.length ? `含${item.foodSpec.allergens.join('、')}` : '配料可查',
    ].filter(Boolean),
    accent: ['sage', 'apricot', 'blue'][index % 3],
    demandId: item.demandId,
  }
}

function mapOrder(item: BackendOrder): ConsumerOrder {
  const statusMap: Record<string, ConsumerOrder['status']> = {
    PENDING_PAYMENT: '待成团',
    PAID: '待成团',
    ACCEPTED: '制作中',
    PREPARING: '制作中',
    READY_FOR_PICKUP: '制作中',
    PICKED_UP: '配送中',
    DELIVERING: '配送中',
    DELIVERED: '已完成',
    CANCELLED: '待成团',
    REFUNDED: '待成团',
  }
  return {
    id: item.id,
    product: `预售餐品 ${item.campaignId.slice(0, 8)}`,
    merchant: '合作商家',
    quantity: item.quantity,
    amount: item.totalCents / 100,
    status: statusMap[item.status] ?? '待成团',
    time: new Date(item.createdAt).toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }),
    address: item.deliveryAddress,
  }
}

export type LoadResult = {
  data: ConsumerSnapshot
  isFallback: boolean
  message?: string
}

export async function loadConsumerSnapshot(): Promise<LoadResult> {
  try {
    const [demands, campaigns, orders] = await Promise.all([
      request<BackendDemand[]>('/v1/demands'),
      request<BackendCampaign[]>('/v1/campaigns'),
      request<BackendOrder[]>('/v1/orders'),
    ])
    return {
      data: {
        demands: demands.map(mapDemand),
        products: campaigns.map(mapCampaign),
        orders: orders.map(mapOrder),
      },
      isFallback: false,
    }
  } catch (error) {
    return {
      data: structuredClone(mockConsumerSnapshot),
      isFallback: true,
      message: error instanceof Error ? error.message : '接口暂不可用，已使用演示数据',
    }
  }
}

export async function createDemand(payload: Omit<Demand, 'id' | 'joined' | 'status' | 'merchantHint'>) {
  try {
    const budget = payload.budget.match(/\d+/g)?.map(Number) ?? [20, 30]
    const response = await request<{ demand: BackendDemand }>('/v1/demands', {
      method: 'POST',
      body: JSON.stringify({
        title: payload.title,
        category: payload.category,
        serviceArea: payload.location,
        servingDate: new Date().toISOString().slice(0, 10),
        servingTime: '11:30',
        budgetMinCents: budget[0] * 100,
        budgetMaxCents: (budget[1] ?? budget[0]) * 100,
        quantity: 1,
        weightMinGrams: 300,
        weightMaxGrams: 400,
        hardConstraints: payload.tags,
        preferences: [],
        notes: payload.description,
        minimumMembers: payload.target,
        maximumMembers: payload.target + 20,
      }),
    })
    return mapDemand(response.demand)
  } catch {
    return {
      ...payload,
      id: `demand-local-${Date.now()}`,
      joined: 1,
      status: '聚合中' as const,
      merchantHint: '你的需求已进入聚合，达到人数后会通知你。',
    }
  }
}

export async function joinDemand(demand: Demand) {
  try {
    const response = await request<{ demand: BackendDemand }>(`/v1/demands/${demand.id}/join`, {
      method: 'POST',
      body: JSON.stringify({ quantity: 1, preferences: [] }),
    })
    return mapDemand(response.demand)
  } catch {
    return { ...demand, joined: Math.min(demand.target, demand.joined + 1) }
  }
}

export async function placeOrder(product: Product, quantity: number): Promise<ConsumerOrder> {
  try {
    const response = await request<BackendOrder>(`/v1/campaigns/${product.id}/orders`, {
      method: 'POST',
      body: JSON.stringify({
        quantity,
        deliveryAddress: '科技园 A 区 2 号楼前台',
        contactName: '演示消费者',
        contactPhone: '13800000000',
      }),
    })
    const paid = await request<BackendOrder>(`/v1/orders/${response.id}/pay`, { method: 'POST' })
    return mapOrder(paid)
  } catch {
    return {
      id: `CH-${new Date().toISOString().slice(0, 10).replace(/-/g, '')}-${Math.floor(Math.random() * 90 + 10)}`,
      product: product.title,
      merchant: product.merchant,
      quantity,
      amount: product.price * quantity + 3.5,
      status: '待成团',
      time: product.delivery,
      address: '科技园 A 区 2 号楼前台',
    }
  }
}
