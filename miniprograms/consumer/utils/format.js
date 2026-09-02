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

function demand(item) {
  const statusMap = {
    READY: '已达标',
    CLOSED: '已结束',
    REJECTED: '已驳回',
    PENDING_REVIEW: '审核中'
  }
  const joined = Number(value(item, 'memberCount', 'member_count', 0))
  const target = Number(value(item, 'minimumMembers', 'minimum_members', 0))
  return {
    raw: item,
    id: item.id,
    title: item.title || '未命名需求',
    category: item.category || '餐饮',
    serviceArea: value(item, 'serviceArea', 'service_area', '待确认区域'),
    schedule: `${value(item, 'servingDate', 'serving_date', '日期待定')} ${value(item, 'servingTime', 'serving_time', '')}`,
    budget: `${money(value(item, 'budgetMinCents', 'budget_min_cents', 0))} - ${money(value(item, 'budgetMaxCents', 'budget_max_cents', 0))}`,
    joined,
    target,
    progress: target ? Math.min(100, Math.round(joined / target * 100)) : 0,
    status: statusMap[item.status] || '聚合中',
    statusClass: item.status === 'READY' ? 'success' : item.status === 'REJECTED' ? 'muted-status' : '',
    tags: list(value(item, 'hardConstraints', 'hard_constraints', [])).concat(list(item.preferences)),
    notes: item.notes || '平台会把相近需求聚合后提供给附近商家评估。'
  }
}

function campaign(item) {
  const spec = value(item, 'foodSpec', 'food_spec', {}) || {}
  const current = Number(value(item, 'currentOrders', 'current_orders', 0))
  const maximum = Number(value(item, 'maximumOrders', 'maximum_orders', 0))
  return {
    raw: item,
    id: item.id,
    title: item.title || '预售餐品',
    description: item.description || '按聚合需求批量制作，规格以商品详情为准。',
    price: money(value(item, 'unitPriceCents', 'unit_price_cents', 0)),
    unitPriceCents: Number(value(item, 'unitPriceCents', 'unit_price_cents', 0)),
    deliveryFee: money(value(item, 'deliveryFeeCents', 'delivery_fee_cents', 0)),
    stock: Math.max(0, maximum - current),
    soldText: `${current}/${maximum || '不限'} 份`,
    pickupPoint: value(item, 'pickupPoint', 'pickup_point', '集中配送点待确认'),
    startsAt: dateText(value(item, 'startsAt', 'starts_at', '')) || '时间待确认',
    endsAt: dateText(value(item, 'endsAt', 'ends_at', '')),
    weight: `${value(spec, 'weightGrams', 'weight_grams', '-') }g`,
    oilLevel: (spec.oilLevel || spec.oil_level) === 'LOW' ? '低油' : (spec.oilLevel || spec.oil_level) === 'MEDIUM' ? '适中油' : '油量待确认',
    saltLevel: (spec.saltLevel || spec.salt_level) === 'LOW' ? '少盐' : (spec.saltLevel || spec.salt_level) === 'MEDIUM' ? '适中盐' : '盐量待确认',
    ingredients: list(spec.ingredients),
    allergens: list(spec.allergens),
    allergensText: list(spec.allergens).join('、'),
    storageInstructions: spec.storageInstructions || '请按商品页面要求保存。',
    shelfLife: spec.shelfLifeMinutes ? `${spec.shelfLifeMinutes} 分钟内食用` : '食用时间待确认'
  }
}

function order(item) {
  const statusMap = {
    PENDING_PAYMENT: ['待支付', 'muted-status'],
    PAID: ['待成团', ''],
    ACCEPTED: ['商家已接单', ''],
    PREPARING: ['制作中', ''],
    READY_FOR_PICKUP: ['待取餐', ''],
    PICKED_UP: ['配送中', 'success'],
    DELIVERING: ['配送中', 'success'],
    DELIVERED: ['已完成', 'success'],
    CANCELLED: ['已取消', 'muted-status'],
    REFUNDED: ['已退款', 'muted-status']
  }
  const status = statusMap[item.status] || ['处理中', '']
  return {
    raw: item,
    id: item.id,
    status: status[0],
    statusClass: status[1],
    quantity: Number(item.quantity || 0),
    total: money(value(item, 'totalCents', 'total_cents', 0)),
    subtotal: money(value(item, 'subtotalCents', 'subtotal_cents', 0)),
    deliveryFee: money(value(item, 'deliveryFeeCents', 'delivery_fee_cents', 0)),
    platformFee: money(value(item, 'platformFeeCents', 'platform_fee_cents', 0)),
    address: value(item, 'deliveryAddress', 'delivery_address', '未填写地址'),
    contact: `${value(item, 'contactName', 'contact_name', '')} ${value(item, 'contactPhone', 'contact_phone', '')}`.trim(),
    createdAt: dateText(value(item, 'createdAt', 'created_at', ''))
  }
}

module.exports = { list, money, demand, campaign, order }
