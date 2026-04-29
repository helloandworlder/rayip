-- +goose Up
CREATE TABLE IF NOT EXISTS runtime_lab_accounts (
  proxy_account_id uuid PRIMARY KEY,
  node_id uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  runtime_email text NOT NULL UNIQUE,
  protocol text NOT NULL,
  listen_ip text NOT NULL,
  port integer NOT NULL,
  username text NOT NULL,
  password text NOT NULL,
  expires_at timestamptz,
  egress_limit_bps bigint NOT NULL DEFAULT 0,
  ingress_limit_bps bigint NOT NULL DEFAULT 0,
  max_connections integer NOT NULL DEFAULT 0,
  status text NOT NULL,
  policy_version bigint NOT NULL DEFAULT 1,
  desired_generation bigint NOT NULL DEFAULT 1,
  applied_generation bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (protocol IN ('SOCKS5', 'HTTP')),
  CHECK (status IN ('ENABLED', 'DISABLED', 'DELETED'))
);

CREATE INDEX IF NOT EXISTS idx_runtime_lab_accounts_node_id ON runtime_lab_accounts(node_id);
CREATE INDEX IF NOT EXISTS idx_runtime_lab_accounts_status ON runtime_lab_accounts(status);

CREATE TABLE IF NOT EXISTS runtime_lab_apply_results (
  command_id uuid PRIMARY KEY,
  proxy_account_id uuid,
  node_id uuid,
  operation text NOT NULL,
  status text NOT NULL,
  error_code text NOT NULL DEFAULT '',
  error_message text NOT NULL DEFAULT '',
  applied_generation bigint NOT NULL DEFAULT 0,
  usage jsonb NOT NULL DEFAULT '{}'::jsonb,
  digest jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_runtime_lab_apply_results_account ON runtime_lab_apply_results(proxy_account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runtime_lab_apply_results_node ON runtime_lab_apply_results(node_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS runtime_lab_apply_results;
DROP TABLE IF EXISTS runtime_lab_accounts;
