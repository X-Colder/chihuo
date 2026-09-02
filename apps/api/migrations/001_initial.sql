CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('CONSUMER', 'MERCHANT', 'RIDER', 'ADMIN')),
  demo_key TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS merchants (
  id UUID PRIMARY KEY,
  owner_user_id UUID NOT NULL REFERENCES users(id),
  name TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'SUSPENDED')),
  license JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS demand_clusters (
  id UUID PRIMARY KEY,
  created_by UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  category TEXT NOT NULL,
  service_area TEXT NOT NULL,
  serving_date DATE NOT NULL,
  serving_time TEXT NOT NULL,
  budget_min_cents INTEGER NOT NULL,
  budget_max_cents INTEGER NOT NULL,
  quantity INTEGER NOT NULL,
  weight_min_grams INTEGER NOT NULL,
  weight_max_grams INTEGER NOT NULL,
  hard_constraints JSONB NOT NULL DEFAULT '[]'::jsonb,
  preferences JSONB NOT NULL DEFAULT '[]'::jsonb,
  notes TEXT,
  minimum_members INTEGER NOT NULL,
  maximum_members INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('PENDING_REVIEW', 'OPEN', 'READY', 'CLOSED', 'REJECTED')),
  reviewed_by UUID REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  review_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS demand_clusters_browse_idx
  ON demand_clusters (status, category, service_area, serving_date, serving_time);

CREATE TABLE IF NOT EXISTS demand_members (
  id UUID PRIMARY KEY,
  demand_id UUID NOT NULL REFERENCES demand_clusters(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id),
  quantity INTEGER NOT NULL,
  weight_grams INTEGER,
  preferences JSONB NOT NULL DEFAULT '[]'::jsonb,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (demand_id, user_id)
);

CREATE TABLE IF NOT EXISTS offers (
  id UUID PRIMARY KEY,
  demand_id UUID NOT NULL REFERENCES demand_clusters(id),
  merchant_id UUID NOT NULL REFERENCES merchants(id),
  unit_price_cents INTEGER NOT NULL,
  production_capacity INTEGER NOT NULL,
  weight_grams INTEGER NOT NULL,
  ingredients JSONB NOT NULL DEFAULT '[]'::jsonb,
  allergens JSONB NOT NULL DEFAULT '[]'::jsonb,
  oil_level TEXT NOT NULL,
  salt_level TEXT NOT NULL,
  production_time TEXT NOT NULL,
  shelf_life_minutes INTEGER NOT NULL,
  storage_instructions TEXT NOT NULL,
  notes TEXT,
  status TEXT NOT NULL CHECK (status IN ('SUBMITTED', 'ACCEPTED', 'REJECTED', 'WITHDRAWN')),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS offers_demand_idx ON offers (demand_id, status);

CREATE TABLE IF NOT EXISTS campaigns (
  id UUID PRIMARY KEY,
  demand_id UUID NOT NULL REFERENCES demand_clusters(id),
  offer_id UUID NOT NULL REFERENCES offers(id),
  merchant_id UUID NOT NULL REFERENCES merchants(id),
  title TEXT NOT NULL,
  description TEXT,
  unit_price_cents INTEGER NOT NULL,
  delivery_fee_cents INTEGER NOT NULL,
  platform_fee_bps INTEGER NOT NULL,
  minimum_orders INTEGER NOT NULL,
  maximum_orders INTEGER NOT NULL,
  starts_at TIMESTAMPTZ NOT NULL,
  ends_at TIMESTAMPTZ NOT NULL,
  pickup_point TEXT NOT NULL,
  food_spec JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('DRAFT', 'PENDING_REVIEW', 'OPEN', 'SOLD_OUT', 'CLOSED', 'CANCELLED')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS campaigns_browse_idx ON campaigns (status, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS orders (
  id UUID PRIMARY KEY,
  campaign_id UUID NOT NULL REFERENCES campaigns(id),
  consumer_id UUID NOT NULL REFERENCES users(id),
  quantity INTEGER NOT NULL,
  delivery_address TEXT NOT NULL,
  contact_name TEXT NOT NULL,
  contact_phone TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('PENDING_PAYMENT', 'PAID', 'ACCEPTED', 'PREPARING', 'READY_FOR_PICKUP', 'PICKED_UP', 'DELIVERING', 'DELIVERED', 'CANCELLED', 'REFUNDED')),
  unit_price_cents INTEGER NOT NULL,
  subtotal_cents INTEGER NOT NULL,
  delivery_fee_cents INTEGER NOT NULL,
  platform_fee_cents INTEGER NOT NULL,
  total_cents INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS orders_consumer_idx ON orders (consumer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS rider_tasks (
  id UUID PRIMARY KEY,
  order_id UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
  rider_id UUID REFERENCES users(id),
  status TEXT NOT NULL CHECK (status IN ('UNASSIGNED', 'ASSIGNED', 'PICKED_UP', 'DELIVERING', 'COMPLETED', 'CANCELLED')),
  pickup_point TEXT NOT NULL,
  delivery_address TEXT NOT NULL,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS rider_tasks_queue_idx ON rider_tasks (status, rider_id);

CREATE TABLE IF NOT EXISTS food_safety_incidents (
  id UUID PRIMARY KEY,
  merchant_id UUID NOT NULL REFERENCES merchants(id),
  order_id UUID REFERENCES orders(id),
  campaign_id UUID REFERENCES campaigns(id),
  reported_by UUID NOT NULL REFERENCES users(id),
  severity TEXT NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  evidence_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
  affected_quantity INTEGER,
  status TEXT NOT NULL CHECK (status IN ('OPEN', 'INVESTIGATING', 'RESOLVED', 'REPORTED')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS food_safety_incidents_merchant_idx
  ON food_safety_incidents (merchant_id, status, created_at DESC);
