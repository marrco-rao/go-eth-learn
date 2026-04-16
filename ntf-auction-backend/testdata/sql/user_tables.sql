-- Base SQL fixtures for ntf-auction-backend (SQLite)

-- 用户表
CREATE TABLE IF NOT EXISTS users (
  id            INTEGER      PRIMARY KEY AUTOINCREMENT,
  username      VARCHAR(64)  NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 请求日志表
CREATE TABLE IF NOT EXISTS logs (
  id          INTEGER     PRIMARY KEY AUTOINCREMENT,
  level       VARCHAR(16) NOT NULL,
  message     TEXT        NOT NULL,
  method      VARCHAR(16) NOT NULL,
  path        VARCHAR(255) NOT NULL,
  status_code INTEGER     NOT NULL,
  created_at  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 交易记录表
CREATE TABLE IF NOT EXISTS tx_records (
  id                     INTEGER      PRIMARY KEY AUTOINCREMENT,
  idempotency_key        VARCHAR(128) NOT NULL UNIQUE,
  biz_id                 VARCHAR(128),
  from_address           VARCHAR(64)  NOT NULL,
  to_address             VARCHAR(64)  NOT NULL,
  value_wei              VARCHAR(78)  NOT NULL,
  data_hex               TEXT,
  gas_limit              INTEGER      NOT NULL,
  max_fee_per_gas_wei    VARCHAR(78),
  max_priority_fee_gas_wei VARCHAR(78),
  nonce                  INTEGER,
  tx_hash                VARCHAR(80),
  status                 VARCHAR(32)  NOT NULL,
  error_message          TEXT,
  retry_count            INTEGER      NOT NULL DEFAULT 0,
  created_at             DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at             DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tx_records_biz_id   ON tx_records (biz_id);
CREATE INDEX IF NOT EXISTS idx_tx_records_nonce     ON tx_records (nonce);
CREATE INDEX IF NOT EXISTS idx_tx_records_tx_hash   ON tx_records (tx_hash);
CREATE INDEX IF NOT EXISTS idx_tx_records_status    ON tx_records (status);

-- 合约事件表（链上订阅事件持久化，TxHash+LogIndex 联合唯一防止重复入库）
CREATE TABLE IF NOT EXISTS contract_events (
  id               INTEGER      PRIMARY KEY AUTOINCREMENT,
  network_ws_url   VARCHAR(255) NOT NULL,
  contract_address VARCHAR(64)  NOT NULL,
  event_type       VARCHAR(64)  NOT NULL,
  tx_hash          VARCHAR(80)  NOT NULL,
  block_number     INTEGER      NOT NULL,
  log_index        INTEGER      NOT NULL,
  account_address  VARCHAR(64),
  token_id         VARCHAR(78),
  amount_wei       VARCHAR(78),
  end_time         INTEGER      NOT NULL DEFAULT 0,
  payment_method   INTEGER      NOT NULL DEFAULT 0,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT idx_contract_event_dedup UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_contract_events_network_ws_url    ON contract_events (network_ws_url);
CREATE INDEX IF NOT EXISTS idx_contract_events_contract_address  ON contract_events (contract_address);
CREATE INDEX IF NOT EXISTS idx_contract_events_event_type        ON contract_events (event_type);
CREATE INDEX IF NOT EXISTS idx_contract_events_block_number      ON contract_events (block_number);
CREATE INDEX IF NOT EXISTS idx_contract_events_account_address   ON contract_events (account_address);
CREATE INDEX IF NOT EXISTS idx_contract_events_token_id          ON contract_events (token_id);
