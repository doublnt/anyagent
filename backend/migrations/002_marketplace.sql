-- AnyAgent Marketplace Migration
-- Activates dormant agents/agent_versions/project_agents tables
-- and adds entitlements + usage_events for subscription metering.

-- Entitlements: per-buyer subscription to a hosted agent
CREATE TABLE IF NOT EXISTS entitlements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id    UUID NOT NULL REFERENCES users(id),
    agent_id    UUID NOT NULL REFERENCES agents(id),
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',  -- active | suspended | expired
    period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_end   TIMESTAMPTZ NOT NULL,
    quota_calls  INT          NOT NULL DEFAULT 100,
    quota_tokens INT          NOT NULL DEFAULT 500000,  -- 500k tokens/month base
    used_calls   INT          NOT NULL DEFAULT 0,
    used_tokens  INT          NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- one active subscription per buyer per agent
    UNIQUE(buyer_id, agent_id)
);

-- Usage events: one row per hosted-agent call (cost + audit)
CREATE TABLE IF NOT EXISTS usage_events (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entitlement_id   UUID NOT NULL REFERENCES entitlements(id),
    agent_id         UUID NOT NULL REFERENCES agents(id),
    buyer_id         UUID NOT NULL REFERENCES users(id),
    trace_id         UUID REFERENCES traces(id),
    call_sequence    INT  NOT NULL DEFAULT 0,           -- ordinal within entitlement period
    input_tokens     INT  NOT NULL DEFAULT 0,
    output_tokens    INT  NOT NULL DEFAULT 0,
    cost_micros      BIGINT NOT NULL DEFAULT 0,          -- platform cost in micro-dollars
    status           VARCHAR(20) NOT NULL DEFAULT 'ok', -- ok | rejected_quota | rejected_error
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for marketplace queries
CREATE INDEX IF NOT EXISTS idx_entitlements_buyer     ON entitlements(buyer_id);
CREATE INDEX IF NOT EXISTS idx_entitlements_agent     ON entitlements(agent_id);
CREATE INDEX IF NOT EXISTS idx_entitlements_status    ON entitlements(status);
CREATE INDEX IF NOT EXISTS idx_usage_events_entitlement ON usage_events(entitlement_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_buyer     ON usage_events(buyer_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_created   ON usage_events(created_at DESC);

-- Add is_hosted flag to agents so local-only vs marketplace agents are distinguishable
ALTER TABLE agents ADD COLUMN IF NOT EXISTS is_hosted BOOLEAN NOT NULL DEFAULT false;
-- is_hosted = true: platform runs it in sandbox, buyer calls over remote MCP
-- is_hosted = false: traditional downloadable pack (existing behavior)

-- Add price column for marketplace agents (NULL = not for sale yet)
ALTER TABLE agents ADD COLUMN IF NOT EXISTS price_cents INT;
