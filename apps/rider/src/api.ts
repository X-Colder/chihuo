import { mockTasks, type RiderTask, type TaskStatus } from './data'

const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:4000').replace(/\/$/, '')
const tokenKey = 'chihuo.rider.demo-token'

type Envelope<T> = { data?: T; error?: { message?: string } }
type BackendTask = {
  id: string
  orderId: string
  riderId?: string
  status: string
  pickupPoint: string
  deliveryAddress: string
  note?: string
  createdAt: string
  updatedAt: string
}

async function ensureToken() {
  const existing = window.localStorage.getItem(tokenKey)
  if (existing) return existing
  const response = await fetch(`${apiBaseUrl}/v1/auth/demo-login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: '演示骑手', role: 'RIDER' }),
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

function mapTask(task: BackendTask): RiderTask {
  const status: TaskStatus =
    task.status === 'COMPLETED'
      ? '已完成'
      : task.status === 'DELIVERING'
        ? '配送中'
        : task.status === 'PICKED_UP'
          ? '待配送'
          : task.status === 'CANCELLED'
            ? '异常待处理'
            : '待取餐'
  return {
    id: task.id,
    orderNo: task.orderId.slice(0, 8).toUpperCase(),
    storeName: '合作商家',
    pickupAddress: task.pickupPoint,
    dropoffAddress: task.deliveryAddress,
    customer: '演示消费者',
    phone: '138****0000',
    distance: '待计算',
    fee: 5,
    status,
    pickupWindow: '集中配送时段',
    itemSummary: '预售餐品',
    notes: task.note ?? '请核对餐品和批次信息。',
    createdAt: new Date(task.createdAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    updatedAt: new Date(task.updatedAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
  }
}

export async function fetchRiderTasks(): Promise<RiderTask[]> {
  try {
    const [assigned, queue] = await Promise.all([
      request<BackendTask[]>('/v1/rider/tasks', []),
      request<BackendTask[]>('/v1/rider/tasks/queue', []),
    ])
    const tasks = [...assigned, ...queue].filter((task, index, all) => all.findIndex((item) => item.id === task.id) === index)
    return tasks.map(mapTask)
  } catch {
    return mockTasks
  }
}

export async function updateRiderTaskStatus(task: RiderTask, status: TaskStatus): Promise<RiderTask> {
  const statusMap: Partial<Record<TaskStatus, 'PICKED_UP' | 'DELIVERING' | 'COMPLETED'>> = {
    待配送: 'PICKED_UP',
    配送中: 'DELIVERING',
    已完成: 'COMPLETED',
  }
  try {
    await request(`/v1/rider/tasks/${task.id}/claim`, { ok: true }, { method: 'POST' })
    const fallbackBackend: BackendTask = {
      id: task.id,
      orderId: task.orderNo,
      status: statusMap[status] ?? 'CANCELLED',
      pickupPoint: task.pickupAddress,
      deliveryAddress: task.dropoffAddress,
      note: task.notes,
      createdAt: task.createdAt,
      updatedAt: new Date().toISOString(),
    }
    const updated = await request<BackendTask>(
      `/v1/rider/tasks/${task.id}`,
      fallbackBackend,
      { method: 'PATCH', body: JSON.stringify({ status: statusMap[status] }) },
    )
    return mapTask(updated)
  } catch {
    return { ...task, status, updatedAt: '刚刚' }
  }
}

export async function reportRiderIssue(
  task: RiderTask,
  issue: { category: string; description: string },
): Promise<{ ok: boolean }> {
  return request(
    `/v1/rider/tasks/${task.id}`,
    { ok: true },
    { method: 'PATCH', body: JSON.stringify({ status: 'CANCELLED', note: `${issue.category}：${issue.description}` }) },
  )
}
