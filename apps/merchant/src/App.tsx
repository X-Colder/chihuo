import {
  AlertCircle,
  ArrowUpRight,
  BadgeCheck,
  Bell,
  CalendarClock,
  Check,
  ChevronRight,
  ClipboardCheck,
  ClipboardList,
  FileCheck2,
  FilePlus2,
  FileText,
  Info,
  Leaf,
  Menu,
  PackageCheck,
  Plus,
  ShieldCheck,
  ShoppingBasket,
  Store,
  Truck,
  Upload,
  Utensils,
  X,
} from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { createSafetyRecord, loadMerchantSnapshot, submitMerchantQuote, updateDemandStatus } from './lib/api'
import type { MerchantDemand, MerchantQuote, MerchantSnapshot, MerchantView, ProductionOrder, SafetyRecord } from './types'

const navItems: Array<{ id: MerchantView; label: string; icon: typeof Store; count?: number }> = [
  { id: 'dashboard', label: '工作台概览', icon: Store },
  { id: 'demand-hall', label: '需求大厅', icon: ClipboardList, count: 3 },
  { id: 'quotes', label: '我的报价', icon: FileText },
  { id: 'production', label: '订单生产', icon: ShoppingBasket, count: 2 },
  { id: 'qualification', label: '资质中心', icon: BadgeCheck },
  { id: 'safety', label: '食品安全', icon: ShieldCheck },
]

function statusTone(status: string) {
  if (['已完成', '已通过', '已入选', '高匹配', '已转预售'].includes(status)) return 'success'
  if (['处理中', '审核中', '可尝试'].includes(status)) return 'accent'
  if (['待生产', '待交接', '待补充', '需评估'].includes(status)) return 'neutral'
  return 'info'
}

function StatusPill({ children }: { children: string }) {
  return <span className={`status-pill ${statusTone(children)}`}>{children}</span>
}

function MerchantSidebar({
  activeView,
  onNavigate,
  mobileOpen,
  onClose,
}: {
  activeView: MerchantView
  onNavigate: (view: MerchantView) => void
  mobileOpen: boolean
  onClose: () => void
}) {
  return (
    <>
      {mobileOpen && <button className="mobile-scrim" aria-label="关闭导航" onClick={onClose} />}
      <aside className={`sidebar ${mobileOpen ? 'sidebar-open' : ''}`}>
        <div className="brand-lockup">
          <div className="brand-mark"><Utensils size={19} strokeWidth={2.3} /></div>
          <div><strong>吃货</strong><span>需求驱动的餐桌</span></div>
          <button className="icon-button sidebar-close" aria-label="关闭导航" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="workspace-label">商家工作台</div>
        <nav className="primary-nav" aria-label="商家导航">
          {navItems.map(({ id, label, icon: Icon, count }) => (
            <button className={`nav-item ${activeView === id ? 'nav-item-active' : ''}`} key={id} onClick={() => { onNavigate(id); onClose() }}>
              <Icon size={18} />
              <span>{label}</span>
              {count && <span className="nav-count">{count}</span>}
            </button>
          ))}
        </nav>
        <div className="sidebar-spacer" />
        <div className="health-card">
          <div className="health-card-head"><ShieldCheck size={16} /><strong>今日经营健康度</strong><span>良好</span></div>
          <div className="health-bar"><span /></div>
          <p>资质有效，安全记录连续 12 天完整</p>
        </div>
        <div className="profile-chip">
          <div className="avatar avatar-orange">禾</div>
          <div className="profile-copy"><strong>禾下餐桌</strong><span>商家 ID · HX-018</span></div>
          <span className="online-dot" />
        </div>
      </aside>
    </>
  )
}

function Topbar({ onMenu }: { onMenu: () => void }) {
  return (
    <header className="topbar">
      <button className="icon-button mobile-menu-button" aria-label="打开导航" onClick={onMenu}><Menu size={20} /></button>
      <div className="breadcrumb"><span>商家工作台</span><ChevronRight size={14} /><strong>今日经营</strong></div>
      <div className="topbar-actions">
        <div className="verification-label"><span className="online-dot" />资质已核验</div>
        <button className="icon-button notification-button" aria-label="通知"><Bell size={19} /><span /></button>
        <div className="avatar avatar-orange">禾</div>
      </div>
    </header>
  )
}

function SectionHeader({ eyebrow, title, action, onAction }: { eyebrow?: string; title: string; action?: string; onAction?: () => void }) {
  return (
    <div className="section-header">
      <div>{eyebrow && <div className="eyebrow eyebrow-green">{eyebrow}</div>}<h2>{title}</h2></div>
      {action && <button className="text-button" onClick={onAction}>{action}<ArrowUpRight size={15} /></button>}
    </div>
  )
}

function MetricCard({ label, value, suffix, note, icon: Icon, tone }: { label: string; value: string; suffix?: string; note: string; icon: typeof Store; tone: string }) {
  return (
    <article className="metric-card">
      <div className={`metric-icon ${tone}`}><Icon size={19} /></div>
      <div className="metric-copy"><span>{label}</span><strong>{value}<small>{suffix}</small></strong><em>{note}</em></div>
      <ArrowUpRight className="metric-arrow" size={16} />
    </article>
  )
}

function DemandMiniCard({ demand, onQuote, onOpen }: { demand: MerchantDemand; onQuote: () => void; onOpen: () => void }) {
  const percent = Math.round((demand.joined / demand.target) * 100)
  return (
    <article className="demand-mini-card">
      <div className="demand-mini-head"><StatusPill>{demand.fit}</StatusPill><span>{demand.deadline}</span></div>
      <button className="card-title-button" onClick={onOpen}><div><span className="category-label">{demand.category}</span><h3>{demand.title}</h3></div><ArrowUpRight size={16} /></button>
      <div className="demand-mini-meta"><span><CalendarClock size={14} />{demand.schedule}</span><span><Truck size={14} />{demand.location}</span></div>
      <div className="tag-list compact-tags">{demand.tags.map((tag) => <span className="tag" key={tag}>{tag}</span>)}</div>
      <div className="demand-mini-foot">
        <div><div className="progress-label"><span><strong>{demand.joined}</strong>/{demand.target} 份意向</span><b>{percent}%</b></div><div className="progress-track"><span className="progress-value" style={{ width: `${percent}%` }} /></div></div>
        <button className="button button-small button-primary" onClick={onQuote}>{demand.status === '已报价' ? '查看报价' : demand.status === '已转预售' ? '查看方案' : '提交报价'}<ChevronRight size={14} /></button>
      </div>
    </article>
  )
}

function Dashboard({
  snapshot,
  onNavigate,
  onOpenDemand,
  onQuote,
}: {
  snapshot: MerchantSnapshot
  onNavigate: (view: MerchantView) => void
  onOpenDemand: (demand: MerchantDemand) => void
  onQuote: (demand: MerchantDemand) => void
}) {
  const readyDemands = snapshot.demands.filter((demand) => demand.status !== '已转预售')
  return (
    <>
      <section className="welcome-row">
        <div><div className="eyebrow eyebrow-green">星期三 · 9 月 2 日</div><h1>早上好，禾下餐桌。</h1><p>今天有 3 个附近需求与你的现有产能匹配，预计可转化为 75 份订单。</p></div>
        <div className="live-status"><span className="online-dot" /><span>门店营业中</span><button>管理状态<ChevronRight size={14} /></button></div>
      </section>
      <section className="metric-grid">
        <MetricCard label="今日待生产" value="28" suffix="份" note="较昨日 +12%" icon={ShoppingBasket} tone="green" />
        <MetricCard label="待报价需求" value="3" suffix="组" note="最高匹配度 96%" icon={ClipboardList} tone="orange" />
        <MetricCard label="本月成交额" value="¥18.6" suffix="k" note="较上月 +18.4%" icon={ArrowUpRight} tone="blue" />
        <MetricCard label="食品安全记录" value="12" suffix="天" note="连续完整记录" icon={ShieldCheck} tone="purple" />
      </section>
      <section className="dashboard-grid">
        <div>
          <SectionHeader eyebrow="优先处理" title="与你匹配的需求" action="进入需求大厅" onAction={() => onNavigate('demand-hall')} />
          <div className="demand-mini-list">{readyDemands.slice(0, 2).map((demand) => <DemandMiniCard key={demand.id} demand={demand} onQuote={() => onQuote(demand)} onOpen={() => onOpenDemand(demand)} />)}</div>
        </div>
        <div>
          <SectionHeader eyebrow="今天的履约" title="生产进度" action="查看全部" onAction={() => onNavigate('production')} />
          <ProductionPanel orders={snapshot.productionOrders} onNavigate={() => onNavigate('production')} />
        </div>
      </section>
      <section className="dashboard-bottom">
        <div className="data-panel">
          <SectionHeader eyebrow="合规状态" title="资质与食品安全" action="进入管理" onAction={() => onNavigate('qualification')} />
          <div className="compliance-list">
            <div className="compliance-row"><div className="compliance-icon green"><BadgeCheck size={16} /></div><div><strong>平台资质</strong><span>营业执照与食品经营许可证有效</span></div><StatusPill>已通过</StatusPill></div>
            <div className="compliance-row"><div className="compliance-icon orange"><ClipboardCheck size={16} /></div><div><strong>今日开档检查</strong><span>冷藏、消毒、明厨亮灶均已完成</span></div><StatusPill>已完成</StatusPill></div>
            <div className="compliance-row"><div className="compliance-icon blue"><FilePlus2 size={16} /></div><div><strong>待补充资料</strong><span>食品安全负责人联系电话</span></div><StatusPill>待补充</StatusPill></div>
          </div>
        </div>
        <div className="data-panel revenue-panel">
          <SectionHeader eyebrow="本周数据" title="按需生产表现" />
          <div className="revenue-highlight"><div><span>本周按需订单</span><strong>128 <small>份</small></strong></div><div className="revenue-trend">+24%<small>vs 上周</small></div></div>
          <div className="bar-chart" aria-label="本周订单趋势">
            {[38, 52, 43, 68, 56, 83, 74].map((value, index) => <div className="chart-column" key={index}><span style={{ height: `${value}%` }} /><small>{['一', '二', '三', '四', '五', '六', '日'][index]}</small></div>)}
          </div>
        </div>
      </section>
    </>
  )
}

function ProductionPanel({ orders, onNavigate }: { orders: ProductionOrder[]; onNavigate: () => void }) {
  const active = orders.filter((order) => order.status !== '已完成')
  return (
    <div className="production-panel">
      <div className="production-summary"><div className="production-number"><strong>{active.reduce((sum, order) => sum + order.quantity, 0)}</strong><span>份待履约</span></div><div className="production-progress"><span>今日进度</span><strong>56%</strong><div className="progress-track"><span className="progress-value" style={{ width: '56%' }} /></div></div></div>
      <div className="production-list">
        {active.slice(0, 3).map((order) => <div className="production-row" key={order.id}><div className={`production-status ${statusTone(order.status)}`}><PackageCheck size={15} /></div><div><strong>{order.product}</strong><span>{order.quantity} 份 · {order.window}</span></div><StatusPill>{order.status}</StatusPill></div>)}
      </div>
      <button className="panel-link" onClick={onNavigate}>打开生产看板<ChevronRight size={15} /></button>
    </div>
  )
}

function DemandHall({ demands, onOpen, onQuote }: { demands: MerchantDemand[]; onOpen: (demand: MerchantDemand) => void; onQuote: (demand: MerchantDemand) => void }) {
  const [filter, setFilter] = useState('全部')
  const filters = ['全部', '高匹配', '可尝试', '已报价']
  const filtered = demands.filter((demand) => filter === '全部' || (filter === '已报价' ? demand.status === '已报价' : demand.fit === filter))
  return (
    <section className="view-section">
      <section className="page-heading"><div><div className="eyebrow eyebrow-green">Demand intelligence</div><h1>需求大厅</h1><p>先看真实需求，再决定是否生产。平台已按区域、时间和规格完成聚合。</p></div><div className="page-heading-stat"><strong>{demands.filter((demand) => demand.status === '待报价').length}</strong><span>组待报价需求</span></div></section>
      <div className="merchant-toolbar"><div className="filter-chips">{filters.map((item) => <button className={`filter-chip ${filter === item ? 'filter-chip-active' : ''}`} key={item} onClick={() => setFilter(item)}>{item}</button>)}</div><div className="toolbar-note"><CalendarClock size={15} />仅显示未来 7 天的需求</div></div>
      <div className="demand-hall-grid">{filtered.map((demand) => <DemandMiniCard demand={demand} key={demand.id} onQuote={() => onQuote(demand)} onOpen={() => onOpen(demand)} />)}</div>
      {!filtered.length && <EmptyState title="没有符合条件的需求" description="调整筛选条件，或者稍后等待新需求聚合。" />}
    </section>
  )
}

function QuotesView({ quotes }: { quotes: MerchantQuote[] }) {
  return (
    <section className="view-section">
      <section className="page-heading"><div><div className="eyebrow eyebrow-green">Your offers</div><h1>我的报价</h1><p>报价提交后由平台审核，入选后会转成可预订的预售方案。</p></div><div className="page-heading-stat"><strong>{quotes.length}</strong><span>份历史报价</span></div></section>
      <div className="quotes-list">
        {quotes.map((quote) => <article className="quote-card" key={quote.id}><div className="quote-card-icon"><FileText size={21} /></div><div className="quote-card-content"><div className="quote-title-row"><div><span className="category-label">报价 #{quote.id.replace('quote-', '')}</span><h3>{quote.demandTitle}</h3></div><StatusPill>{quote.status}</StatusPill></div><p>{quote.note}</p><div className="quote-meta"><span>提交于 {quote.submittedAt}</span><span>规格 {quote.grams}g</span><span>最低 {quote.minOrder} 份</span></div></div><div className="quote-price"><span>报价</span><strong>¥{quote.price}</strong><small>/ 份</small><button className="text-button">查看详情<ChevronRight size={15} /></button></div></article>)}
      </div>
      {!quotes.length && <EmptyState title="还没有提交报价" description="去需求大厅看看附近有哪些适合你的生产机会。" />}
    </section>
  )
}

function ProductionView({ orders }: { orders: ProductionOrder[] }) {
  const [statusFilter, setStatusFilter] = useState('全部')
  const statuses = ['全部', '待生产', '制作中', '待交接']
  const filtered = orders.filter((order) => statusFilter === '全部' || order.status === statusFilter)
  return (
    <section className="view-section">
      <section className="page-heading"><div><div className="eyebrow eyebrow-green">Production board</div><h1>订单生产</h1><p>按批次记录生产、包装和交接，订单状态会同步给消费者。</p></div><button className="button button-primary"><Plus size={17} />创建生产批次</button></section>
      <div className="production-callout"><div className="callout-icon"><Info size={18} /></div><div><strong>今天有 2 个批次需要交接</strong><span>请在配送窗口前完成包装封签和骑手交接拍照。</span></div><button className="text-button">查看交接要求<ChevronRight size={15} /></button></div>
      <div className="production-toolbar"><div className="filter-chips">{statuses.map((item) => <button className={`filter-chip ${statusFilter === item ? 'filter-chip-active' : ''}`} key={item} onClick={() => setStatusFilter(item)}>{item}</button>)}</div><span className="toolbar-note">共 {filtered.length} 个订单</span></div>
      <div className="production-table-wrap">
        <table className="production-table"><thead><tr><th>订单 / 餐品</th><th>配送窗口</th><th>数量</th><th>生产批次</th><th>状态</th><th /></tr></thead><tbody>{filtered.map((order) => <tr key={order.id}><td><div className="table-product"><div className="table-product-icon"><Utensils size={16} /></div><div><strong>{order.product}</strong><span>{order.id} · {order.note}</span></div></div></td><td><span className="table-primary">{order.window}</span><small>{order.location}</small></td><td><strong>{order.quantity} 份</strong><small>已收款 ¥{order.paid}</small></td><td><span className="batch-code">{order.batch}</span></td><td><StatusPill>{order.status}</StatusPill></td><td><button className="icon-button" aria-label={`查看 ${order.product}`}><ChevronRight size={17} /></button></td></tr>)}</tbody></table>
      </div>
      {!filtered.length && <EmptyState title="没有待处理订单" description="当前筛选条件下没有生产任务。" />}
    </section>
  )
}

function QualificationView({ qualifications, onQualificationChange }: { qualifications: MerchantSnapshot['qualifications']; onQualificationChange: () => void }) {
  const [uploading, setUploading] = useState('')
  function simulateUpload(name: string) {
    setUploading(name)
    window.setTimeout(() => { setUploading(''); onQualificationChange() }, 650)
  }
  return (
    <section className="view-section">
      <section className="page-heading"><div><div className="eyebrow eyebrow-green">Verification center</div><h1>资质中心</h1><p>平台会核验证照有效期、经营地址和经营项目，证照变化请及时更新。</p></div><div className="verified-badge"><BadgeCheck size={18} />已完成基础核验</div></section>
      <div className="qualification-summary"><div className="qualification-score"><div className="score-ring"><strong>92</strong><span>分</span></div><div><strong>经营资料完整度</strong><span>3 项已通过，1 项待补充</span></div></div><div className="qualification-rule"><ShieldCheck size={17} /><span>平台每 6 个月复核一次商家信息</span></div></div>
      <div className="qualification-list">{qualifications.map((item) => <article className="qualification-row" key={item.name}><div className={`qualification-file ${statusTone(item.status)}`}><FileCheck2 size={20} /></div><div className="qualification-info"><strong>{item.name}</strong><span>{item.updated}</span></div><div className="qualification-valid"><span>有效期</span><strong>{item.expires}</strong></div><StatusPill>{item.status}</StatusPill><button className="button button-small button-quiet" onClick={() => simulateUpload(item.name)} disabled={uploading === item.name}>{uploading === item.name ? '上传中…' : item.status === '待补充' ? '补充资料' : '更新证照'}<Upload size={14} /></button></article>)}</div>
      <div className="qualification-note"><Info size={16} /><span>上传的证照仅用于平台审核和食品安全追溯，平台不会公开展示证件号码。</span></div>
    </section>
  )
}

function SafetyView({ records, onCreate }: { records: SafetyRecord[]; onCreate: (record: SafetyRecord) => void }) {
  const [modalOpen, setModalOpen] = useState(false)
  return (
    <section className="view-section">
      <section className="page-heading"><div><div className="eyebrow eyebrow-green">Food safety log</div><h1>食品安全</h1><p>记录日常检查、生产批次和异常处理，让每份餐品都有迹可查。</p></div><button className="button button-primary" onClick={() => setModalOpen(true)}><Plus size={17} />新建记录</button></section>
      <div className="safety-overview"><div><div className="safety-overview-icon"><ShieldCheck size={22} /></div><div><strong>安全记录连续 12 天</strong><span>今日开档检查已完成，明厨亮灶在线</span></div></div><div className="safety-stats"><span><strong>{records.length}</strong>条记录</span><span><strong>0</strong>条逾期</span><span><strong>100%</strong>批次留档率</span></div></div>
      <div className="safety-filter-row"><div className="filter-chips"><button className="filter-chip filter-chip-active">全部记录</button><button className="filter-chip">日常检查</button><button className="filter-chip">批次留档</button><button className="filter-chip">投诉跟进</button></div><button className="sort-button">最近更新<ChevronRight size={15} /></button></div>
      <div className="safety-list">{records.map((record) => <article className="safety-row" key={record.id}><div className={`safety-record-icon ${statusTone(record.status)}`}>{record.type === '批次留档' ? <PackageCheck size={18} /> : record.type === '投诉跟进' ? <AlertCircle size={18} /> : <ClipboardCheck size={18} />}</div><div className="safety-record-main"><div className="safety-record-title"><span className="category-label">{record.type}</span><StatusPill>{record.status}</StatusPill></div><h3>{record.title}</h3><p>{record.detail}</p><span className="safety-record-meta">{record.date} · 负责人 {record.owner}</span></div><button className="icon-button" aria-label={`查看 ${record.title}`}><ChevronRight size={17} /></button></article>)}</div>
      {!records.length && <EmptyState title="还没有食品安全记录" description="新建一条日常检查或批次留档记录。" />}
      {modalOpen && <SafetyRecordDialog onClose={() => setModalOpen(false)} onSubmit={(record) => { onCreate(record); setModalOpen(false) }} />}
    </section>
  )
}

function SafetyRecordDialog({ onClose, onSubmit }: { onClose: () => void; onSubmit: (record: SafetyRecord) => void }) {
  const [saving, setSaving] = useState(false)
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    const form = new FormData(event.currentTarget)
    const record = await createSafetyRecord({
      type: String(form.get('type') ?? '日常检查') as SafetyRecord['type'],
      title: String(form.get('title') ?? '日常检查记录'),
      owner: String(form.get('owner') ?? '门店负责人'),
      detail: String(form.get('detail') ?? ''),
    })
    setSaving(false)
    onSubmit(record)
  }
  return (
    <div className="modal-layer modal-centered"><button className="modal-scrim" aria-label="关闭新建记录" onClick={onClose} /><form className="confirm-dialog safety-dialog" onSubmit={handleSubmit}><button type="button" className="icon-button dialog-close" aria-label="关闭" onClick={onClose}><X size={18} /></button><div className="dialog-icon"><ClipboardCheck size={22} /></div><div className="eyebrow eyebrow-green">New safety log</div><h2>新建食品安全记录</h2><div className="dialog-form"><label className="field-label">记录类型<select name="type" className="text-input"><option>日常检查</option><option>批次留档</option><option>投诉跟进</option><option>整改记录</option></select></label><label className="field-label">记录标题<input name="title" className="text-input" placeholder="例如：开档前环境与设备检查" required /></label><label className="field-label">负责人<input name="owner" className="text-input" defaultValue="门店负责人" /></label><label className="field-label">记录说明<textarea name="detail" className="text-input textarea" placeholder="填写检查结果、批次信息或处置进展。" /></label></div><div className="dialog-actions"><button type="button" className="button button-quiet" onClick={onClose}>取消</button><button className="button button-primary" disabled={saving}>{saving ? '正在保存…' : '保存记录'}{!saving && <Check size={16} />}</button></div></form></div>
  )
}

function DemandDetail({ demand, onClose, onQuote }: { demand: MerchantDemand; onClose: () => void; onQuote: () => void }) {
  const percent = Math.round((demand.joined / demand.target) * 100)
  return (
    <div className="modal-layer"><button className="modal-scrim" aria-label="关闭需求详情" onClick={onClose} /><aside className="detail-drawer" role="dialog" aria-modal="true" aria-label="需求详情"><div className="drawer-header"><span className="eyebrow eyebrow-green">需求机会</span><button className="icon-button" aria-label="关闭" onClick={onClose}><X size={19} /></button></div><div className="drawer-title-row"><div><span className="category-label">{demand.category}</span><h2>{demand.title}</h2></div><StatusPill>{demand.fit}</StatusPill></div><div className="match-callout"><div className="match-score">96<small>%</small></div><div><strong>与你的经营能力高度匹配</strong><span>{demand.fitNote}</span></div></div><div className="detail-meta-grid"><div><span>需求区域</span><strong><Truck size={15} />{demand.location}</strong></div><div><span>配送窗口</span><strong><CalendarClock size={15} />{demand.schedule}</strong></div><div><span>用户预算</span><strong>{demand.budget}</strong></div><div><span>需求进度</span><strong>{demand.joined}/{demand.target} 份</strong></div></div><div className="drawer-section"><div className="progress-label"><span><strong>{demand.joined}</strong> 份有效需求</span><b>{percent}%</b></div><div className="progress-track"><span className="progress-value" style={{ width: `${percent}%` }} /></div><p className="microcopy">{demand.deadline} · 达到最低数量后消费者将收到预售方案。</p></div><div className="drawer-section"><span className="field-label-title">用户共同要求</span><div className="tag-list drawer-tags">{demand.tags.map((tag) => <span className="tag" key={tag}><Check size={13} />{tag}</span>)}</div></div><div className="drawer-footer"><div><span>建议报价区间</span><strong>{demand.budget} / 份</strong></div><button className="button button-primary" onClick={onQuote}>{demand.status === '已报价' ? '查看我的报价' : '提交报价'}<ArrowUpRight size={17} /></button></div></aside></div>
  )
}

function QuoteDialog({ demand, onClose, onSubmitted }: { demand: MerchantDemand; onClose: () => void; onSubmitted: (quote: MerchantQuote) => void }) {
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setSaving(true)
    const form = new FormData(event.currentTarget)
    const price = Number(form.get('price'))
    if (!price || price < 1) {
      setError('请填写有效的单份报价')
      setSaving(false)
      return
    }
    const quote = await submitMerchantQuote({
      demandId: demand.id,
      demandTitle: demand.title,
      price,
      grams: Number(form.get('grams') ?? 350),
      minOrder: Number(form.get('minOrder') ?? demand.target),
      note: String(form.get('note') ?? ''),
    })
    setSaving(false)
    onSubmitted(quote)
  }
  return (
    <div className="modal-layer modal-centered"><button className="modal-scrim" aria-label="关闭报价" onClick={onClose} /><form className="confirm-dialog quote-dialog" onSubmit={handleSubmit}><button type="button" className="icon-button dialog-close" aria-label="关闭" onClick={onClose}><X size={18} /></button><div className="dialog-icon"><FileText size={21} /></div><div className="eyebrow eyebrow-green">Submit offer</div><h2>为「{demand.title}」报价</h2><p>{demand.joined} 份有效需求 · 用户预算 {demand.budget}</p><div className="quote-form-grid"><label className="field-label">单份报价<div className="input-prefix"><span>¥</span><input name="price" className="text-input" defaultValue={demand.budget.replace(/[^\d]+.*/, '')} inputMode="decimal" /></div></label><label className="field-label">餐品重量<input name="grams" className="text-input" defaultValue="350" inputMode="numeric" /></label><label className="field-label">最低生产量<input name="minOrder" className="text-input" defaultValue={demand.target} inputMode="numeric" /></label></div><label className="field-label">方案说明<textarea name="note" className="text-input textarea" defaultValue={`${demand.tags.join('、')}，集中配送。`} /></label>{error && <div className="inline-error"><AlertCircle size={16} />{error}</div>}<div className="dialog-actions"><button type="button" className="button button-quiet" onClick={onClose}>取消</button><button className="button button-primary" disabled={saving}>{saving ? '正在提交…' : '提交报价'}{!saving && <ArrowUpRight size={16} />}</button></div></form></div>
  )
}

function EmptyState({ title, description }: { title: string; description: string }) {
  return <div className="empty-state"><div className="empty-icon"><FileText size={22} /></div><h3>{title}</h3><p>{description}</p></div>
}

function LoadingState() {
  return <section className="loading-state"><div className="loading-orb" /><div className="skeleton-line skeleton-line-long" /><div className="skeleton-line skeleton-line-short" /><div className="skeleton-grid"><span /><span /><span /></div><p>正在同步需求与生产数据…</p></section>
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return <section className="error-state"><div className="error-icon"><AlertCircle size={25} /></div><h2>工作台暂时无法打开</h2><p>{message}</p><button className="button button-primary" onClick={onRetry}>重新加载</button></section>
}

export default function App() {
  const [activeView, setActiveView] = useState<MerchantView>('dashboard')
  const [mobileOpen, setMobileOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [snapshot, setSnapshot] = useState<MerchantSnapshot>({ demands: [], quotes: [], productionOrders: [], qualifications: [], safetyRecords: [] })
  const [fallbackMessage, setFallbackMessage] = useState('')
  const [loadError, setLoadError] = useState('')
  const [selectedDemand, setSelectedDemand] = useState<MerchantDemand | null>(null)
  const [quoteTarget, setQuoteTarget] = useState<MerchantDemand | null>(null)
  const [toast, setToast] = useState('')

  useEffect(() => {
    let active = true
    loadMerchantSnapshot().then((result) => {
      if (!active) return
      setSnapshot(result.data)
      setFallbackMessage(result.isFallback ? result.message : '')
    }).catch(() => active && setLoadError('数据加载失败，请刷新重试')).finally(() => active && setLoading(false))
    return () => { active = false }
  }, [])

  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(''), 3000)
    return () => window.clearTimeout(timer)
  }, [toast])

  function navigate(view: MerchantView) {
    setActiveView(view)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  function openQuote(demand: MerchantDemand) {
    setSelectedDemand(null)
    setQuoteTarget(demand)
  }

  async function handleQuoteSubmitted(quote: MerchantQuote) {
    setSnapshot((current) => ({ ...current, quotes: [quote, ...current.quotes], demands: current.demands.map((demand) => demand.id === quote.demandId ? { ...demand, status: '已报价' } : demand) }))
    setQuoteTarget(null)
    setToast('报价已提交，平台审核后会通知你')
    navigate('quotes')
  }

  async function handleQualificationChange() {
    setToast('资料已提交，平台会在 1–2 个工作日内复核')
  }

  async function handleCreateSafety(record: SafetyRecord) {
    setSnapshot((current) => ({ ...current, safetyRecords: [record, ...current.safetyRecords] }))
    setToast('食品安全记录已保存')
  }

  function renderView() {
    if (loading) return <LoadingState />
    if (loadError) return <ErrorState message={loadError} onRetry={() => window.location.reload()} />
    if (activeView === 'dashboard') return <Dashboard snapshot={snapshot} onNavigate={navigate} onOpenDemand={setSelectedDemand} onQuote={openQuote} />
    if (activeView === 'demand-hall') return <DemandHall demands={snapshot.demands} onOpen={setSelectedDemand} onQuote={openQuote} />
    if (activeView === 'quotes') return <QuotesView quotes={snapshot.quotes} />
    if (activeView === 'production') return <ProductionView orders={snapshot.productionOrders} />
    if (activeView === 'qualification') return <QualificationView qualifications={snapshot.qualifications} onQualificationChange={handleQualificationChange} />
    return <SafetyView records={snapshot.safetyRecords} onCreate={handleCreateSafety} />
  }

  return (
    <div className="app-shell">
      <MerchantSidebar activeView={activeView} onNavigate={navigate} mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />
      <div className="main-shell">
        <Topbar onMenu={() => setMobileOpen(true)} />
        <main className="main-content">
          {fallbackMessage && <div className="fallback-notice"><AlertCircle size={16} />{fallbackMessage}，当前操作可在本地演示<button className="notice-close" aria-label="关闭提示" onClick={() => setFallbackMessage('')}><X size={15} /></button></div>}
          {renderView()}
        </main>
      </div>
      {selectedDemand && <DemandDetail demand={selectedDemand} onClose={() => setSelectedDemand(null)} onQuote={() => openQuote(selectedDemand)} />}
      {quoteTarget && <QuoteDialog demand={quoteTarget} onClose={() => setQuoteTarget(null)} onSubmitted={handleQuoteSubmitted} />}
      {toast && <div className="toast"><Check size={16} />{toast}</div>}
    </div>
  )
}
