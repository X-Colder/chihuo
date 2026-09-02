CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE REFERENCES orders(id),
    provider TEXT NOT NULL,
    out_trade_no TEXT NOT NULL UNIQUE,
    transaction_id TEXT,
    prepay_id TEXT,
    status TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    created_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS payments_status_idx ON payments (status, updated_at DESC);

CREATE TABLE IF NOT EXISTS refunds (
    id UUID PRIMARY KEY,
    payment_id UUID NOT NULL REFERENCES payments(id),
    out_refund_no TEXT NOT NULL UNIQUE,
    refund_id TEXT,
    status TEXT NOT NULL,
    total_amount_cents BIGINT NOT NULL,
    refund_amount_cents BIGINT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    succeeded_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS refunds_payment_idx ON refunds (payment_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS payment_notifications (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    out_trade_no TEXT NOT NULL,
    out_refund_no TEXT,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS merchant_qualifications (
    id UUID PRIMARY KEY,
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    legal_entity_name TEXT NOT NULL,
    store_name TEXT NOT NULL,
    business_license_number TEXT NOT NULL,
    food_permit_number TEXT NOT NULL,
    registered_address TEXT NOT NULL,
    operating_address TEXT NOT NULL,
    business_license_issued_at TIMESTAMPTZ NOT NULL,
    business_license_expires_at TIMESTAMPTZ NOT NULL,
    food_permit_issued_at TIMESTAMPTZ NOT NULL,
    food_permit_expires_at TIMESTAMPTZ NOT NULL,
    business_scope JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL,
    submitted_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS merchant_qualifications_status_idx
    ON merchant_qualifications (merchant_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS qualification_documents (
    id UUID PRIMARY KEY,
    qualification_id UUID NOT NULL REFERENCES merchant_qualifications(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    object_uri TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS site_inspections (
    id UUID PRIMARY KEY,
    qualification_id UUID NOT NULL REFERENCES merchant_qualifications(id) ON DELETE CASCADE,
    inspector_id UUID NOT NULL REFERENCES users(id),
    result TEXT NOT NULL,
    notes TEXT,
    evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    inspected_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS food_batches (
    id UUID PRIMARY KEY,
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    campaign_id UUID REFERENCES campaigns(id),
    product_id UUID,
    production_date TIMESTAMPTZ NOT NULL,
    produced_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    shelf_life_minutes INTEGER NOT NULL,
    storage_condition TEXT NOT NULL,
    quantity_planned INTEGER NOT NULL,
    quantity_produced INTEGER NOT NULL DEFAULT 0,
    quantity_remaining INTEGER NOT NULL DEFAULT 0,
    unit_weight_grams INTEGER NOT NULL,
    specification JSONB NOT NULL,
    ingredient_lots JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS food_batches_lookup_idx
    ON food_batches (merchant_id, status, production_date DESC);

CREATE TABLE IF NOT EXISTS batch_order_associations (
    batch_id UUID NOT NULL REFERENCES food_batches(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id),
    quantity INTEGER NOT NULL,
    linked_by UUID NOT NULL REFERENCES users(id),
    linked_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (batch_id, order_id)
);

CREATE TABLE IF NOT EXISTS safety_incidents (
    id UUID PRIMARY KEY,
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    reported_by UUID NOT NULL REFERENCES users(id),
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    batch_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    order_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL,
    containment_action TEXT,
    investigation_summary TEXT,
    resolution_summary TEXT,
    regulatory_report JSONB NOT NULL DEFAULT '{}'::jsonb,
    recall_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    reported_at TIMESTAMPTZ NOT NULL,
    contained_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS safety_incidents_status_idx
    ON safety_incidents (merchant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS safety_evidences (
    id UUID PRIMARY KEY,
    subject_type TEXT NOT NULL,
    subject_id UUID NOT NULL,
    evidence_type TEXT NOT NULL,
    object_uri TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    sequence_no BIGINT NOT NULL,
    previous_hash TEXT,
    current_hash TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    collected_by UUID NOT NULL REFERENCES users(id),
    collected_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS safety_evidences_subject_sequence_idx
    ON safety_evidences (subject_type, subject_id, sequence_no);

CREATE TABLE IF NOT EXISTS recalls (
    id UUID PRIMARY KEY,
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    scope TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT NOT NULL,
    batch_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    order_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    quantity_recalled INTEGER NOT NULL DEFAULT 0,
    quantity_recovered INTEGER NOT NULL DEFAULT 0,
    quantity_destroyed INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS recalls_status_idx ON recalls (merchant_id, status, created_at DESC);
