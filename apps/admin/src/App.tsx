import { useEffect, useMemo, useState } from 'react'
import {
  Activity,
  AlertOctagon,
  AlertTriangle,
  ArrowUpRight,
  BadgeCheck,
  BarChart3,
  Bell,
  Building2,
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ClipboardCheck,
  FileCheck2,
  Filter,
  LayoutDashboard,
  LifeBuoy,
  ListChecks,
  Menu,
  MessageSquareText,
  MoreHorizontal,
  PackageCheck,
  RefreshCw,
  Search,
  ShieldAlert,
  ShieldCheck,
  ShoppingBag,
  Store,
  Utensils,
  UserRound,
  Users,
  X,
} from 'lucide-react'
import {
  type AdminOrder,
  type AdminView,
  type AuditLog,
  type Merchant,
  type MerchantStatus,
  type ReviewItem,
  type ReviewKind,
  type SafetyEvent,
  type SafetyStatus,
} from './data'
import { fetchAdminSnapshot, refundOrder, updateMerchantStatus, updateReviewStatus, updateSafetyStatus } from './api'

const navItems: Array<{ key: AdminView; label: string; icon: React.ReactNode; badge?: string }> = [
  { key: 'dashboard', label: '工作台', icon: <LayoutDashboard size={18} /> },
  { key: 'merchants', label: '商家资质', icon: <Building2 size={18} />, badge: '3' },
  { key: 'review', label: '需求 / 商品审核', icon: <ClipboardCheck size={18} />, badge: '4' },
  { key: 'orders', label: '订单与售后', icon: <ShoppingBag size={18} />, badge: '2' },
  { key: 'safety', label: '食品安全事件', icon: <ShieldAlert size={18} />, badge: '3' },
  { key: 'audit', label: '审计日志', icon: <ListChecks size={18} /> },
]

const viewMeta: Record<AdminView, { eyebrow: string; title: string; description: string }> = {
  dashboard: { eyebrow: '平台概览', title: '早上好，周衡', description: '这是今天的运营与风险概览。' },
  merchants: { eyebrow: '商家管理', title: '商家资质', description: '审核、复核并管理平台入驻商家。' },
  review: { eyebrow: '内容治理', title: '需求 / 商品审核', description: '确保需求规格与商品信息清晰、真实、可履约。' },
  orders: { eyebrow: '交易运营', title: '订单与售后', description: '处理订单异常、退款和消费者反馈。' },
  safety: { eyebrow: '风险中心', title: '食品安全事件', description: '记录、跟进和闭环处理食品安全相关事件。' },
  audit: { eyebrow: '安全与合规', title: '审计日志', description: '查看平台关键操作与系统自动化记录。' },
}

function App() {
  const [activeView, setActiveView] = useState<AdminView>('dashboard')
  const [merchants, setMerchants] = useState<Merchant[]>([])
  const [reviews, setReviews] = useState<ReviewItem[]>([])
  const [orders, setOrders] = useState<AdminOrder[]>([])
  const [safetyEvents, setSafetyEvents] = useState<SafetyEvent[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [menuOpen, setMenuOpen] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const [selectedMerchant, setSelectedMerchant] = useState<Merchant | null>(null)
  const [selectedReview, setSelectedReview] = useState<ReviewItem | null>(null)
  const [selectedOrder, setSelectedOrder] = useState<AdminOrder | null>(null)
  const [selectedSafety, setSelectedSafety] = useState<SafetyEvent | null>(null)

  useEffect(() => {
    fetchAdminSnapshot().then((snapshot) => {
      setMerchants(snapshot.merchants)
      setReviews(snapshot.reviews)
      setOrders(snapshot.orders)
      setSafetyEvents(snapshot.safetyEvents)
      setAuditLogs(snapshot.auditLogs)
      setLoading(false)
    })
  }, [])

  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(null), 2800)
    return () => window.clearTimeout(timer)
  }, [toast])

  const pendingMerchants = merchants.filter((merchant) => ['待审核', '需补充'].includes(merchant.status)).length
  const pendingReviews = reviews.filter((item) => item.status === '待审核').length
  const openSafety = safetyEvents.filter((event) => event.status !== '已结案').length
  const afterSales = orders.filter((order) => order.status === '售后中').length

  function navigate(view: AdminView) {
    setActiveView(view)
    setMenuOpen(false)
    closeDrawers()
  }

  function closeDrawers() {
    setSelectedMerchant(null)
    setSelectedReview(null)
    setSelectedOrder(null)
    setSelectedSafety(null)
  }

  async function handleMerchantStatus(merchant: Merchant, status: MerchantStatus) {
    const updated = await updateMerchantStatus(merchant, status) as Merchant
    setMerchants((current) => current.map((item) => item.id === merchant.id ? updated : item))
    setSelectedMerchant(updated)
    setToast(`已将「${merchant.name}」更新为${status}`)
  }

  async function handleReviewStatus(item: ReviewItem, status: ReviewItem['status']) {
    const updated = await updateReviewStatus(item, status) as ReviewItem
    setReviews((current) => current.map((review) => review.id === item.id ? updated : review))
    setSelectedReview(updated)
    setToast(`审核结果已记录：${status}`)
  }

  async function handleRefund(order: AdminOrder) {
    const updated = await refundOrder(order) as AdminOrder
    setOrders((current) => current.map((item) => item.id === order.id ? updated : item))
    setSelectedOrder(updated)
    setToast(`订单 ${order.orderNo} 已标记为已退款`)
  }

  async function handleSafetyStatus(event: SafetyEvent, status: SafetyStatus) {
    const updated = await updateSafetyStatus(event, status) as SafetyEvent
    setSafetyEvents((current) => current.map((item) => item.id === event.id ? updated : item))
    setSelectedSafety(updated)
    setToast(`事件状态已更新为${status}`)
  }

  return (
    <div className="admin-app">
      <aside className={`admin-sidebar ${menuOpen ? 'is-open' : ''}`}>
        <div className="admin-brand">
          <div className="admin-brand-mark"><Utensils size={20} /></div>
          <div><strong>吃货</strong><span>运营控制台</span></div>
          <button className="icon-button mobile-close" onClick={() => setMenuOpen(false)} aria-label="关闭菜单" title="关闭菜单"><X size={18} /></button>
        </div>
        <div className="workspace-badge"><div className="workspace-logo">C</div><div><strong>深圳 · 南山区域</strong><span>平台运营空间</span></div><ChevronDown size={15} /></div>
        <nav className="admin-nav" aria-label="后台导航">
          {navItems.map((item) => (
            <button key={item.key} className={activeView === item.key ? 'admin-nav-item active' : 'admin-nav-item'} onClick={() => navigate(item.key)}>
              {item.icon}<span>{item.label}</span>{item.badge && <em>{item.badge}</em>}
            </button>
          ))}
        </nav>
        <div className="admin-side-footer">
          <div className="side-help"><LifeBuoy size={16} /><div><strong>需要协助？</strong><span>联系食品安全顾问</span></div><ChevronRight size={15} /></div>
          <div className="operator-card"><div className="operator-avatar">周</div><div><strong>周衡</strong><span>平台管理员</span></div><MoreHorizontal size={17} /></div>
        </div>
      </aside>
      {menuOpen && <button className="admin-scrim" aria-label="关闭菜单" onClick={() => setMenuOpen(false)} />}

      <main className="admin-main">
        <header className="admin-topbar">
          <button className="icon-button admin-menu-trigger" onClick={() => setMenuOpen(true)} aria-label="打开菜单" title="打开菜单"><Menu size={20} /></button>
          <div className="breadcrumbs"><span>控制台</span><ChevronRight size={14} /><strong>{viewMeta[activeView].title}</strong></div>
          <div className="top-actions">
            <div className="global-search"><Search size={16} /><input placeholder="搜索订单、商家或事件" aria-label="搜索订单、商家或事件" /></div>
            <button className="icon-button notification-button" title="消息通知"><Bell size={18} /><i /></button>
            <div className="top-operator"><div className="operator-avatar">周</div><span>周衡</span><ChevronDown size={14} /></div>
          </div>
        </header>

        <div className="admin-content">
          <div className="page-header">
            <div><span className="page-eyebrow">{viewMeta[activeView].eyebrow}</span><h1>{viewMeta[activeView].title}</h1><p>{viewMeta[activeView].description}</p></div>
            <div className="page-actions"><span className="sync-label"><span />数据更新于刚刚</span><button className="outline-button"><RefreshCw size={15} />刷新</button></div>
          </div>

          {loading ? <LoadingState /> : (
            <>
              {activeView === 'dashboard' && <DashboardView merchants={merchants} reviews={reviews} orders={orders} safetyEvents={safetyEvents} navigate={navigate} selectMerchant={setSelectedMerchant} selectSafety={setSelectedSafety} />}
              {activeView === 'merchants' && <MerchantView merchants={merchants} selectMerchant={setSelectedMerchant} />}
              {activeView === 'review' && <ReviewView reviews={reviews} selectReview={setSelectedReview} />}
              {activeView === 'orders' && <OrderView orders={orders} selectOrder={setSelectedOrder} />}
              {activeView === 'safety' && <SafetyView events={safetyEvents} selectSafety={setSelectedSafety} />}
              {activeView === 'audit' && <AuditView logs={auditLogs} />}
            </>
          )}
        </div>
      </main>

      {selectedMerchant && <MerchantDrawer merchant={selectedMerchant} onClose={closeDrawers} onStatus={handleMerchantStatus} />}
      {selectedReview && <ReviewDrawer item={selectedReview} onClose={closeDrawers} onStatus={handleReviewStatus} />}
      {selectedOrder && <OrderDrawer order={selectedOrder} onClose={closeDrawers} onRefund={handleRefund} />}
      {selectedSafety && <SafetyDrawer event={selectedSafety} onClose={closeDrawers} onStatus={handleSafetyStatus} />}
      {toast && <div className="admin-toast"><CheckCircle2 size={17} />{toast}</div>}
    </div>
  )
}

function LoadingState() {
  return <div className="loading-state"><div className="spinner" /><span>正在同步运营数据</span></div>
}

function DashboardView({
  merchants,
  reviews,
  orders,
  safetyEvents,
  navigate,
  selectMerchant,
  selectSafety,
}: {
  merchants: Merchant[]
  reviews: ReviewItem[]
  orders: AdminOrder[]
  safetyEvents: SafetyEvent[]
  navigate: (view: AdminView) => void
  selectMerchant: (merchant: Merchant) => void
  selectSafety: (event: SafetyEvent) => void
}) {
  const pendingMerchants = merchants.filter((merchant) => ['待审核', '需补充'].includes(merchant.status))
  const pendingReviews = reviews.filter((review) => review.status === '待审核')
  const activeSafety = safetyEvents.filter((event) => event.status !== '已结案')
  return (
    <div className="dashboard-grid">
      <section className="dashboard-main">
        <div className="stat-grid">
          <StatCard label="今日交易额" value="¥18,642" change="+12.8%" note="较昨日同期" icon={<BarChart3 size={18} />} tone="green" />
          <StatCard label="有效订单" value="486" change="+8.4%" note="今日累计" icon={<ShoppingBag size={18} />} tone="blue" />
          <StatCard label="活跃商家" value="42" change="+3" note="本周新增" icon={<Store size={18} />} tone="yellow" />
          <StatCard label="待处理风险" value={String(activeSafety.length + pendingMerchants.length)} change="需关注" note="资质与食品安全" icon={<ShieldAlert size={18} />} tone="red" />
        </div>
        <section className="panel chart-panel">
          <div className="panel-header"><div><span className="panel-eyebrow">业务趋势</span><h2>订单与交易概览</h2></div><button className="select-button">最近 7 天 <ChevronDown size={14} /></button></div>
          <div className="chart-legend"><span><i className="legend-green" />成交额</span><span><i className="legend-blue" />订单量</span></div>
          <div className="fake-chart">
            <div className="chart-y"><span>¥30k</span><span>¥20k</span><span>¥10k</span><span>¥0</span></div>
            <div className="chart-body"><div className="grid-lines"><i /><i /><i /><i /></div><svg viewBox="0 0 700 220" preserveAspectRatio="none" aria-label="近七日交易趋势图"><path className="chart-area" d="M0,178 C45,164 72,171 112,143 S176,155 218,124 S273,116 315,128 S365,74 408,91 S464,117 503,72 S560,94 601,57 S654,52 700,18 L700,220 L0,220 Z" /><path className="chart-line green-line" d="M0,178 C45,164 72,171 112,143 S176,155 218,124 S273,116 315,128 S365,74 408,91 S464,117 503,72 S560,94 601,57 S654,52 700,18" /><path className="order-line" d="M0,192 C60,185 85,189 122,176 S187,184 222,164 S283,168 321,173 S373,132 411,143 S472,155 507,123 S563,132 603,116 S656,106 700,91" /></svg><div className="chart-x"><span>8/27</span><span>8/28</span><span>8/29</span><span>8/30</span><span>8/31</span><span>9/1</span><span>9/2</span></div></div>
          </div>
        </section>
        <section className="panel activity-panel">
          <div className="panel-header"><div><span className="panel-eyebrow">需要你处理</span><h2>运营待办</h2></div><button className="text-button" onClick={() => navigate('audit')}>查看日志 <ArrowUpRight size={14} /></button></div>
          <div className="todo-list">
            <TodoItem icon={<Building2 size={17} />} tone="yellow" title={`${pendingMerchants.length} 家商家需要资质复核`} meta="资质审核" onClick={() => navigate('merchants')} />
            <TodoItem icon={<ClipboardCheck size={17} />} tone="blue" title={`${pendingReviews.length} 条需求 / 商品等待审核`} meta="内容审核" onClick={() => navigate('review')} />
            <TodoItem icon={<ShieldAlert size={17} />} tone="red" title={`${activeSafety.length} 起食品安全事件未结案`} meta="风险中心" onClick={() => navigate('safety')} />
            <TodoItem icon={<MessageSquareText size={17} />} tone="green" title="2 条消费者售后反馈待回复" meta="订单售后" onClick={() => navigate('orders')} />
          </div>
        </section>
      </section>
      <aside className="dashboard-side">
        <section className="panel quick-panel">
          <div className="panel-header"><div><span className="panel-eyebrow">实时监控</span><h2>风险动态</h2></div><span className="live-badge"><i />LIVE</span></div>
          <div className="risk-list">{activeSafety.map((event) => <RiskItem key={event.id} event={event} onClick={() => selectSafety(event)} />)}</div>
          <button className="panel-link" onClick={() => navigate('safety')}>进入风险中心 <ArrowUpRight size={14} /></button>
        </section>
        <section className="panel merchant-mini-panel">
          <div className="panel-header"><div><span className="panel-eyebrow">商家质量</span><h2>近期入驻</h2></div><button className="icon-button" onClick={() => navigate('merchants')} title="查看全部商家"><ArrowUpRight size={16} /></button></div>
          <div className="mini-merchant-list">{merchants.slice(0, 3).map((merchant) => <button className="mini-merchant" key={merchant.id} onClick={() => selectMerchant(merchant)}><div className="store-avatar"><Store size={16} /></div><div><strong>{merchant.name}</strong><span>{merchant.area}</span></div><StatusTag label={merchant.status} /></button>)}</div>
        </section>
        <section className="panel fulfillment-panel">
          <div className="panel-header"><div><span className="panel-eyebrow">履约表现</span><h2>今日配送</h2></div><PackageCheck size={18} className="panel-icon" /></div>
          <div className="fulfillment-value"><strong>96.4%</strong><span>准时完成率</span></div>
          <div className="progress-track"><span style={{ width: '96.4%' }} /></div>
          <div className="fulfillment-meta"><span>完成订单 <b>{orders.filter((order) => order.status === '已完成').length + 480}</b></span><span>异常订单 <b className="danger-text">{orders.filter((order) => order.status === '售后中').length}</b></span></div>
        </section>
      </aside>
    </div>
  )
}

function StatCard({ label, value, change, note, icon, tone }: { label: string; value: string; change: string; note: string; icon: React.ReactNode; tone: string }) {
  return <div className={`stat-card ${tone}`}><div className="stat-icon">{icon}</div><span>{label}</span><strong>{value}</strong><small><b>{change}</b> {note}</small></div>
}

function TodoItem({ icon, tone, title, meta, onClick }: { icon: React.ReactNode; tone: string; title: string; meta: string; onClick: () => void }) {
  return <button className="todo-item" onClick={onClick}><div className={`todo-icon ${tone}`}>{icon}</div><div><strong>{title}</strong><span>{meta}</span></div><ChevronRight size={16} /></button>
}

function RiskItem({ event, onClick }: { event: SafetyEvent; onClick: () => void }) {
  return <button className="risk-item" onClick={onClick}><div className={`risk-marker ${event.severity === '高' ? 'high' : event.severity === '中' ? 'medium' : 'low'}`}><AlertTriangle size={14} /></div><div><strong>{event.title}</strong><span>{event.merchant} · {event.reportedAt}</span></div><ChevronRight size={15} /></button>
}

function MerchantView({ merchants, selectMerchant }: { merchants: Merchant[]; selectMerchant: (merchant: Merchant) => void }) {
  const [status, setStatus] = useState<'全部' | MerchantStatus>('全部')
  const [query, setQuery] = useState('')
  const filtered = merchants.filter((merchant) => (status === '全部' || merchant.status === status) && `${merchant.name}${merchant.owner}${merchant.area}`.includes(query))
  return <ListPage toolbar={<><div className="search-field"><Search size={15} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索商家名称或区域" /></div><button className="outline-button"><Filter size={15} />筛选</button></>} tabs={['全部', '待审核', '需补充', '已通过', '已暂停']} activeTab={status} onTab={(tab) => setStatus(tab as '全部' | MerchantStatus)}><section className="panel table-panel"><TableHeader title="商家列表" count={`${filtered.length} 家`} /><div className="data-table merchant-table"><div className="table-row table-head"><span>商家</span><span>经营区域</span><span>证照有效期</span><span>风险等级</span><span>当前状态</span><span /></div>{filtered.map((merchant) => <button className="table-row table-body" key={merchant.id} onClick={() => selectMerchant(merchant)}><span className="merchant-cell"><div className="store-avatar"><Store size={16} /></div><span><strong>{merchant.name}</strong><small>{merchant.owner}</small></span></span><span>{merchant.area}</span><span>{merchant.licenseExpire}</span><span><RiskTag risk={merchant.risk} /></span><span><StatusTag label={merchant.status} /></span><ChevronRight size={16} /></button>)}</div></section></ListPage>
}

function ReviewView({ reviews, selectReview }: { reviews: ReviewItem[]; selectReview: (review: ReviewItem) => void }) {
  const [kind, setKind] = useState<'全部' | ReviewKind>('全部')
  const [status, setStatus] = useState<'全部' | ReviewItem['status']>('全部')
  const filtered = reviews.filter((item) => (kind === '全部' || item.kind === kind) && (status === '全部' || item.status === status))
  return <ListPage toolbar={<><button className="outline-button"><Filter size={15} />高级筛选</button><button className="primary-small"><PlusIcon />新建审核批次</button></>} tabs={['全部', '需求', '商品']} activeTab={kind} onTab={(tab) => setKind(tab as '全部' | ReviewKind)}><section className="panel table-panel"><div className="review-filter-row"><TableHeader title="审核队列" count={`${filtered.length} 条`} /><div className="mini-select"><span>状态</span><select value={status} onChange={(event) => setStatus(event.target.value as '全部' | ReviewItem['status'])}><option>全部</option><option>待审核</option><option>需补充</option><option>已通过</option><option>已驳回</option></select><ChevronDown size={14} /></div></div><div className="data-table review-table"><div className="table-row table-head"><span>内容</span><span>商家 / 发起方</span><span>关键指标</span><span>提交时间</span><span>状态</span><span /></div>{filtered.map((item) => <button className="table-row table-body" key={item.id} onClick={() => selectReview(item)}><span className="review-cell"><div className={`review-type ${item.kind === '需求' ? 'demand' : 'product'}`}>{item.kind === '需求' ? <Users size={16} /> : <PackageCheck size={16} />}</div><span><strong>{item.title}</strong><small>{item.summary}</small></span></span><span>{item.merchant}</span><span className="signals-cell">{item.signals.slice(0, 2).map((signal) => <em key={signal}>{signal}</em>)}</span><span>{item.submittedAt}</span><span><StatusTag label={item.status} /></span><ChevronRight size={16} /></button>)}</div></section></ListPage>
}

function OrderView({ orders, selectOrder }: { orders: AdminOrder[]; selectOrder: (order: AdminOrder) => void }) {
  const [status, setStatus] = useState<'全部' | AdminOrder['status']>('全部')
  const filtered = orders.filter((order) => status === '全部' || order.status === status)
  return <ListPage toolbar={<><div className="search-field"><Search size={15} /><input placeholder="搜索订单号或顾客" /></div><button className="outline-button"><Filter size={15} />筛选</button></>} tabs={['全部', '配送中', '售后中', '已完成', '已退款']} activeTab={status} onTab={(tab) => setStatus(tab as '全部' | AdminOrder['status'])}><section className="panel table-panel"><TableHeader title="订单列表" count={`${filtered.length} 笔`} /><div className="data-table order-table"><div className="table-row table-head"><span>订单信息</span><span>商家</span><span>顾客</span><span>金额</span><span>状态</span><span>下单时间</span><span /></div>{filtered.map((order) => <button className="table-row table-body" key={order.id} onClick={() => selectOrder(order)}><span className="order-cell"><strong>{order.orderNo}</strong><small>{order.delivery}</small></span><span>{order.merchant}</span><span>{order.customer}</span><span className="amount-cell">¥{order.amount.toFixed(2)}</span><span><StatusTag label={order.status} /></span><span>{order.createdAt}</span><ChevronRight size={16} /></button>)}</div></section></ListPage>
}

function SafetyView({ events, selectSafety }: { events: SafetyEvent[]; selectSafety: (event: SafetyEvent) => void }) {
  const [severity, setSeverity] = useState<'全部' | SafetyEvent['severity']>('全部')
  const filtered = events.filter((event) => severity === '全部' || event.severity === severity)
  return <ListPage toolbar={<><button className="outline-button"><Filter size={15} />筛选事件</button><button className="primary-small"><ShieldCheck size={15} />导出事件报告</button></>} tabs={['全部', '高', '中', '低']} activeTab={severity} onTab={(tab) => setSeverity(tab as '全部' | SafetyEvent['severity'])}><div className="safety-summary"><SafetySummary icon={<AlertOctagon size={19} />} value={String(events.filter((event) => event.status === '待处理').length)} label="待处理" tone="red" /><SafetySummary icon={<Activity size={19} />} value={String(events.filter((event) => event.status === '调查中').length)} label="调查中" tone="yellow" /><SafetySummary icon={<BadgeCheck size={19} />} value={String(events.filter((event) => event.status === '已结案').length + 18)} label="本月已结案" tone="green" /></div><section className="panel table-panel"><TableHeader title="事件列表" count={`${filtered.length} 起`} /><div className="data-table safety-table"><div className="table-row table-head"><span>事件</span><span>关联商家</span><span>级别</span><span>状态</span><span>上报人</span><span>时间</span><span /></div>{filtered.map((event) => <button className="table-row table-body" key={event.id} onClick={() => selectSafety(event)}><span className="event-cell"><div className={`event-marker ${event.severity.toLowerCase()}`}><AlertTriangle size={15} /></div><span><strong>{event.title}</strong><small>{event.orderNo}</small></span></span><span>{event.merchant}</span><span><SeverityTag severity={event.severity} /></span><span><StatusTag label={event.status} /></span><span>{event.reporter}</span><span>{event.reportedAt}</span><ChevronRight size={16} /></button>)}</div></section></ListPage>
}

function AuditView({ logs }: { logs: AuditLog[] }) {
  return <ListPage toolbar={<><div className="search-field"><Search size={15} /><input placeholder="搜索操作人、对象或动作" /></div><button className="outline-button"><Filter size={15} />筛选</button></>} tabs={['全部', '人工操作', '系统操作']} activeTab="全部" onTab={() => undefined}><section className="panel table-panel"><TableHeader title="操作记录" count={`${logs.length} 条`} /><div className="data-table audit-table"><div className="table-row table-head"><span>操作人</span><span>动作</span><span>对象</span><span>结果</span><span>IP 地址</span><span>时间</span></div>{logs.map((log) => <div className="table-row table-body" key={log.id}><span className="actor-cell"><div className="actor-dot">{log.actor.startsWith('系统') || log.actor.startsWith('AI') ? <Activity size={14} /> : <UserRound size={14} />}</div>{log.actor}</span><span>{log.action}</span><span>{log.target}</span><span><ResultTag result={log.result} /></span><span className="mono-text">{log.ip}</span><span>{log.createdAt}</span></div>)}</div></section></ListPage>
}

function ListPage({ toolbar, tabs, activeTab, onTab, children }: { toolbar: React.ReactNode; tabs: string[]; activeTab: string; onTab: (tab: string) => void; children: React.ReactNode }) {
  return <div className="list-page"><div className="list-toolbar"><div className="segmented-tabs">{tabs.map((tab) => <button key={tab} className={activeTab === tab ? 'active' : ''} onClick={() => onTab(tab)}>{tab}</button>)}</div><div className="toolbar-actions">{toolbar}</div></div>{children}</div>
}

function TableHeader({ title, count }: { title: string; count: string }) {
  return <div className="table-header"><h2>{title}</h2><span>{count}</span></div>
}

function SafetySummary({ icon, value, label, tone }: { icon: React.ReactNode; value: string; label: string; tone: string }) {
  return <div className={`safety-summary-card ${tone}`}><div>{icon}</div><span>{label}</span><strong>{value}</strong></div>
}

function StatusTag({ label }: { label: string }) {
  const tone = label.includes('待') || label === '售后中' || label === '调查中' ? 'pending' : label.includes('补充') || label === '高' ? 'warning' : label.includes('驳回') || label === '已暂停' ? 'danger' : 'success'
  return <em className={`status-tag ${tone}`}><i />{label}</em>
}

function RiskTag({ risk }: { risk: Merchant['risk'] }) {
  return <em className={`risk-tag ${risk === '高' ? 'danger' : risk === '中' ? 'warning' : 'safe'}`}><i />{risk}风险</em>
}

function SeverityTag({ severity }: { severity: SafetyEvent['severity'] }) {
  return <em className={`severity-tag ${severity === '高' ? 'high' : severity === '中' ? 'medium' : 'low'}`}><i />{severity}风险</em>
}

function ResultTag({ result }: { result: AuditLog['result'] }) {
  return <em className={`result-tag ${result === '成功' ? 'success' : result === '失败' ? 'danger' : 'pending'}`}>{result}</em>
}

function PlusIcon() {
  return <span className="plus-symbol">+</span>
}

function Drawer({ title, eyebrow, children, onClose }: { title: string; eyebrow: string; children: React.ReactNode; onClose: () => void }) {
  return <div className="drawer-backdrop" onClick={onClose}><aside className="drawer" onClick={(event) => event.stopPropagation()}><div className="drawer-head"><div><span>{eyebrow}</span><h2>{title}</h2></div><button className="icon-button" onClick={onClose} aria-label="关闭详情" title="关闭详情"><X size={18} /></button></div>{children}</aside></div>
}

function MerchantDrawer({ merchant, onClose, onStatus }: { merchant: Merchant; onClose: () => void; onStatus: (merchant: Merchant, status: MerchantStatus) => void }) {
  return <Drawer title={merchant.name} eyebrow="商家资质详情" onClose={onClose}><div className="drawer-status-line"><StatusTag label={merchant.status} /><span>提交于 {merchant.appliedAt}</span></div><div className="drawer-section"><h3>主体信息</h3><DetailLine label="经营主体" value={merchant.owner} /><DetailLine label="经营区域" value={merchant.area} /><DetailLine label="经营地址" value={merchant.address} /><DetailLine label="联系电话" value={merchant.phone} /></div><div className="drawer-section"><h3>资质与证照</h3><DetailLine label="食品经营许可证" value={merchant.licenseNo} /><DetailLine label="有效期至" value={merchant.licenseExpire} /><div className="document-list">{merchant.documents.map((document) => <div key={document} className="document-row"><FileCheck2 size={16} /><span>{document}</span><CheckCircle2 size={15} className="document-check" /></div>)}</div></div><div className="drawer-section"><h3>系统判断</h3><div className="risk-callout"><ShieldCheck size={17} /><div><strong>{merchant.risk}风险等级</strong><span>证照字段已完成 OCR，经营地址等待人工确认。</span></div></div></div><div className="drawer-actions">{merchant.status !== '已通过' && <button className="primary-small" onClick={() => onStatus(merchant, '已通过')}><Check size={15} />通过审核</button>}{merchant.status !== '需补充' && <button className="outline-button" onClick={() => onStatus(merchant, '需补充')}><MessageSquareText size={15} />要求补充</button>}{merchant.status !== '已暂停' && <button className="danger-button" onClick={() => onStatus(merchant, '已暂停')}><ShieldAlert size={15} />暂停交易</button>}</div></Drawer>
}

function ReviewDrawer({ item, onClose, onStatus }: { item: ReviewItem; onClose: () => void; onStatus: (item: ReviewItem, status: ReviewItem['status']) => void }) {
  return <Drawer title={item.title} eyebrow={`${item.kind}审核详情`} onClose={onClose}><div className="drawer-status-line"><StatusTag label={item.status} /><span>提交于 {item.submittedAt}</span></div><div className="review-preview"><div className={`review-type large ${item.kind === '需求' ? 'demand' : 'product'}`}>{item.kind === '需求' ? <Users size={25} /> : <PackageCheck size={25} />}</div><div><strong>{item.kind === '需求' ? '聚合需求规格' : '商家商品规格'}</strong><p>{item.summary}</p></div></div><div className="drawer-section"><h3>审核信号</h3><div className="signal-grid">{item.signals.map((signal) => <div key={signal}><CheckCircle2 size={15} /><span>{signal}</span></div>)}</div></div>{item.price && <DetailLine label="拟定售价" value={item.price} />}{item.quantity && <DetailLine label="预计需求量" value={item.quantity} />}<div className="drawer-section reviewer-note"><h3>审核备注</h3><textarea rows={4} placeholder="记录审核依据和需要商家补充的信息" /></div><div className="drawer-actions">{item.status !== '已通过' && <button className="primary-small" onClick={() => onStatus(item, '已通过')}><Check size={15} />通过审核</button>}{item.status !== '已驳回' && <button className="danger-button" onClick={() => onStatus(item, '已驳回')}><X size={15} />驳回</button>}{item.status !== '需补充' && <button className="outline-button" onClick={() => onStatus(item, '需补充')}><MessageSquareText size={15} />要求补充</button>}</div></Drawer>
}

function OrderDrawer({ order, onClose, onRefund }: { order: AdminOrder; onClose: () => void; onRefund: (order: AdminOrder) => void }) {
  return <Drawer title={order.orderNo} eyebrow="订单与售后详情" onClose={onClose}><div className="drawer-status-line"><StatusTag label={order.status} /><span>下单于今天 {order.createdAt}</span></div><div className="drawer-section"><h3>订单信息</h3><DetailLine label="商家" value={order.merchant} /><DetailLine label="顾客" value={order.customer} /><DetailLine label="配送信息" value={order.delivery} /><DetailLine label="订单金额" value={`¥${order.amount.toFixed(2)}`} strong /></div>{order.issue ? <div className="issue-callout"><AlertTriangle size={17} /><div><strong>售后原因</strong><span>{order.issue}</span></div></div> : <div className="success-callout"><CheckCircle2 size={17} /><span>当前订单没有售后异常。</span></div>}<div className="drawer-section"><h3>订单时间线</h3><div className="order-timeline"><TimelineEntry label="订单创建" time={order.createdAt} done /><TimelineEntry label="商家接单" time="09:54" done /><TimelineEntry label="骑手配送" time={order.status === '配送中' ? '进行中' : '10:12'} done={order.status !== '已退款'} /><TimelineEntry label="订单完成" time={order.status === '已完成' ? '10:38' : '等待处理'} done={order.status === '已完成'} /></div></div><div className="drawer-actions">{order.status === '售后中' && <button className="primary-small" onClick={() => onRefund(order)}><Check size={15} />同意退款</button>}{order.status !== '已退款' && order.status !== '已完成' && <button className="outline-button"><MessageSquareText size={15} />联系消费者</button>}</div></Drawer>
}

function SafetyDrawer({ event, onClose, onStatus }: { event: SafetyEvent; onClose: () => void; onStatus: (event: SafetyEvent, status: SafetyStatus) => void }) {
  return <Drawer title={event.title} eyebrow="食品安全事件详情" onClose={onClose}><div className="drawer-status-line"><SeverityTag severity={event.severity} /><StatusTag label={event.status} /></div><div className="issue-callout safety"><AlertOctagon size={18} /><div><strong>{event.merchant}</strong><span>{event.summary}</span></div></div><div className="drawer-section"><h3>事件信息</h3><DetailLine label="关联订单" value={event.orderNo} /><DetailLine label="上报人" value={event.reporter} /><DetailLine label="上报时间" value={event.reportedAt} /></div><div className="drawer-section"><h3>已执行措施</h3><div className="action-checklist">{event.actions.map((action) => <div key={action}><CheckCircle2 size={15} /><span>{action}</span></div>)}</div></div><div className="drawer-section"><h3>处理记录</h3><textarea rows={4} placeholder="填写调查过程、责任判断与后续动作" /></div><div className="drawer-actions">{event.status === '待处理' && <button className="primary-small" onClick={() => onStatus(event, '调查中')}><Activity size={15} />开始调查</button>}{event.status === '调查中' && <button className="primary-small" onClick={() => onStatus(event, '已结案')}><Check size={15} />结案</button>}{event.status !== '已结案' && <button className="danger-button"><ShieldAlert size={15} />暂停相关商品</button>}</div></Drawer>
}

function DetailLine({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return <div className="detail-line"><span>{label}</span><strong className={strong ? 'teal-text' : ''}>{value}</strong></div>
}

function TimelineEntry({ label, time, done }: { label: string; time: string; done: boolean }) {
  return <div className={`timeline-entry ${done ? 'done' : ''}`}><i>{done && <Check size={10} />}</i><span>{label}</span><time>{time}</time></div>
}

export default App
