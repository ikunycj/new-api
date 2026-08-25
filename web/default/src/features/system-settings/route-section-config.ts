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

// Route validation must not import the JSX section registries. Doing so pulls
// every settings form and its dependencies into the public application entry.
export const AUTH_SECTION_IDS: readonly string[] = [
  'basic-auth',
  'oauth',
  'passkey',
  'bot-protection',
  'custom-oauth',
]
export const AUTH_DEFAULT_SECTION = 'basic-auth'

export const BILLING_SECTION_IDS: readonly string[] = [
  'quota',
  'currency',
  'model-pricing',
  'group-pricing',
  'payment',
  'affiliate',
  'checkin',
]
export const BILLING_DEFAULT_SECTION = 'quota'

export const CONTENT_SECTION_IDS: readonly string[] = [
  'dashboard',
  'announcements',
  'api-info',
  'faq',
  'uptime-kuma',
  'chat',
  'drawing',
]
export const CONTENT_DEFAULT_SECTION = 'dashboard'

export const MODELS_SECTION_IDS: readonly string[] = [
  'global',
  'routing-reliability',
  'gemini',
  'claude',
  'grok',
  'channel-affinity',
  'model-deployment',
]
export const MODELS_DEFAULT_SECTION = 'global'

export const OPERATIONS_SECTION_IDS: readonly string[] = [
  'behavior',
  'alerts',
  'email',
  'worker',
  'logs',
  'performance',
  'loadtest',
  'update-checker',
]
export const OPERATIONS_DEFAULT_SECTION = 'behavior'

export const SECURITY_SECTION_IDS: readonly string[] = [
  'rate-limit',
  'sensitive-words',
  'ssrf',
  'token-limits',
]
export const SECURITY_DEFAULT_SECTION = 'rate-limit'

export const SITE_SECTION_IDS: readonly string[] = [
  'system-info',
  'notice',
  'header-navigation',
  'sidebar-modules',
]
export const SITE_DEFAULT_SECTION = 'system-info'
