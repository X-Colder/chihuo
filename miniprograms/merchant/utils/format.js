function money(cents) {
  return `¥${(Number(cents || 0) / 100).toFixed(2)}`
}

function list(value) {
  return Array.isArray(value) ? value : []
}

function value(item, camelName, snakeName, fallback) {
  if (item[camelName] !== undefined && item[camelName] !== null) return item[camelName]
  if (item[snakeName] !== undefined && item[snakeName] !== null) return item[snakeName]
  return fallback
}

function dateText(valueToFormat) {
  return valueToFormat ? String(valueToFormat).replace('T', ' ').slice(0, 16) : ''
}

function demand(item, quotedIds) {
  const statusMap = {
    READY: '可报价',
    CLOSED: '已结束',
    REJECTED: '已驳回',
    PENDING_REVIEW: '平台审核中'
  }
  const quoted = quotedIds && quotedIds[item.id]
  return {
    raw: item,
    id: item.id,
    title: item.title || '未命名需求',
    category: item.category || '餐饮',
    serviceArea: value(item, 'serviceArea', 'service_area', '区域待确认'),
    schedule: `${value(item, 'servingDate', 'serving_date', '日期待定')} ${value(item, 'servingTime', 'serving_time', '')}`,
    budget: `${money(value(item, 'budgetMinCents', 'budget_min_cents', 0))} - ${money(value(item, 'budgetMaxCents', 'budget_max_cents', 0))}`,
    joined: Number(value(item, 'memberCount', 'member_count', 0)),
    target: Number(value(item, 'minimumMembers', 'minimum_members', 0)),
    status: quoted ? '已报价' : statusMap[item.status] || '聚合中',
    statusClass: item.status === 'READY' ? 'success' : item.status === 'REJECTED' ? 'muted-status' : '',
    tags: list(value(item, 'hardConstraints', 'hard_constraints', [])).concat(list(item.preferences)),
    notes: item.notes || '平台将相近需求匿名聚合，供商家评估生产可行性。',
    quoted
  }
}

function offer(item, title) {
  const statusMap = {
    SUBMITTED: ['审核中', ''],
    ACCEPTED: ['已入选', 'success'],
    REJECTED: ['未入选', 'muted-status'],
    WITHDRAWN: ['已撤回', 'muted-status']
  }
  const status = statusMap[item.status] || ['处理中', '']
  return {
    raw: item,
    id: item.id,
    demandId: item.demandId,
    demandTitle: title || '需求报价',
    price: money(value(item, 'unitPriceCents', 'unit_price_cents', 0)),
    capacity: Number(value(item, 'productionCapacity', 'production_capacity', 0)),
    weight: `${value(item, 'weightGrams', 'weight_grams', '-')}g`,
    oilLevel: (item.oilLevel || item.oil_level) === 'LOW' ? '低油' : (item.oilLevel || item.oil_level) === 'MEDIUM' ? '适中油' : '油量待确认',
    saltLevel: (item.saltLevel || item.salt_level) === 'LOW' ? '少盐' : (item.saltLevel || item.salt_level) === 'MEDIUM' ? '适中盐' : '盐量待确认',
    shelfLife: value(item, 'shelfLifeMinutes', 'shelf_life_minutes', 0) ? `${value(item, 'shelfLifeMinutes', 'shelf_life_minutes', 0)} 分钟` : '待确认',
    storageInstructions: value(item, 'storageInstructions', 'storage_instructions', '待补充'),
    status: status[0],
    statusClass: status[1],
    createdAt: dateText(value(item, 'createdAt', 'created_at', ''))
  }
}

function campaign(item) {
  const spec = value(item, 'foodSpec', 'food_spec', {}) || {}
  const current = Number(value(item, 'currentOrders', 'current_orders', 0))
  const maximum = Number(value(item, 'maximumOrders', 'maximum_orders', 0))
  const statusMap = {
    DRAFT: ['草稿', 'muted-status'],
    PENDING_REVIEW: ['平台审核中', ''],
    OPEN: ['进行中', 'success'],
    SOLD_OUT: ['已售罄', ''],
    CLOSED: ['已结束', 'muted-status'],
    CANCELLED: ['已取消', 'muted-status']
  }
  const status = statusMap[item.status] || ['处理中', '']
  return {
    raw: item,
    id: item.id,
    title: item.title || '预售活动',
    status: status[0],
    statusClass: status[1],
    current,
    maximum,
    stock: Math.max(0, maximum - current),
    unitPrice: money(value(item, 'unitPriceCents', 'unit_price_cents', 0)),
    deliveryFee: money(value(item, 'deliveryFeeCents', 'delivery_fee_cents', 0)),
    pickupPoint: value(item, 'pickupPoint', 'pickup_point', '待设置'),
    startsAt: dateText(value(item, 'startsAt', 'starts_at', '')) || '待设置',
    endsAt: dateText(value(item, 'endsAt', 'ends_at', '')) || '待设置',
    spec: `${value(spec, 'weightGrams', 'weight_grams', '-')}g · ${spec.oilLevel === 'LOW' || spec.oil_level === 'LOW' ? '低油' : '规格待确认'} · ${spec.saltLevel === 'LOW' || spec.salt_level === 'LOW' ? '少盐' : '盐量待确认'}`
  }
}

function incident(item) {
  const statusMap = {
    OPEN: ['待处理', ''],
    INVESTIGATING: ['调查中', ''],
    RESOLVED: ['已完成', 'success'],
    REPORTED: ['已上报', 'muted-status']
  }
  const status = statusMap[item.status] || ['处理中', '']
  return {
    raw: item,
    id: item.id,
    title: item.title || '食品安全记录',
    description: item.description || '',
    severity: item.severity || 'LOW',
    severityText: item.severity === 'CRITICAL' ? '紧急' : item.severity === 'HIGH' ? '高' : item.severity === 'MEDIUM' ? '中' : '低',
    status: status[0],
    statusClass: status[1],
    createdAt: dateText(value(item, 'createdAt', 'created_at', ''))
  }
}

module.exports = { list, money, demand, offer, campaign, incident }
