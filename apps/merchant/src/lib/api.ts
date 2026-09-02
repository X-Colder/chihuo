import { mockMerchantSnapshot } from '../mock'
import type { MerchantDemand, MerchantQuote, MerchantSnapshot, SafetyRecord } from '../types'

const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:4000').replace(/\/$/, '')
const tokenKey = 'chihuo.merchant.demo-token'
const merchantKey = 'chihuo.merchant.id'

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
}
type BackendOffer = {
  id: string
  demandId: string
  unitPriceCents: number
  productionCapacity: number
  weightGrams: number
  status: string
  createdAt: string
  storageInstructions: string
}
type BackendCampaign = {
  id: string
  demandId: string
  title: string
  currentOrders: number
  maximumOrders: number
  unitPriceCents: number
  status: string
  pickupPoint: string
  startsAt: string
  foodSpec: { weightGrams: number; oilLevel: string }
}
type BackendMerchant = { id: string; name: string; status: string; license: Record<string, unknown> }
type BackendIncident = { id: string; title: string; description: string; status: string; createdAt: string }

async function ensureSession() {
  const existing = window.localStorage.getItem(tokenKey)
  if (existing) return { token: existing, merchantId: window.localStorage.getItem(merchantKey) ?? '' }
  const response = await fetch(`${apiBaseUrl}/v1/auth/demo-login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: '演示商家', role: 'MERCHANT', merchantName: '吃货演示餐厅' }),
    signal: AbortSignal.timeout(1600),
  })
  if (!response.ok) throw new Error(`登录失败（${response.status}）`)
  const payload = (await response.json()) as ApiEnvelope<{ token: string; user: { merchantId?: string } }>
  if (!payload.data?.token) throw new Error('登录响应缺少 token')
  window.localStorage.setItem(tokenKey, payload.data.token)
  window.localStorage.setItem(merchantKey, payload.data.user.merchantId ?? '')
  return { token: payload.data.token, merchantId: payload.data.user.merchantId ?? '' }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const session = await ensureSession()
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.token}`,
      ...(init?.headers ?? {}),
    },
    signal: AbortSignal.timeout(3000),
  })
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as ApiEnvelope<unknown>
    throw new Error(payload.error?.message ?? `接口请求失败（${response.status}）`)
  }
  const payload = (await response.json()) as ApiEnvelope<T>
  return (payload.data ?? payload) as T
}

function budget(min: number, max: number) {
  return `¥${(min / 100).toFixed(0)}–${(max / 100).toFixed(0)}`
}

function mapDemand(item: BackendDemand, offers: BackendOffer[], campaigns: BackendCampaign[]): MerchantDemand {
  const hasOffer = offers.some((offer) => offer.demandId === item.id)
  const hasCampaign = campaigns.some((campaign) => campaign.demandId === item.id)
  return {
    id: item.id,
    title: item.title,
    category: item.category,
    location: item.serviceArea,
    schedule: `${item.servingDate.slice(5)} ${item.servingTime}`,
    joined: item.memberCount,
    target: item.minimumMembers,
    budget: budget(item.budgetMinCents, item.budgetMaxCents),
    deadline: item.status === 'READY' ? '已达标 · 等待生产方案' : '需求持续聚合中',
    tags: [...item.hardConstraints, ...item.preferences],
    fit: '高匹配',
    fitNote: '平台根据区域、时段和餐品规格为你匹配了这条需求。',
    status: hasCampaign ? '已转预售' : hasOffer ? '已报价' : '待报价',
  }
}

function mapQuote(item: BackendOffer, demandTitle: string): MerchantQuote {
  return {
    id: item.id,
    demandId: item.demandId,
    demandTitle,
    submittedAt: new Date(item.createdAt).toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }),
    price: item.unitPriceCents / 100,
    grams: item.weightGrams,
    minOrder: item.productionCapacity,
    status: item.status === 'ACCEPTED' ? '已入选' : item.status === 'REJECTED' ? '未入选' : '审核中',
    note: item.storageInstructions,
  }
}

function mapCampaign(item: BackendCampaign): MerchantSnapshot['productionOrders'][number] {
  const status = item.status === 'OPEN' ? '待生产' : item.status === 'CLOSED' ? '已完成' : '制作中'
  return {
    id: item.id,
    product: item.title,
    location: item.pickupPoint,
    quantity: item.currentOrders,
    paid: item.unitPriceCents / 100,
    status,
    window: `${item.startsAt.slice(5, 10)} 集中配送`,
    batch: `待创建-${item.id.slice(0, 6)}`,
    note: `${item.foodSpec.weightGrams}g · ${item.foodSpec.oilLevel === 'LOW' ? '低油' : '规格已提交'}`,
  }
}

function mapIncident(item: BackendIncident): SafetyRecord {
  return {
    id: item.id,
    type: '投诉跟进',
    title: item.title,
    date: new Date(item.createdAt).toLocaleString('zh-CN'),
    owner: '平台协作',
    status: item.status === 'RESOLVED' ? '已完成' : '处理中',
    detail: item.description,
  }
}

export async function loadMerchantSnapshot() {
  try {
    const [demands, offers, campaigns, profile, incidents] = await Promise.all([
      request<BackendDemand[]>('/v1/merchant/demands'),
      request<BackendOffer[]>('/v1/merchant/offers'),
      request<BackendCampaign[]>('/v1/merchant/campaigns'),
      request<BackendMerchant>('/v1/merchant/profile'),
      request<BackendIncident[]>('/v1/food-safety/incidents'),
    ])
    const demandMap = new Map(demands.map((item) => [item.id, item.title]))
    return {
      data: {
        demands: demands.map((item) => mapDemand(item, offers, campaigns)),
        quotes: offers.map((item) => mapQuote(item, demandMap.get(item.demandId) ?? '需求报价')),
        productionOrders: campaigns.map(mapCampaign),
        qualifications: [
          { name: '商家经营资质', status: profile.status === 'APPROVED' ? '已通过' : '待复核', expires: '待补充', updated: '平台演示审核状态' },
          { name: '食品经营许可证', status: profile.status === 'APPROVED' ? '已通过' : '待补充', expires: '待补充', updated: '请提交真实证照' },
          { name: '加工地址核验', status: '待复核', expires: '长期有效', updated: '等待现场核验' },
        ],
        safetyRecords: incidents.map(mapIncident),
      } satisfies MerchantSnapshot,
      isFallback: false,
      message: '',
    }
  } catch (error) {
    return {
      data: structuredClone(mockMerchantSnapshot),
      isFallback: true,
      message: error instanceof Error ? error.message : '接口暂不可用，已使用演示数据',
    }
  }
}

export async function submitMerchantQuote(payload: Omit<MerchantQuote, 'id' | 'submittedAt' | 'status'>) {
  try {
    const response = await request<BackendOffer>('/v1/merchant/offers', {
      method: 'POST',
      body: JSON.stringify({
        demandId: payload.demandId,
        unitPriceCents: Math.round(payload.price * 100),
        productionCapacity: payload.minOrder,
        weightGrams: payload.grams,
        ingredients: ['按需求采购并加工'],
        allergens: [],
        oilLevel: 'LOW',
        saltLevel: 'LOW',
        productionTime: '11:30',
        shelfLifeMinutes: 180,
        storageInstructions: payload.note || '建议两小时内食用',
      }),
    })
    return mapQuote(response, payload.demandTitle)
  } catch {
    return { ...payload, id: `quote-local-${Date.now()}`, submittedAt: '刚刚', status: '审核中' as const }
  }
}

export async function updateDemandStatus(demand: MerchantDemand, status: MerchantDemand['status']) {
  return { ...demand, status }
}

export async function createSafetyRecord(payload: Omit<SafetyRecord, 'id' | 'date' | 'status'>) {
  try {
    const session = await ensureSession()
    const response = await request<BackendIncident>('/v1/food-safety/incidents', {
      method: 'POST',
      body: JSON.stringify({
        merchantId: session.merchantId,
        severity: 'LOW',
        title: payload.title,
        description: `${payload.owner}：${payload.detail}`,
        evidenceUrls: [],
      }),
    })
    return mapIncident(response)
  } catch {
    return { ...payload, id: `safe-local-${Date.now()}`, date: '刚刚', status: '待补充' as const }
  }
}
