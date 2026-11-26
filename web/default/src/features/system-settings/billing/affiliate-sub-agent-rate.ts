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

export function formatSubAgentRatePercent(rateBps: number): string {
  const digits = String(rateBps).padStart(3, '0')
  const whole = digits.slice(0, -2)
  const fraction = digits.slice(-2).replace(/0+$/, '')
  return fraction ? `${whole}.${fraction}` : whole
}

export function parseSubAgentRateBps(ratePercent: string): number | null {
  if (!/^(?:\d+|\d*\.\d{1,2})$/.test(ratePercent)) return null
  const [whole, fraction = ''] = ratePercent.split('.')
  const rateBps = Number(whole + fraction.padEnd(2, '0'))
  return Number.isSafeInteger(rateBps) ? rateBps : null
}
