/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import Antigravity from '@lobehub/icons/es/Antigravity'
import Claude from '@lobehub/icons/es/Claude'
import ClaudeCode from '@lobehub/icons/es/ClaudeCode'
import Codex from '@lobehub/icons/es/Codex'
import DeepSeek from '@lobehub/icons/es/DeepSeek'
import GeminiCLI from '@lobehub/icons/es/GeminiCLI'
import GithubCopilot from '@lobehub/icons/es/GithubCopilot'
import Grok from '@lobehub/icons/es/Grok'
import HermesAgent from '@lobehub/icons/es/HermesAgent'
import Kimi from '@lobehub/icons/es/Kimi'
import Minimax from '@lobehub/icons/es/Minimax'
import OpenClaw from '@lobehub/icons/es/OpenClaw'
import OpenCode from '@lobehub/icons/es/OpenCode'
import type { IconType } from '@lobehub/icons/es/types'

import piLogo from '@/assets/home/pi-logo.svg'

import type { CCSwitchApp } from '../lib/model-catalog'
import ohMyPiLogo from './cc-switch-icons/oh-my-pi.svg'
import workBuddyLogo from './cc-switch-icons/workbuddy.svg'

const APP_ICONS = {
  agy: Antigravity.Color,
  'claude-desktop': Claude.Color,
  claude: ClaudeCode.Color,
  codex: Codex.Color,
  copilot: GithubCopilot,
  dsh: DeepSeek.Color,
  gemini: GeminiCLI.Color,
  grokbuild: Grok,
  hermes: HermesAgent,
  kimi: Kimi,
  mcode: Minimax.Color,
  openclaw: OpenClaw.Color,
  opencode: OpenCode,
} satisfies Record<Exclude<CCSwitchApp, 'pi' | 'omp' | 'workbuddy'>, IconType>

const APP_LOGOS = {
  omp: ohMyPiLogo,
  pi: piLogo,
  workbuddy: workBuddyLogo,
}

export function CCSwitchAppIcon(props: {
  app: CCSwitchApp
  className?: string
}) {
  if (props.app === 'omp' || props.app === 'pi' || props.app === 'workbuddy') {
    return (
      <img
        src={APP_LOGOS[props.app]}
        alt=''
        aria-hidden='true'
        width={16}
        height={16}
        className={props.className}
      />
    )
  }

  const Icon = APP_ICONS[props.app]
  return <Icon size={16} aria-hidden='true' className={props.className} />
}
