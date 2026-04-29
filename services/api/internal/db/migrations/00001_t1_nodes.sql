-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS nodes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  status text NOT NULL DEFAULT 'ONLINE',
  bundle_version text NOT NULL DEFAULT '',
  agent_version text NOT NULL DEFAULT '',
  xray_version text NOT NULL DEFAULT '',
  capabilities jsonb NOT NULL DEFAULT '[]'::jsonb,
  last_online_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_last_online_at ON nodes(last_online_at);

CREATE TABLE IF NOT EXISTS node_agent_sessions (
  session_id text PRIMARY KEY,
  node_id uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  api_instance_id text NOT NULL,
  status text NOT NULL DEFAULT 'CONNECTED',
  bundle_version text NOT NULL DEFAULT '',
  connected_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_node_agent_sessions_node_id ON node_agent_sessions(node_id);
CREATE INDEX IF NOT EXISTS idx_node_agent_sessions_api_instance ON node_agent_sessions(api_instance_id);
CREATE INDEX IF NOT EXISTS idx_node_agent_sessions_last_seen_at ON node_agent_sessions(last_seen_at);

CREATE TABLE IF NOT EXISTS node_capability_snapshots (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id uuid NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  bundle_version text NOT NULL DEFAULT '',
  agent_version text NOT NULL DEFAULT '',
  xray_version text NOT NULL DEFAULT '',
  capabilities jsonb NOT NULL DEFAULT '[]'::jsonb,
  capabilities_hash text NOT NULL,
  captured_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(node_id, bundle_version, agent_version, xray_version, capabilities_hash)
);

CREATE INDEX IF NOT EXISTS idx_node_capability_snapshots_node_id ON node_capability_snapshots(node_id);

-- +goose Down
DROP TABLE IF EXISTS node_capability_snapshots;
DROP TABLE IF EXISTS node_agent_sessions;
DROP TABLE IF EXISTS nodes;
