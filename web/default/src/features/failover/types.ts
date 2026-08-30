export type RoutingMode = 'cost_first' | 'balanced' | 'stability_first'
export type BillingGroupType = 'toB' | 'toC'
export type RoutingStrategy = 'priority' | 'weighted'
export type ProfitGuardMode = 'off' | 'warn' | 'enforce'
export type FailureScope = 'request' | 'credential' | 'channel' | 'provider'
export type RoutingAction =
  | 'none'
  | 'retry_channel'
  | 'switch_channel'
  | 'retry_later'
  | 'abort'
  | 'manual'

export type BillingGroupRoute = {
  id: number
  billing_group: string
  name: string
  mode: RoutingMode
  group_type: BillingGroupType
  strategy_config: string
  enabled: boolean
  max_total_attempts: number
  total_timeout_ms: number
  circuit_failure_threshold: number
  circuit_window_seconds: number
  circuit_cooldown_seconds: number
  circuit_half_open_requests: number
  profit_guard_mode: ProfitGuardMode
  minimum_profit_margin: number
  created_time: number
  updated_time: number
}

export type BillingGroupChannel = {
  id: number
  billing_group_route_id: number
  channel_id: number
  priority: number
  weight: number
  max_attempts: number
  enabled: boolean
  cost_factor: number
}

export type UpstreamErrorMapping = {
  id: number
  channel_id: number
  channel_type: number
  raw_code: string
  status_code: number
  alltoken_code: number
  category: string
  failure_scope: FailureScope
  action: RoutingAction
  retryable: boolean
  enabled: boolean
}

export type FailoverConfig = {
  routes: BillingGroupRoute[]
  route_channels: BillingGroupChannel[]
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
  channel_switches: number
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
  channel_id?: string
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
