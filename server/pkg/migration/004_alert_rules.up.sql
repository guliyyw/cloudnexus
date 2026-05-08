CREATE TABLE IF NOT EXISTS alert_rules (
    id               BIGINT PRIMARY KEY,
    name             VARCHAR(128) NOT NULL,
    description      VARCHAR(512) DEFAULT '',
    enabled          BOOLEAN      DEFAULT TRUE,
    node_name        VARCHAR(64)  DEFAULT '*',
    trigger_type     VARCHAR(32)  DEFAULT 'status_change',
    condition        TEXT         DEFAULT '{}',
    webhook_url      VARCHAR(512) NOT NULL,
    cooldown_seconds INT          DEFAULT 300,
    created_by       BIGINT       DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_alert_rules_name ON alert_rules(name);

CREATE TABLE IF NOT EXISTS alert_history (
    id             BIGINT PRIMARY KEY,
    rule_id        BIGINT       NOT NULL,
    rule_name      VARCHAR(128) NOT NULL,
    node_name      VARCHAR(64)  NOT NULL,
    alert_type     VARCHAR(32)  NOT NULL,
    status         VARCHAR(16)  DEFAULT 'firing',
    message        TEXT         DEFAULT '',
    fired_at       TIMESTAMPTZ  NOT NULL,
    resolved_at    TIMESTAMPTZ,
    webhook_url    VARCHAR(512) DEFAULT '',
    response_code  INT          DEFAULT 0,
    error_message  TEXT         DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_alert_history_rule  ON alert_history(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_node  ON alert_history(node_name);
CREATE INDEX IF NOT EXISTS idx_alert_history_fired ON alert_history(fired_at);
