import { useEffect, useMemo, useState } from 'react'
import {
  AlertTriangle,
  ArrowLeft,
  Bike,
  Check,
  ChevronRight,
  CircleDollarSign,
  Clock3,
  Copy,
  FileText,
  Home,
  MapPin,
  Menu,
  MessageCircle,
  Navigation,
  PackageCheck,
  Phone,
  Plus,
  Route,
  ShieldCheck,
  UserRound,
  X,
} from 'lucide-react'
import { fetchRiderTasks, reportRiderIssue, updateRiderTaskStatus } from './api'
import { statusMeta, type RiderTask, type TaskStatus } from './data'

type View = 'tasks' | 'earnings' | 'profile'
type Filter = '全部' | TaskStatus

const filterItems: Filter[] = ['全部', '待取餐', '待配送', '配送中', '已完成', '异常待处理']

function money(value: number) {
  return `¥${value.toFixed(2)}`
}

function App() {
  const [view, setView] = useState<View>('tasks')
  const [tasks, setTasks] = useState<RiderTask[]>([])
  const [selectedTask, setSelectedTask] = useState<RiderTask | null>(null)
  const [filter, setFilter] = useState<Filter>('全部')
  const [loading, setLoading] = useState(true)
  const [issueTask, setIssueTask] = useState<RiderTask | null>(null)
  const [toast, setToast] = useState<string | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)

  useEffect(() => {
    fetchRiderTasks().then((data) => {
      setTasks(data)
      setLoading(false)
    })
  }, [])

  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(null), 2600)
    return () => window.clearTimeout(timer)
  }, [toast])

  const visibleTasks = useMemo(
    () => (filter === '全部' ? tasks : tasks.filter((task) => task.status === filter)),
    [filter, tasks],
  )

  const activeCount = tasks.filter((task) =>
    ['待取餐', '待配送', '配送中'].includes(task.status),
  ).length
  const todayIncome = tasks
    .filter((task) => task.status === '已完成')
    .reduce((sum, task) => sum + task.fee, 0)

  function selectTask(task: RiderTask) {
    setSelectedTask(task)
    setMenuOpen(false)
  }

  async function advanceTask(task: RiderTask) {
    const nextStatus: Partial<Record<TaskStatus, TaskStatus>> = {
      待取餐: '待配送',
      待配送: '配送中',
      配送中: '已完成',
    }
    const next = nextStatus[task.status]
    if (!next) return
    const updated = await updateRiderTaskStatus(task, next)
    setTasks((current) => current.map((item) => (item.id === updated.id ? updated : item)))
    setSelectedTask(updated)
    setToast(next === '已完成' ? '订单已完成，配送费已计入今日收入' : `订单状态已更新为「${next}」`)
  }

  async function submitIssue(category: string, description: string) {
    if (!issueTask) return
    await reportRiderIssue(issueTask, { category, description })
    const updated = { ...issueTask, status: '异常待处理' as const, updatedAt: '刚刚' }
    setTasks((current) => current.map((item) => (item.id === updated.id ? updated : item)))
    setSelectedTask(updated)
    setIssueTask(null)
    setToast('异常已上报，平台客服会尽快联系你')
  }

  return (
    <div className="rider-app">
      <aside className={`rider-sidebar ${menuOpen ? 'is-open' : ''}`}>
        <div className="brand-lockup">
          <div className="brand-mark">
            <Bike size={22} strokeWidth={2.5} />
          </div>
          <div>
            <strong>吃货</strong>
            <span>骑手工作台</span>
          </div>
          <button
            className="icon-button sidebar-close"
            onClick={() => setMenuOpen(false)}
            aria-label="关闭菜单"
            title="关闭菜单"
          >
            <X size={18} />
          </button>
        </div>

        <div className="rider-profile">
          <div className="avatar">林</div>
          <div>
            <strong>林师傅</strong>
            <span><span className="status-dot" /> 在线接单</span>
          </div>
          <ChevronRight size={17} className="muted-icon" />
        </div>

        <nav className="side-nav" aria-label="主导航">
          <NavButton icon={<Home size={18} />} label="配送任务" active={view === 'tasks'} onClick={() => { setView('tasks'); setSelectedTask(null); setMenuOpen(false) }} badge={activeCount} />
          <NavButton icon={<CircleDollarSign size={18} />} label="收入明细" active={view === 'earnings'} onClick={() => { setView('earnings'); setSelectedTask(null); setMenuOpen(false) }} />
          <NavButton icon={<UserRound size={18} />} label="我的资料" active={view === 'profile'} onClick={() => { setView('profile'); setSelectedTask(null); setMenuOpen(false) }} />
        </nav>

        <div className="sidebar-footer">
          <div className="support-card">
            <div className="support-icon"><ShieldCheck size={17} /></div>
            <div>
              <strong>配送保障</strong>
              <span>遇到问题随时上报</span>
            </div>
          </div>
          <span className="version-label">吃货骑手端 · v0.1</span>
        </div>
      </aside>

      {menuOpen && <button className="sidebar-scrim" aria-label="关闭菜单" onClick={() => setMenuOpen(false)} />}

      <main className="rider-main">
        <header className="topbar">
          <div className="topbar-left">
            <button
              className="icon-button menu-trigger"
              onClick={() => setMenuOpen(true)}
              aria-label="打开菜单"
              title="打开菜单"
            >
              <Menu size={21} />
            </button>
            <div>
              <span className="eyebrow">星期三 · 9月2日</span>
              <h1>{view === 'tasks' ? '今天也辛苦了，林师傅' : view === 'earnings' ? '收入明细' : '个人资料'}</h1>
            </div>
          </div>
          <div className="topbar-actions">
            <button className="help-button" title="联系客服">
              <MessageCircle size={17} />
              <span>联系客服</span>
            </button>
            <div className="topbar-avatar">林</div>
          </div>
        </header>

        {view === 'tasks' && (
          <TaskWorkspace
            tasks={tasks}
            visibleTasks={visibleTasks}
            filter={filter}
            setFilter={setFilter}
            loading={loading}
            selectedTask={selectedTask}
            selectTask={selectTask}
            advanceTask={advanceTask}
            setIssueTask={setIssueTask}
            activeCount={activeCount}
            todayIncome={todayIncome}
          />
        )}
        {view === 'earnings' && <EarningsView tasks={tasks} />}
        {view === 'profile' && <ProfileView />}
      </main>

      <div className="mobile-nav">
        <NavButton icon={<Home size={19} />} label="任务" active={view === 'tasks'} onClick={() => { setView('tasks'); setSelectedTask(null) }} />
        <NavButton icon={<CircleDollarSign size={19} />} label="收入" active={view === 'earnings'} onClick={() => { setView('earnings'); setSelectedTask(null) }} />
        <NavButton icon={<UserRound size={19} />} label="我的" active={view === 'profile'} onClick={() => { setView('profile'); setSelectedTask(null) }} />
      </div>

      {issueTask && <IssueModal task={issueTask} onClose={() => setIssueTask(null)} onSubmit={submitIssue} />}
      {toast && <div className="toast"><Check size={16} />{toast}</div>}
    </div>
  )
}

function NavButton({
  icon,
  label,
  active,
  onClick,
  badge,
}: {
  icon: React.ReactNode
  label: string
  active: boolean
  onClick: () => void
  badge?: number
}) {
  return (
    <button className={`nav-button ${active ? 'active' : ''}`} onClick={onClick}>
      {icon}
      <span>{label}</span>
      {badge ? <em>{badge}</em> : null}
    </button>
  )
}

function TaskWorkspace({
  tasks,
  visibleTasks,
  filter,
  setFilter,
  loading,
  selectedTask,
  selectTask,
  advanceTask,
  setIssueTask,
  activeCount,
  todayIncome,
}: {
  tasks: RiderTask[]
  visibleTasks: RiderTask[]
  filter: Filter
  setFilter: (filter: Filter) => void
  loading: boolean
  selectedTask: RiderTask | null
  selectTask: (task: RiderTask) => void
  advanceTask: (task: RiderTask) => void
  setIssueTask: (task: RiderTask) => void
  activeCount: number
  todayIncome: number
}) {
  return (
    <div className="workspace">
      <section className="welcome-strip">
        <div>
          <span className="section-kicker">今日概览</span>
          <h2>把每一份餐，准时送到。</h2>
          <p>当前有 <strong>{activeCount} 个进行中的配送任务</strong>，预计收入 {money(todayIncome + 18)}。</p>
        </div>
        <div className="welcome-route">
          <Route size={20} />
          <span>南山 · 科技园片区</span>
        </div>
      </section>

      <div className="metric-grid">
        <MetricCard icon={<PackageCheck size={19} />} label="待处理任务" value={String(activeCount)} note="尽快完成取餐" tone="orange" />
        <MetricCard icon={<CircleDollarSign size={19} />} label="今日已赚" value={money(todayIncome)} note="已完成订单" tone="teal" />
        <MetricCard icon={<Clock3 size={19} />} label="平均配送时长" value="28 min" note="比昨日快 4 分钟" tone="blue" />
      </div>

      <section className="task-section">
        <div className="section-heading">
          <div>
            <span className="section-kicker">任务中心</span>
            <h2>配送任务</h2>
          </div>
          <div className="task-filters" role="tablist" aria-label="任务筛选">
            {filterItems.map((item) => (
              <button
                key={item}
                className={filter === item ? 'filter-chip active' : 'filter-chip'}
                onClick={() => setFilter(item)}
                role="tab"
                aria-selected={filter === item}
              >
                {item === '全部' ? `全部 ${tasks.length}` : `${item} ${tasks.filter((task) => task.status === item).length}`}
              </button>
            ))}
          </div>
        </div>

        <div className={`task-layout ${selectedTask ? 'with-detail' : ''}`}>
          <div className="task-list">
            {loading ? (
              <div className="empty-state"><div className="spinner" />正在加载任务</div>
            ) : visibleTasks.length === 0 ? (
              <div className="empty-state">
                <PackageCheck size={32} />
                <strong>这个筛选下暂无任务</strong>
                <span>新的配送任务会出现在这里</span>
              </div>
            ) : (
              visibleTasks.map((task) => (
                <TaskCard key={task.id} task={task} selected={selectedTask?.id === task.id} onClick={() => selectTask(task)} />
              ))
            )}
          </div>
          {selectedTask ? (
            <TaskDetail task={selectedTask} onAdvance={() => advanceTask(selectedTask)} onReport={() => setIssueTask(selectedTask)} />
          ) : (
            <div className="detail-placeholder">
              <div className="placeholder-icon"><Navigation size={24} /></div>
              <strong>选择一个任务查看详情</strong>
              <span>订单地址、餐品信息和配送操作会显示在这里</span>
            </div>
          )}
        </div>
      </section>
    </div>
  )
}

function MetricCard({
  icon,
  label,
  value,
  note,
  tone,
}: {
  icon: React.ReactNode
  label: string
  value: string
  note: string
  tone: string
}) {
  return (
    <div className={`metric-card ${tone}`}>
      <div className="metric-icon">{icon}</div>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{note}</small>
    </div>
  )
}

function TaskCard({ task, selected, onClick }: { task: RiderTask; selected: boolean; onClick: () => void }) {
  const meta = statusMeta[task.status]
  return (
    <button className={`task-card ${selected ? 'selected' : ''}`} onClick={onClick}>
      <div className="task-card-top">
        <span className={`status-pill ${meta.tone}`}><span />{meta.label}</span>
        <strong>{money(task.fee)}</strong>
      </div>
      <div className="task-card-title">
        <strong>{task.storeName}</strong>
        <span>{task.orderNo.slice(-4)}</span>
      </div>
      <div className="task-route">
        <span className="route-dot pickup" />
        <div><small>取餐</small><p>{task.pickupAddress}</p></div>
        <span className="route-line" />
        <span className="route-dot dropoff" />
        <div><small>送达</small><p>{task.dropoffAddress}</p></div>
      </div>
      <div className="task-card-bottom">
        <span><Clock3 size={14} />{task.pickupWindow}</span>
        <span><Navigation size={14} />{task.distance}</span>
        <ChevronRight size={17} className="task-arrow" />
      </div>
    </button>
  )
}

function TaskDetail({
  task,
  onAdvance,
  onReport,
}: {
  task: RiderTask
  onAdvance: () => void
  onReport: () => void
}) {
  const meta = statusMeta[task.status]
  const actionLabel: Partial<Record<TaskStatus, string>> = {
    待取餐: '确认已取餐',
    待配送: '开始配送',
    配送中: '确认送达',
  }
  return (
    <article className="task-detail">
      <div className="detail-heading">
        <div>
          <button className="back-inline"><ArrowLeft size={15} />任务详情</button>
          <h3>{task.storeName}</h3>
          <span className="detail-order">订单号 {task.orderNo}</span>
        </div>
        <span className={`status-pill ${meta.tone}`}><span />{meta.label}</span>
      </div>

      <div className="timeline">
        <TimelineStep icon={<MapPin size={16} />} label="取餐地址" value={task.pickupAddress} done={task.status !== '待取餐'} />
        <TimelineStep icon={<Navigation size={16} />} label="送达地址" value={task.dropoffAddress} done={task.status === '已完成'} />
      </div>

      <div className="detail-info-grid">
        <div><span>取餐时间</span><strong>{task.pickupWindow}</strong></div>
        <div><span>配送距离</span><strong>{task.distance}</strong></div>
        <div><span>配送费</span><strong className="highlight-value">{money(task.fee)}</strong></div>
        <div><span>顾客</span><strong>{task.customer}</strong></div>
      </div>

      <div className="detail-block">
        <div className="block-title"><PackageCheck size={17} />餐品信息</div>
        <p>{task.itemSummary}</p>
        <div className="note-box"><FileText size={15} /><span>{task.notes}</span></div>
      </div>

      <div className="detail-actions">
        {actionLabel[task.status] ? (
          <button className="primary-action" onClick={onAdvance}>
            {task.status === '配送中' ? <Check size={18} /> : <PackageCheck size={18} />}
            {actionLabel[task.status]}
          </button>
        ) : task.status === '已完成' ? (
          <div className="completed-banner"><Check size={18} />该订单已完成</div>
        ) : (
          <div className="completed-banner warning"><AlertTriangle size={18} />平台正在处理异常</div>
        )}
        <div className="secondary-actions">
          <a className="soft-action" href={`tel:${task.phone.replaceAll('*', '0')}`}><Phone size={16} />联系顾客</a>
          <button className="soft-action" onClick={onReport}><AlertTriangle size={16} />上报异常</button>
        </div>
      </div>
    </article>
  )
}

function TimelineStep({ icon, label, value, done }: { icon: React.ReactNode; label: string; value: string; done: boolean }) {
  return (
    <div className={`timeline-step ${done ? 'done' : ''}`}>
      <div className="timeline-icon">{done ? <Check size={15} /> : icon}</div>
      <div><span>{label}</span><strong>{value}</strong></div>
    </div>
  )
}

function EarningsView({ tasks }: { tasks: RiderTask[] }) {
  const completed = tasks.filter((task) => task.status === '已完成')
  const total = completed.reduce((sum, task) => sum + task.fee, 0)
  return (
    <div className="workspace simple-view">
      <section className="page-intro">
        <span className="section-kicker">账单概览</span>
        <h2>收入明细</h2>
        <p>今日订单完成情况和配送费收入。</p>
      </section>
      <div className="earnings-hero">
        <div><span>今日预计到账</span><strong>{money(total)}</strong><small>共完成 {completed.length} 单</small></div>
        <div className="earnings-progress"><span style={{ width: '72%' }} /><small>本月目标 ¥1,800 · 已完成 72%</small></div>
      </div>
      <section className="list-panel">
        <div className="panel-heading"><h3>今日订单</h3><button className="icon-button" title="导出账单"><Copy size={16} /></button></div>
        {tasks.map((task) => (
          <div className="earning-row" key={task.id}>
            <div className={`earning-symbol ${task.status === '已完成' ? 'is-done' : ''}`}><Bike size={17} /></div>
            <div><strong>{task.storeName}</strong><span>{task.orderNo} · {task.updatedAt}</span></div>
            <b className={task.status === '已完成' ? 'income' : 'pending-income'}>{task.status === '已完成' ? `+${money(task.fee)}` : '进行中'}</b>
          </div>
        ))}
      </section>
    </div>
  )
}

function ProfileView() {
  return (
    <div className="workspace simple-view">
      <section className="page-intro">
        <span className="section-kicker">账户设置</span>
        <h2>我的资料</h2>
        <p>查看骑手认证和服务区域信息。</p>
      </section>
      <section className="profile-panel">
        <div className="profile-hero"><div className="large-avatar">林</div><div><h3>林师傅</h3><span className="verified"><ShieldCheck size={15} />已完成实名认证</span></div><button className="soft-action">编辑资料</button></div>
        <div className="profile-fields">
          <ProfileField label="联系电话" value="138 0000 1268" />
          <ProfileField label="配送区域" value="深圳市 · 南山区 · 科技园片区" />
          <ProfileField label="服务时段" value="10:00 - 14:00 / 17:00 - 20:00" />
          <ProfileField label="骑手编号" value="RIDER-0836" />
        </div>
      </section>
      <section className="profile-panel compact">
        <div className="panel-heading"><h3>服务记录</h3><ChevronRight size={18} /></div>
        <div className="record-grid"><div><strong>4.96</strong><span>服务评分</span></div><div><strong>98%</strong><span>准时率</span></div><div><strong>326</strong><span>累计完成</span></div></div>
      </section>
    </div>
  )
}

function ProfileField({ label, value }: { label: string; value: string }) {
  return <div><span>{label}</span><strong>{value}</strong></div>
}

function IssueModal({
  task,
  onClose,
  onSubmit,
}: {
  task: RiderTask
  onClose: () => void
  onSubmit: (category: string, description: string) => void
}) {
  const [category, setCategory] = useState('商家出餐延迟')
  const [description, setDescription] = useState('')
  const categories = ['商家出餐延迟', '顾客无法联系', '地址无法送达', '餐品或包装破损', '其他异常']
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <section className="issue-modal" onClick={(event) => event.stopPropagation()}>
        <div className="modal-heading"><div><span className="section-kicker">订单 {task.orderNo.slice(-4)}</span><h3>上报配送异常</h3></div><button className="icon-button" onClick={onClose} aria-label="关闭" title="关闭"><X size={18} /></button></div>
        <label className="form-label">异常类型</label>
        <div className="issue-options">{categories.map((item) => <button key={item} className={category === item ? 'issue-option active' : 'issue-option'} onClick={() => setCategory(item)}>{category === item && <Check size={14} />}{item}</button>)}</div>
        <label className="form-label" htmlFor="issue-description">补充说明 <span>可选</span></label>
        <textarea id="issue-description" value={description} onChange={(event) => setDescription(event.target.value)} placeholder="请描述现场情况，便于平台快速处理" rows={4} />
        <button className="primary-action full" onClick={() => onSubmit(category, description)}>提交异常</button>
      </section>
    </div>
  )
}

export default App
