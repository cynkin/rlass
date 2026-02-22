CREATE TABLE IF NOT EXISTS rules (
    id          SERIAL PRIMARY KEY,
    rule_id     VARCHAR(100) UNIQUE NOT NULL,
    client_id   VARCHAR(100),
    algorithm   VARCHAR(50) NOT NULL DEFAULT 'fixed_window',
    "limit"     INTEGER NOT NULL,
    window_secs INTEGER NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS request_logs (
    id          BIGSERIAL PRIMARY KEY,
    client_id   VARCHAR(100) NOT NULL,
    rule_id     VARCHAR(100) NOT NULL,
    allowed     BOOLEAN NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_request_logs_client_time 
    ON request_logs(client_id, created_at);

CREATE INDEX IF NOT EXISTS idx_request_logs_rule_time 
    ON request_logs(rule_id, created_at);