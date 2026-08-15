export type FailoverMode = 'conservative' | 'balanced' | 'aggressive'
export type FailureScope =
  | 'request'
  | 'credential'
  | 'channel'
  | 'cluster'
  | 'provider'
export type FailoverAction =
  | 'none'
  | 'failover'
  | 'retry_later'
  | 'abort'
  | 'manual'

export type Cluster = {
  id: number
  name: string
  type: string
  status: number
  remark: string
  archived: boolean
  created_time: number
  updated_time: number
}

export type ClusterPool = {
  id: number
  cluster_id: number
  tier: number
  name: string
  status: number
  cost_factor: number
  remark: string
  created_time: number
  updated_time: number
}

export type FailoverPolicy = {
  id: number
  name: string
  mode: FailoverMode
  enabled: boolean
  same_pool_retries: number
  connect_timeout_ms: number
  first_byte_timeout_ms: number
  max_pool_attempts: number
  max_cluster_attempts: number
  max_total_attempts: number
  total_failover_budget_ms: number
  switch_status_codes: string
  switch_error_codes: string
  circuit_failure_threshold: number
  circuit_window_seconds: number
  circuit_cooldown_seconds: number
  circuit_half_open_requests: number
  allow_paid_escalation: boolean
  allow_fallback: boolean
  max_cost_multiplier: number
  created_time: number
  updated_time: number
}

export type UpstreamErrorMapping = {
  id: number
  cluster_type: string
  raw_code: string
  status_code: number
  alltoken_code: number
  category: string
  failure_scope: FailureScope
  action: FailoverAction
  retryable: boolean
  enabled: boolean
}

export type FailoverGroup = {
  id: number
  name: string
  policy_id: number
  enabled: boolean
  created_time: number
  updated_time: number
}

export type FailoverGroupMember = {
  id: number
  failover_group_id: number
  cluster_id: number
  priority: number
  weight: number
}

export type FailoverRule = {
  id: number
  failover_group_id: number
  model_pattern: string
  route_pattern: string
  user_group: string
  policy_id: number
  priority: number
  enabled: boolean
}

export type FailoverConfig = {
  clusters: Cluster[]
  pools: ClusterPool[]
  policies: FailoverPolicy[]
  policy_steps: unknown[]
  groups: FailoverGroup[]
  group_members: FailoverGroupMember[]
  rules: FailoverRule[]
  error_mappings: UpstreamErrorMapping[]
}

export type FailoverMonitoringSource = {
  name: 'prometheus' | 'alertmanager' | 'grafana'
  status: 'healthy' | 'degraded' | 'unavailable' | 'not_configured' | 'pending'
  message?: string
}

export type FailoverMonitoringMetrics = {
  request_rps: number
  error_rate: number
  p95_latency_seconds: number
  in_flight: number
  cluster_failovers: number
  pool_failovers: number
  open_circuits: number
  database_usage: number
  redis_timeouts: number
}

export type FailoverMonitoringAlert = {
  fingerprint: string
  name: string
  severity: string
  status: string
  summary: string
  description: string
  cluster_code?: string
  pool_tier?: string
  instance?: string
  started_at: string
}

export type FailoverMonitoringSnapshot = {
  updated_at: number
  window: string
  metrics: FailoverMonitoringMetrics
  alerts: FailoverMonitoringAlert[]
  sources: FailoverMonitoringSource[]
  grafana_url?: string
}
