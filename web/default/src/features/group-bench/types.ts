/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type GroupBenchEndpointType = 'openai' | 'anthropic' | 'openai-response'

export type GroupBenchModel = {
  model: string
  endpoint_type: GroupBenchEndpointType | ''
}

export type GroupBenchPreset = {
  key: string
  name: string
  prompt: string
  max_tokens: number
}

export type GroupBenchConfig = {
  group: string
  enabled: boolean
  preset: string
  models: GroupBenchModel[]
  interval_minutes: number
  last_run_at: number | null
  next_run_at: number
  manual_pending: boolean
  presets: GroupBenchPreset[]
  group_models: string[]
}

export type GroupBenchConfigPayload = Pick<
  GroupBenchConfig,
  'group' | 'enabled' | 'preset' | 'models' | 'interval_minutes'
>

export type GroupBenchRun = {
  id: number
  group: string
  round_at: number
  channel_id: number
  channel_name: string
  model: string
  endpoint_type: string
  preset: string
  success: boolean
  error_message?: string
  latency_ms: number
  input_tokens: number
  output_tokens: number
  stop_reason: string
  response_model: string
  truncated: boolean
  has_artifact: boolean
  metrics: string
  created_at: number
}

export type GroupBenchArtifact = {
  run_id: number
  content_type: string
  content: string
  bytes: number
  clipped: boolean
}

export type SvgMetrics = {
  bytes: number
  elements: number
  animate: number
  animate_transform: number
  animate_motion: number
  rotate: number
  indefinite: number
  css_keyframes: number
  has_script: boolean
  animated: boolean
}
