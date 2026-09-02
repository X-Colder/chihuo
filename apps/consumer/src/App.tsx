import {
  AlertCircle,
  ArrowUpRight,
  Bell,
  Check,
  ChevronRight,
  CircleHelp,
  Clock3,
  FileText,
  HeartPulse,
  Leaf,
  MapPin,
  Menu,
  PackageCheck,
  Plus,
  Search,
  ShoppingBag,
  SlidersHorizontal,
  Sparkles,
  Utensils,
  X,
} from 'lucide-react'
import { FormEvent, useEffect, useMemo, useState } from 'react'
import { createDemand, joinDemand, loadConsumerSnapshot, placeOrder } from './lib/api'
import type { ConsumerOrder, ConsumerView, Demand, Product } from './types'

const navItems: Array<{ id: ConsumerView; label: string; icon: typeof Utensils }> = [
  { id: 'overview', label: '需求广场', icon: Utensils },
  { id: 'demands', label: '我的需求', icon: HeartPulse },
  { id: 'publish', label: '发布需求', icon: Plus },
  { id: 'presale', label: '预售商品', icon: ShoppingBag },
  { id: 'orders', label: '我的订单', icon: PackageCheck },
]

const number = new Intl.NumberFormat('zh-CN')

function statusTone(status: Demand['status'] | ConsumerOrder['status']) {
  if (status === '已达标' || status === '已完成') return 'success'
  if (status === '即将开售' || status === '配送中') return 'accent'
  if (status === '制作中') return 'info'
  return 'neutral'
}

function Progress({ value, total }: { value: number; total: number }) {
  return (
    <div className="progress-track" aria-label={`已加入 ${value} 人，共需 ${total} 人`}>
      <span className="progress-value" style={{ width: `${Math.min(100, (value / total) * 100)}%` }} />
    </div>
  )
}

function StatusPill({ children }: { children: string }) {
  return <span className={`status-pill ${statusTone(children as Demand['status'])}`}>{children}</span>
}

function ConsumerSidebar({
  activeView,
  onNavigate,
  mobileOpen,
  onClose,
}: {
  activeView: ConsumerView
  onNavigate: (view: ConsumerView) => void
  mobileOpen: boolean
  onClose: () => void
}) {
  return (
    <>
      {mobileOpen && <button className="mobile-scrim" aria-label="关闭导航" onClick={onClose} />}
      <aside className={`sidebar ${mobileOpen ? 'sidebar-open' : ''}`}>
        <div className="brand-lockup">
          <div className="brand-mark">
            <Utensils size={19} strokeWidth={2.3} />
          </div>
          <div>
            <strong>吃货</strong>
            <span>需求驱动的餐桌</span>
          </div>
          <button className="icon-button sidebar-close" aria-label="关闭导航" onClick={onClose}>
            <X size={18} />
          </button>
        </div>
        <div className="workspace-label">消费者工作台</div>
        <nav className="primary-nav" aria-label="主导航">
          {navItems.map(({ id, label, icon: Icon }) => (
            <button
              className={`nav-item ${activeView === id ? 'nav-item-active' : ''}`}
              key={id}
              onClick={() => {
                onNavigate(id)
                onClose()
              }}
            >
              <Icon size={18} />
              <span>{label}</span>
              {id === 'orders' && <span className="nav-count">2</span>}
            </button>
          ))}
        </nav>
        <div className="sidebar-spacer" />
        <div className="help-card">
          <div className="help-icon">
            <CircleHelp size={17} />
          </div>
          <div>
            <strong>需要帮忙？</strong>
            <span>订单或需求有问题，随时联系我们</span>
          </div>
          <ChevronRight size={16} />
        </div>
        <div className="profile-chip">
          <div className="avatar avatar-cream">林</div>
          <div className="profile-copy">
            <strong>林晓雨</strong>
            <span>科技园 A 区</span>
          </div>
          <span className="online-dot" />
        </div>
      </aside>
    </>
  )
}

function Topbar({ onMenu }: { onMenu: () => void }) {
  return (
    <header className="topbar">
      <button className="icon-button mobile-menu-button" aria-label="打开导航" onClick={onMenu}>
        <Menu size={20} />
      </button>
      <div className="breadcrumb">
        <span>消费者工作台</span>
        <ChevronRight size={14} />
        <strong>今日概览</strong>
      </div>
      <div className="topbar-actions">
        <button className="icon-button notification-button" aria-label="通知">
          <Bell size={19} />
          <span />
        </button>
        <div className="topbar-location">
          <MapPin size={15} />
          <span>科技园 A 区</span>
        </div>
        <div className="avatar avatar-navy">林</div>
      </div>
    </header>
  )
}

function SectionHeader({
  eyebrow,
  title,
  action,
  onAction,
}: {
  eyebrow?: string
  title: string
  action?: string
  onAction?: () => void
}) {
  return (
    <div className="section-header">
      <div>
        {eyebrow && <div className="eyebrow">{eyebrow}</div>}
        <h2>{title}</h2>
      </div>
      {action && (
        <button className="text-button" onClick={onAction}>
          {action}
          <ArrowUpRight size={15} />
        </button>
      )}
    </div>
  )
}

function DemandCard({ demand, onOpen, onJoin }: { demand: Demand; onOpen: () => void; onJoin: () => void }) {
  const percentage = Math.round((demand.joined / demand.target) * 100)
  return (
    <article className="demand-card">
      <div className="card-topline">
        <span className="category-label">{demand.category}</span>
        <StatusPill>{demand.status}</StatusPill>
      </div>
      <button className="card-title-button" onClick={onOpen}>
        <h3>{demand.title}</h3>
        <ArrowUpRight size={17} />
      </button>
      <div className="demand-meta">
        <span>
          <MapPin size={14} />
          {demand.location}
        </span>
        <span>
          <Clock3 size={14} />
          {demand.schedule}
        </span>
      </div>
      <div className="tag-list">
        {demand.tags.map((tag) => (
          <span className="tag" key={tag}>
            {tag}
          </span>
        ))}
      </div>
      <div className="progress-block">
        <div className="progress-label">
          <span>
            <strong>{demand.joined}</strong>/{demand.target} 人已加入
          </span>
          <b>{percentage}%</b>
        </div>
        <Progress value={demand.joined} total={demand.target} />
      </div>
      <div className="card-footer">
        <span className="budget-text">预算 {demand.budget}</span>
        {demand.joined >= demand.target ? (
          <button className="button button-small button-primary" onClick={onOpen}>
            查看详情
            <ChevronRight size={15} />
          </button>
        ) : (
          <button className="button button-small button-secondary" onClick={onJoin}>
            加入需求
            <Plus size={15} />
          </button>
        )}
      </div>
    </article>
  )
}

function ProductCard({ product, onOrder }: { product: Product; onOrder: () => void }) {
  return (
    <article className="product-card">
      <div className={`product-visual ${product.accent}`}>
        <div className="product-visual-copy">
          <span>本批次</span>
          <strong>{product.grams}g</strong>
        </div>
        <Utensils size={31} strokeWidth={1.45} />
        <span className="product-batch">可追溯</span>
      </div>
      <div className="product-body">
        <div className="card-topline">
          <span className="category-label">{product.merchant}</span>
          <span className="stock-label">余 {product.stock} 份</span>
        </div>
        <h3>{product.title}</h3>
        <div className="tag-list compact-tags">
          {product.tags.map((tag) => (
            <span className="tag" key={tag}>
              {tag}
            </span>
          ))}
        </div>
        <div className="product-meta">
          <span>
            <Clock3 size={14} />
            {product.delivery}
          </span>
        </div>
        <div className="product-buy-row">
          <div>
            <strong className="price">¥{product.price}</strong>
            <del>¥{product.originalPrice}</del>
          </div>
          <button className="button button-small button-primary" onClick={onOrder}>
            立即预订
            <ShoppingBag size={15} />
          </button>
        </div>
      </div>
    </article>
  )
}

function Overview({
  demands,
  products,
  onNavigate,
  onOpenDemand,
  onJoinDemand,
  onOrder,
}: {
  demands: Demand[]
  products: Product[]
  onNavigate: (view: ConsumerView) => void
  onOpenDemand: (demand: Demand) => void
  onJoinDemand: (demand: Demand) => void
  onOrder: (product: Product) => void
}) {
  const activeDemands = demands.filter((demand) => demand.status !== '已达标')
  return (
    <>
      <section className="welcome-row">
        <div>
          <div className="eyebrow eyebrow-teal">星期三 · 9 月 2 日</div>
          <h1>把想吃的，<em>聚成一桌。</em></h1>
          <p>今天有 {number.format(activeDemands.length)} 个需求正在附近聚合，人数够了就让商家按需开做。</p>
        </div>
        <button className="button button-primary button-large" onClick={() => onNavigate('publish')}>
          <Plus size={18} />
          发布我的需求
        </button>
      </section>

      <section className="hero-strip">
        <div className="hero-strip-copy">
          <div className="hero-kicker">
            <Sparkles size={15} />
            本周共创
          </div>
          <h2>少一点将就，多一点刚刚好。</h2>
          <p>把一个人的小要求，变成一群人的好选择。</p>
        </div>
        <div className="hero-metrics">
          <div>
            <strong>128</strong>
            <span>本周已聚合份数</span>
          </div>
          <div>
            <strong>16</strong>
            <span>正在报价的需求</span>
          </div>
          <div>
            <strong>4.9</strong>
            <span>用户满意度</span>
          </div>
        </div>
      </section>

      <section>
        <SectionHeader
          eyebrow="附近正在发生"
          title="需求广场"
          action="查看全部"
          onAction={() => onNavigate('demands')}
        />
        <div className="demand-grid">
          {activeDemands.slice(0, 3).map((demand) => (
            <DemandCard
              demand={demand}
              key={demand.id}
              onOpen={() => onOpenDemand(demand)}
              onJoin={() => onJoinDemand(demand)}
            />
          ))}
        </div>
      </section>

      <section className="two-column-section">
        <div>
          <SectionHeader
            eyebrow="商家已接单"
            title="正在预售"
            action="全部商品"
            onAction={() => onNavigate('presale')}
          />
          <div className="product-list">
            {products.slice(0, 2).map((product) => (
              <ProductCard product={product} key={product.id} onOrder={() => onOrder(product)} />
            ))}
          </div>
        </div>
        <div>
          <SectionHeader eyebrow="你的进度" title="最近订单" action="订单记录" onAction={() => onNavigate('orders')} />
          <RecentOrderPanel />
        </div>
      </section>
    </>
  )
}

function RecentOrderPanel() {
  return (
    <div className="recent-panel">
      <div className="recent-summary">
        <div className="summary-icon">
          <PackageCheck size={21} />
        </div>
        <div>
          <strong>1 个订单正在配送</strong>
          <span>预计 11:50 前送达</span>
        </div>
        <span className="live-dot">实时</span>
      </div>
      <div className="order-timeline">
        <div className="timeline-step timeline-step-done">
          <span>
            <Check size={13} />
          </span>
          <div>
            <strong>商家已打包</strong>
            <small>11:18 · 禾下餐桌</small>
          </div>
        </div>
        <div className="timeline-line" />
        <div className="timeline-step timeline-step-active">
          <span>
            <PackageCheck size={13} />
          </span>
          <div>
            <strong>骑手配送中</strong>
            <small>骑手已取餐，距离你 0.8km</small>
          </div>
        </div>
        <div className="timeline-line timeline-line-muted" />
        <div className="timeline-step">
          <span />
          <div>
            <strong>送达</strong>
            <small>预计 11:50</small>
          </div>
        </div>
      </div>
      <button className="panel-link">
        查看配送详情
        <ChevronRight size={15} />
      </button>
    </div>
  )
}

function DemandsView({
  demands,
  onOpen,
  onJoin,
  onNavigate,
}: {
  demands: Demand[]
  onOpen: (demand: Demand) => void
  onJoin: (demand: Demand) => void
  onNavigate: (view: ConsumerView) => void
}) {
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('全部')
  const categories = ['全部', ...Array.from(new Set(demands.map((demand) => demand.category)))]
  const filteredDemands = demands.filter((demand) => {
    const matchesCategory = category === '全部' || demand.category === category
    const query = search.trim().toLowerCase()
    return matchesCategory && (!query || `${demand.title}${demand.tags.join('')}`.toLowerCase().includes(query))
  })

  return (
    <section className="view-section">
      <section className="page-heading">
        <div>
          <div className="eyebrow eyebrow-teal">Demand board</div>
          <h1>需求广场</h1>
          <p>附近的人正在提出什么？找到与你口味一致的那一组。</p>
        </div>
        <button className="button button-primary" onClick={() => onNavigate('publish')}>
          <Plus size={17} />
          发布需求
        </button>
      </section>
      <div className="filter-bar">
        <label className="search-field">
          <Search size={17} />
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索餐品、标签或场景" />
        </label>
        <div className="filter-chips">
          {categories.map((item) => (
            <button className={`filter-chip ${category === item ? 'filter-chip-active' : ''}`} key={item} onClick={() => setCategory(item)}>
              {item}
            </button>
          ))}
        </div>
        <button className="icon-button filter-settings" aria-label="更多筛选">
          <SlidersHorizontal size={17} />
        </button>
      </div>
      {filteredDemands.length ? (
        <div className="demand-grid demand-grid-wide">
          {filteredDemands.map((demand) => (
            <DemandCard demand={demand} key={demand.id} onOpen={() => onOpen(demand)} onJoin={() => onJoin(demand)} />
          ))}
        </div>
      ) : (
        <EmptyState title="暂时没有匹配的需求" description="换个关键词或发布一个新的需求。" action="发布需求" onAction={() => onNavigate('publish')} />
      )}
    </section>
  )
}

function PublishView({ onCreated }: { onCreated: (demand: Demand) => void }) {
  const [submitted, setSubmitted] = useState(false)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError('')
    setSaving(true)
    const formData = new FormData(event.currentTarget)
    const title = String(formData.get('title') ?? '').trim()
    if (!title) {
      setFormError('请先填写想吃的餐品或场景')
      setSaving(false)
      return
    }
    const payload = {
      title,
      category: String(formData.get('category') ?? '午餐'),
      location: String(formData.get('location') ?? '科技园 A 区'),
      schedule: String(formData.get('schedule') ?? '工作日 11:30–12:30'),
      target: Number(formData.get('target') ?? 20),
      budget: `¥${formData.get('budgetMin') ?? 20}–${formData.get('budgetMax') ?? 30}`,
      tags: ['少油', '定量 350g'],
      description: String(formData.get('description') ?? ''),
      merchantHint: '你的需求已进入聚合，达到人数后会通知你。',
    }
    const demand = await createDemand(payload)
    onCreated(demand)
    setSaving(false)
    setSubmitted(true)
    event.currentTarget.reset()
  }

  if (submitted) {
    return (
      <section className="view-section">
        <div className="success-state">
          <div className="success-icon">
            <Check size={27} />
          </div>
          <div className="eyebrow eyebrow-teal">需求已发布</div>
          <h1>等一等，大家会找到你。</h1>
          <p>我们会把相近需求聚合给附近商家。达到人数后，你会收到报价和开售提醒。</p>
          <button className="button button-primary" onClick={() => setSubmitted(false)}>
            再发布一个需求
            <Plus size={17} />
          </button>
        </div>
      </section>
    )
  }

  return (
    <section className="view-section">
      <section className="page-heading">
        <div>
          <div className="eyebrow eyebrow-teal">Start a demand</div>
          <h1>发布你的需求</h1>
          <p>把必须满足的条件写清楚，其他偏好可以留给商家发挥。</p>
        </div>
      </section>
      <form className="publish-layout" onSubmit={handleSubmit}>
        <div className="form-card">
          <div className="form-card-heading">
            <div className="section-number">01</div>
            <div>
              <h2>先说说想吃什么</h2>
              <p>一句话描述餐品或用餐场景即可。</p>
            </div>
          </div>
          <label className="field-label">
            餐品或场景
            <input name="title" className="text-input" placeholder="例如：低油轻食工作餐" />
          </label>
          <div className="field-grid">
            <label className="field-label">
              分类
              <select name="category" className="text-input">
                <option>午餐</option>
                <option>家庭餐</option>
                <option>早餐</option>
                <option>烘焙</option>
                <option>饮品</option>
              </select>
            </label>
            <label className="field-label">
              配送区域
              <select name="location" className="text-input">
                <option>科技园 A 区</option>
                <option>云杉社区</option>
                <option>南站办公区</option>
                <option>望江路 3 公里内</option>
              </select>
            </label>
          </div>
          <label className="field-label">
            具体要求
            <textarea name="description" className="text-input textarea" placeholder="例如：少油、不放花生，主食和蔬菜分区，午休时段集中配送。" />
          </label>
        </div>
        <div className="form-card">
          <div className="form-card-heading">
            <div className="section-number">02</div>
            <div>
              <h2>设置成团条件</h2>
              <p>范围越清楚，商家越容易给出可执行的方案。</p>
            </div>
          </div>
          <div className="field-grid">
            <label className="field-label">
              用餐时间
              <select name="schedule" className="text-input">
                <option>工作日 11:30–12:30</option>
                <option>工作日 08:00–09:00</option>
                <option>周三 17:30–18:30</option>
                <option>周六 14:00–16:00</option>
              </select>
            </label>
            <label className="field-label">
              目标人数
              <select name="target" className="text-input">
                <option value="20">20 人</option>
                <option value="30">30 人</option>
                <option value="40">40 人</option>
                <option value="50">50 人</option>
              </select>
            </label>
          </div>
          <div className="field-grid">
            <label className="field-label">
              可接受最低价
              <div className="input-prefix">
                <span>¥</span>
                <input name="budgetMin" className="text-input" defaultValue="20" inputMode="numeric" />
              </div>
            </label>
            <label className="field-label">
              可接受最高价
              <div className="input-prefix">
                <span>¥</span>
                <input name="budgetMax" className="text-input" defaultValue="30" inputMode="numeric" />
              </div>
            </label>
          </div>
          <div className="choice-block">
            <span className="field-label-title">必须满足</span>
            <div className="choice-list">
              {['少油', '少盐', '不含花生', '定量 350g'].map((item) => (
                <label className="choice-item" key={item}>
                  <input type="checkbox" defaultChecked={item === '少油' || item === '定量 350g'} />
                  <span>{item}</span>
                </label>
              ))}
            </div>
          </div>
          {formError && (
            <div className="inline-error">
              <AlertCircle size={16} />
              {formError}
            </div>
          )}
          <div className="form-actions">
            <span>
              <Leaf size={15} />
              发布后可随时修改或退出
            </span>
            <button className="button button-primary" type="submit" disabled={saving}>
              {saving ? '正在发布…' : '发布需求'}
              {!saving && <ArrowUpRight size={17} />}
            </button>
          </div>
        </div>
      </form>
    </section>
  )
}

function PresaleView({ products, onOrder }: { products: Product[]; onOrder: (product: Product) => void }) {
  return (
    <section className="view-section">
      <section className="page-heading">
        <div>
          <div className="eyebrow eyebrow-teal">Ready to order</div>
          <h1>预售商品</h1>
          <p>这些餐品已经由商家确认生产，按约定时间集中制作和配送。</p>
        </div>
        <div className="page-heading-stat">
          <strong>{products.length}</strong>
          <span>个可预订方案</span>
        </div>
      </section>
      <div className="presale-toolbar">
        <div className="toolbar-note">
          <span className="live-dot" />
          订单达到最低生产量后，商家开始备餐
        </div>
        <button className="sort-button">
          默认排序
          <ChevronRight size={15} />
        </button>
      </div>
      <div className="product-grid">
        {products.map((product) => (
          <ProductCard product={product} key={product.id} onOrder={() => onOrder(product)} />
        ))}
      </div>
      {!products.length && <EmptyState title="还没有可预订商品" description="附近的商家正在查看需求，稍后再来看看。" />}
    </section>
  )
}

function OrdersView({ orders }: { orders: ConsumerOrder[] }) {
  return (
    <section className="view-section">
      <section className="page-heading">
        <div>
          <div className="eyebrow eyebrow-teal">Your orders</div>
          <h1>我的订单</h1>
          <p>每一笔订单都关联到生产批次，进度和售后记录清晰可查。</p>
        </div>
      </section>
      <div className="order-tabs">
        <button className="order-tab order-tab-active">全部 <span>{orders.length}</span></button>
        <button className="order-tab">进行中 <span>{orders.filter((order) => order.status !== '已完成').length}</span></button>
        <button className="order-tab">已完成</button>
      </div>
      <div className="orders-list">
        {orders.map((order) => (
          <article className="order-card" key={order.id}>
            <div className="order-card-main">
              <div className="order-thumb">
                <Utensils size={23} />
              </div>
              <div className="order-info">
                <div className="order-title-row">
                  <h3>{order.product}</h3>
                  <StatusPill>{order.status}</StatusPill>
                </div>
                <p>{order.merchant} · {order.id}</p>
                <span className="order-meta">
                  <Clock3 size={14} />
                  {order.time}
                  <MapPin size={14} />
                  {order.address}
                </span>
              </div>
            </div>
            <div className="order-card-side">
              <div>
                <span>实付</span>
                <strong>¥{order.amount.toFixed(2)}</strong>
              </div>
              <button className="text-button">
                订单详情
                <ChevronRight size={15} />
              </button>
            </div>
          </article>
        ))}
      </div>
      {!orders.length && <EmptyState title="还没有订单" description="加入一个需求或预订一份预售餐品，订单会出现在这里。" />}
    </section>
  )
}

function EmptyState({ title, description, action, onAction }: { title: string; description: string; action?: string; onAction?: () => void }) {
  return (
    <div className="empty-state">
      <div className="empty-icon">
        <FileText size={22} />
      </div>
      <h3>{title}</h3>
      <p>{description}</p>
      {action && onAction && <button className="button button-secondary" onClick={onAction}>{action}<Plus size={16} /></button>}
    </div>
  )
}

function DemandDetail({
  demand,
  onClose,
  onJoin,
}: {
  demand: Demand
  onClose: () => void
  onJoin: () => void
}) {
  const percentage = Math.round((demand.joined / demand.target) * 100)
  return (
    <div className="modal-layer">
      <button className="modal-scrim" aria-label="关闭详情" onClick={onClose} />
      <aside className="detail-drawer" role="dialog" aria-modal="true" aria-label="需求详情">
        <div className="drawer-header">
          <span className="eyebrow eyebrow-teal">需求详情</span>
          <button className="icon-button" aria-label="关闭详情" onClick={onClose}><X size={19} /></button>
        </div>
        <div className="drawer-title-row">
          <div>
            <span className="category-label">{demand.category}</span>
            <h2>{demand.title}</h2>
          </div>
          <StatusPill>{demand.status}</StatusPill>
        </div>
        <p className="drawer-description">{demand.description}</p>
        <div className="detail-meta-grid">
          <div><span>配送区域</span><strong><MapPin size={15} />{demand.location}</strong></div>
          <div><span>用餐时间</span><strong><Clock3 size={15} />{demand.schedule}</strong></div>
          <div><span>预算范围</span><strong>{demand.budget}</strong></div>
          <div><span>成团目标</span><strong>{demand.target} 人</strong></div>
        </div>
        <div className="drawer-section">
          <div className="progress-label"><span><strong>{demand.joined}</strong> 人已加入</span><b>{percentage}%</b></div>
          <Progress value={demand.joined} total={demand.target} />
          <p className="microcopy">{demand.merchantHint}</p>
        </div>
        <div className="drawer-section">
          <span className="field-label-title">已聚合的共同要求</span>
          <div className="tag-list drawer-tags">{demand.tags.map((tag) => <span className="tag" key={tag}><Check size={13} />{tag}</span>)}</div>
        </div>
        <div className="drawer-footer">
          <div>
            <span>加入后可收到</span>
            <strong>报价和开售提醒</strong>
          </div>
          <button className="button button-primary" onClick={onJoin}>
            {demand.joined >= demand.target ? '查看预售方案' : '加入这组需求'}
            <ArrowUpRight size={17} />
          </button>
        </div>
      </aside>
    </div>
  )
}

function JoinDialog({ demand, onClose, onConfirm, saving }: { demand: Demand; onClose: () => void; onConfirm: () => void; saving: boolean }) {
  return (
    <div className="modal-layer modal-centered">
      <button className="modal-scrim" aria-label="关闭加入需求" onClick={onClose} />
      <div className="confirm-dialog" role="dialog" aria-modal="true" aria-label="加入需求确认">
        <div className="dialog-icon"><HeartPulse size={22} /></div>
        <button className="icon-button dialog-close" aria-label="关闭" onClick={onClose}><X size={18} /></button>
        <div className="eyebrow eyebrow-teal">加入需求</div>
        <h2>{demand.title}</h2>
        <p>你将加入 <strong>{demand.location}</strong> 的共同需求。商家报价和预售时间确定后，我们会第一时间通知你。</p>
        <div className="dialog-summary">
          <span>当前进度</span>
          <strong>{demand.joined} / {demand.target} 人</strong>
        </div>
        <div className="dialog-actions">
          <button className="button button-quiet" onClick={onClose}>先看看</button>
          <button className="button button-primary" onClick={onConfirm} disabled={saving}>
            {saving ? '正在加入…' : '确认加入'}
            {!saving && <Check size={16} />}
          </button>
        </div>
      </div>
    </div>
  )
}

function OrderDialog({ product, onClose, onConfirm, saving }: { product: Product; onClose: () => void; onConfirm: (quantity: number) => void; saving: boolean }) {
  const [quantity, setQuantity] = useState(1)
  return (
    <div className="modal-layer modal-centered">
      <button className="modal-scrim" aria-label="关闭预订" onClick={onClose} />
      <div className="confirm-dialog order-dialog" role="dialog" aria-modal="true" aria-label="预订商品">
        <button className="icon-button dialog-close" aria-label="关闭" onClick={onClose}><X size={18} /></button>
        <div className={`dialog-product-mark ${product.accent}`}><Utensils size={25} /></div>
        <div className="eyebrow eyebrow-teal">确认预订</div>
        <h2>{product.title}</h2>
        <p>{product.merchant} · {product.delivery}</p>
        <div className="order-quantity">
          <span>购买份数</span>
          <div className="stepper">
            <button aria-label="减少数量" onClick={() => setQuantity(Math.max(1, quantity - 1))}>−</button>
            <strong>{quantity}</strong>
            <button aria-label="增加数量" onClick={() => setQuantity(Math.min(product.stock, quantity + 1))}>＋</button>
          </div>
        </div>
        <div className="dialog-summary">
          <span>商品小计</span>
          <strong>¥{(product.price * quantity).toFixed(2)}</strong>
        </div>
        <div className="dialog-actions">
          <button className="button button-quiet" onClick={onClose}>取消</button>
          <button className="button button-primary" onClick={() => onConfirm(quantity)} disabled={saving}>
            {saving ? '正在提交…' : '确认预订'}
            {!saving && <ShoppingBag size={16} />}
          </button>
        </div>
      </div>
    </div>
  )
}

export default function App() {
  const [activeView, setActiveView] = useState<ConsumerView>('overview')
  const [mobileOpen, setMobileOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [snapshot, setSnapshot] = useState<{ demands: Demand[]; products: Product[]; orders: ConsumerOrder[] }>({ demands: [], products: [], orders: [] })
  const [fallbackMessage, setFallbackMessage] = useState('')
  const [loadError, setLoadError] = useState('')
  const [selectedDemand, setSelectedDemand] = useState<Demand | null>(null)
  const [joinTarget, setJoinTarget] = useState<Demand | null>(null)
  const [orderTarget, setOrderTarget] = useState<Product | null>(null)
  const [actionLoading, setActionLoading] = useState(false)
  const [toast, setToast] = useState('')

  useEffect(() => {
    let active = true
    loadConsumerSnapshot()
      .then((result) => {
        if (!active) return
        setSnapshot(result.data)
        setFallbackMessage(result.isFallback ? result.message ?? '当前使用演示数据' : '')
      })
      .catch(() => active && setLoadError('数据加载失败，请刷新重试'))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(''), 3000)
    return () => window.clearTimeout(timer)
  }, [toast])

  const myDemandIds = useMemo(() => new Set(['demand-1', 'demand-2']), [])

  function navigate(view: ConsumerView) {
    setActiveView(view)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  async function handleJoin(demand: Demand) {
    setJoinTarget(demand)
    setSelectedDemand(null)
  }

  async function confirmJoin() {
    if (!joinTarget) return
    setActionLoading(true)
    const updated = await joinDemand(joinTarget)
    setSnapshot((current) => ({ ...current, demands: current.demands.map((demand) => (demand.id === updated.id ? updated : demand)) }))
    setJoinTarget(null)
    setActionLoading(false)
    setToast(`已加入「${updated.title}」，等人数够了就开做`)
  }

  async function handleOrder(product: Product) {
    setOrderTarget(product)
  }

  async function confirmOrder(quantity: number) {
    if (!orderTarget) return
    setActionLoading(true)
    const order = await placeOrder(orderTarget, quantity)
    setSnapshot((current) => ({ ...current, orders: [order, ...current.orders] }))
    setOrderTarget(null)
    setActionLoading(false)
    setToast('预订成功，订单已进入待成团')
    navigate('orders')
  }

  function handleCreated(demand: Demand) {
    setSnapshot((current) => ({ ...current, demands: [demand, ...current.demands] }))
    setToast('需求已发布，正在等待更多人加入')
    navigate('demands')
  }

  function renderView() {
    if (loading) return <LoadingState />
    if (loadError) return <ErrorState message={loadError} onRetry={() => window.location.reload()} />
    if (activeView === 'overview') {
      return <Overview demands={snapshot.demands} products={snapshot.products} onNavigate={navigate} onOpenDemand={setSelectedDemand} onJoinDemand={handleJoin} onOrder={handleOrder} />
    }
    if (activeView === 'demands') return <DemandsView demands={snapshot.demands} onOpen={setSelectedDemand} onJoin={handleJoin} onNavigate={navigate} />
    if (activeView === 'publish') return <PublishView onCreated={handleCreated} />
    if (activeView === 'presale') return <PresaleView products={snapshot.products} onOrder={handleOrder} />
    return <OrdersView orders={snapshot.orders} />
  }

  return (
    <div className="app-shell">
      <ConsumerSidebar activeView={activeView} onNavigate={navigate} mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />
      <div className="main-shell">
        <Topbar onMenu={() => setMobileOpen(true)} />
        <main className="main-content">
          {fallbackMessage && (
            <div className="fallback-notice">
              <AlertCircle size={16} />
              {fallbackMessage}，操作仍可在本地演示
              <button className="notice-close" aria-label="关闭提示" onClick={() => setFallbackMessage('')}><X size={15} /></button>
            </div>
          )}
          {renderView()}
        </main>
      </div>
      {selectedDemand && <DemandDetail demand={selectedDemand} onClose={() => setSelectedDemand(null)} onJoin={() => handleJoin(selectedDemand)} />}
      {joinTarget && <JoinDialog demand={joinTarget} onClose={() => setJoinTarget(null)} onConfirm={confirmJoin} saving={actionLoading} />}
      {orderTarget && <OrderDialog product={orderTarget} onClose={() => setOrderTarget(null)} onConfirm={confirmOrder} saving={actionLoading} />}
      {toast && <div className="toast"><Check size={16} />{toast}</div>}
    </div>
  )
}

function LoadingState() {
  return (
    <section className="loading-state">
      <div className="loading-orb" />
      <div className="skeleton-line skeleton-line-long" />
      <div className="skeleton-line skeleton-line-short" />
      <div className="skeleton-grid">
        <span /><span /><span />
      </div>
      <p>正在整理附近的需求…</p>
    </section>
  )
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <section className="error-state">
      <div className="error-icon"><AlertCircle size={25} /></div>
      <h2>暂时没能打开工作台</h2>
      <p>{message}</p>
      <button className="button button-primary" onClick={onRetry}>重新加载</button>
    </section>
  )
}
