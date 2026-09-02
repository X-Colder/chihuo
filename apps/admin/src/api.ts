import {
  mockAuditLogs,
  mockMerchants,
  mockOrders,
  mockReviews,
  mockSafetyEvents,
  type AdminOrder,
  type AuditLog,
  type Merchant,
  type ReviewItem,
  type SafetyEvent,
} from './data'

const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:4000').replace(/\/$/, '')
const tokenKey = 'chihuo.admin.demo-token'

type Envelope<T> = { data?: T; error?: { message?: string } }
type BackendMerchant = { id: string; ownerUserId: string; name: string; status: string; license: Record<string, unknown>; createdAt: string }
type BackendDemand = {
  id: string
  title: string
  category: string
  serviceArea: string
  budgetMinCents: number
  budgetMaxCents: number
  memberCount: number
  minimumMembers: number
  createdAt: string
  status: string
}
type BackendCampaign = {
  id: string
  title: string
  merchantId: string
  unitPriceCents: number
  currentOrders: number
  maximumOrders: number
  status: string
  createdAt: string
  foodSpec: { weightGrams: number; ingredients: string[]; allergens: string[] }
}
type BackendOrder = { id: string; campaignId: string; consumerId: string; totalCents: number; status: string; createdAt: string; deliveryAddress: string }
type BackendIncident = { id: string; title: string; merchantId: string; orderId?: string; severity: string; status: string; description: string; reportedBy: string; createdAt: string }

async function ensureToken() {
  const existing = window.localStorage.getItem(tokenKey)
  if (existing) return existing
  const response = await fetch(`${apiBaseUrl}/v1/auth/demo-login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: '演示平台管理员', role: 'ADMIN' }),
    signal: AbortSignal.timeout(1600),
  })
  if (!response.ok) throw new Error(`登录失败（${response.status}）`)
  const payload = (await response.json()) as Envelope<{ token: string }>
  if (!payload.data?.token) throw new Error('登录响应缺少 token')
  window.localStorage.setItem(tokenKey, payload.data.token)
  return payload.data.token
}

async function request<T>(path: string, fallback: T, init?: RequestInit): Promise<T> {
  try {
    const token = await ensureToken()
    const response = await fetch(`${apiBaseUrl}${path}`, {
      ...init,
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...(init?.headers ?? {}) },
      signal: AbortSignal.timeout(3000),
    })
    if (!response.ok) throw new Error(`API ${response.status}`)
    const payload = (await response.json()) as Envelope<T>
    return (payload.data ?? payload) as T
  } catch {
    return fallback
  }
}

function mapMerchant(item: BackendMerchant): Merchant {
  const statusMap: Record<string, Merchant['status']> = { PENDING: '待审核', APPROVED: '已通过', REJECTED: '需补充', SUSPENDED: '已暂停' }
  return {
    id: item.id,
    name: item.name,
    owner: item.ownerUserId,
    phone: '待补充',
    area: '待核验',
    category: '餐饮服务经营者',
    status: statusMap[item.status] ?? '待审核',
    licenseNo: String(item.license.licenseNo ?? '待上传'),
    licenseExpire: String(item.license.expireAt ?? '待核验'),
    appliedAt: new Date(item.createdAt).toLocaleString('zh-CN'),
    products: 0,
    risk: item.status === 'SUSPENDED' ? '高' : '中',
    address: String(item.license.address ?? '待现场核验'),
    documents: ['营业执照', '食品经营许可证'],
  }
}

function mapReview(item: BackendDemand | BackendCampaign, kind: ReviewItem['kind'], merchantName: string): ReviewItem {
  const isDemand = kind === '需求'
  if (isDemand) {
    const demand = item as BackendDemand
    return {
      id: demand.id,
      kind,
      title: demand.title,
      merchant: merchantName,
      summary: `${demand.memberCount} 人已加入，预算 ¥${(demand.budgetMinCents / 100).toFixed(0)}–${(demand.budgetMaxCents / 100).toFixed(0)}`,
      submittedAt: new Date(demand.createdAt).toLocaleString('zh-CN'),
      status: demand.status === 'OPEN' || demand.status === 'READY' ? '已通过' : demand.status === 'REJECTED' ? '已驳回' : '待审核',
      signals: [`${demand.memberCount} 人已确认`, '区域待核验'],
      quantity: `${demand.memberCount} 份`,
    }
  }
  const campaign = item as BackendCampaign
  return {
    id: campaign.id,
    kind,
    title: campaign.title,
    merchant: merchantName,
    summary: `餐品规格已提交，当前预售 ${campaign.currentOrders}/${campaign.maximumOrders} 份`,
    submittedAt: new Date(campaign.createdAt).toLocaleString('zh-CN'),
    status: campaign.status === 'OPEN' ? '已通过' : campaign.status === 'REJECTED' ? '已驳回' : '待审核',
    signals: ['配料已填写', '批次规则待复核'],
    price: `¥${(campaign.unitPriceCents / 100).toFixed(0)} / 份`,
  }
}

function mapOrder(item: BackendOrder): AdminOrder {
  const statusMap: Record<string, AdminOrder['status']> = {
    DELIVERED: '已完成',
    REFUNDED: '已退款',
    CANCELLED: '售后中',
    DELIVERING: '配送中',
    PICKED_UP: '配送中',
    PAID: '配送中',
    PREPARING: '配送中',
    ACCEPTED: '配送中',
    READY_FOR_PICKUP: '配送中',
  }
  return {
    id: item.id,
    orderNo: item.id.slice(0, 12).toUpperCase(),
    merchant: `商家 ${item.campaignId.slice(0, 6)}`,
    customer: `用户 ${item.consumerId.slice(0, 6)}`,
    amount: item.totalCents / 100,
    status: statusMap[item.status] ?? '售后中',
    issue: item.status === 'CANCELLED' ? '订单已取消，等待售后处理' : undefined,
    createdAt: new Date(item.createdAt).toLocaleString('zh-CN'),
    delivery: item.deliveryAddress,
  }
}

function mapSafety(item: BackendIncident): SafetyEvent {
  const severity: SafetyEvent['severity'] = item.severity === 'HIGH' || item.severity === 'CRITICAL' ? '高' : item.severity === 'MEDIUM' ? '中' : '低'
  const status: SafetyEvent['status'] = item.status === 'RESOLVED' ? '已结案' : item.status === 'INVESTIGATING' ? '调查中' : '待处理'
  return {
    id: item.id,
    title: item.title,
    merchant: `商家 ${item.merchantId.slice(0, 6)}`,
    orderNo: item.orderId?.slice(0, 12).toUpperCase() ?? '店铺级事件',
    severity,
    status,
    reportedAt: new Date(item.createdAt).toLocaleString('zh-CN'),
    reporter: item.reportedBy.slice(0, 8),
    summary: item.description,
    actions: ['已记录事件', '等待平台审核和后续处置'],
  }
}

export async function fetchAdminSnapshot() {
  const [merchantRows, demands, campaigns, orders, incidents] = await Promise.all([
    request<BackendMerchant[]>('/v1/admin/merchants', []),
    request<BackendDemand[]>('/v1/admin/demands', []),
    request<BackendCampaign[]>('/v1/admin/campaigns', []),
    request<BackendOrder[]>('/v1/orders', []),
    request<BackendIncident[]>('/v1/food-safety/incidents', []),
  ])
  const merchants = merchantRows.length ? merchantRows.map(mapMerchant) : mockMerchants
  const merchantNames = new Map(merchantRows.map((item) => [item.id, item.name]))
  const reviews = demands.length || campaigns.length
    ? [
        ...demands.map((item) => mapReview(item, '需求', '消费者共创')),
        ...campaigns.map((item) => mapReview(item, '商品', merchantNames.get(item.merchantId) ?? '合作商家')),
      ]
    : mockReviews
  return {
    merchants,
    reviews,
    orders: orders.length ? orders.map(mapOrder) : mockOrders,
    safetyEvents: incidents.length ? incidents.map(mapSafety) : mockSafetyEvents,
    auditLogs: mockAuditLogs,
  }
}

export async function updateMerchantStatus(merchant: Merchant, status: Merchant['status']) {
  const statusMap: Record<Merchant['status'], 'APPROVED' | 'REJECTED' | 'SUSPENDED'> = {
    '待审核': 'REJECTED',
    '已通过': 'APPROVED',
    '需补充': 'REJECTED',
    '已暂停': 'SUSPENDED',
  }
  const result = await request<BackendMerchant | Merchant>(`/v1/admin/merchants/${merchant.id}/review`, merchant, {
    method: 'PATCH',
    body: JSON.stringify({ status: statusMap[status] }),
  })
  return 'ownerUserId' in result ? mapMerchant(result) : { ...result, status }
}

export async function updateReviewStatus(item: ReviewItem, status: ReviewItem['status']) {
  const nextStatus = status === '已通过' ? 'OPEN' : 'REJECTED'
  const path = item.kind === '需求' ? `/v1/admin/demands/${item.id}/review` : `/v1/admin/campaigns/${item.id}/review`
  await request(path, item, { method: 'PATCH', body: JSON.stringify({ status: nextStatus }) })
  return { ...item, status }
}

export async function refundOrder(order: AdminOrder) {
  await request(`/v1/admin/orders/${order.id}/refund`, order, { method: 'POST' })
  return { ...order, status: '已退款' as const }
}

export async function updateSafetyStatus(event: SafetyEvent, status: SafetyEvent['status']) {
  return { ...event, status }
}
