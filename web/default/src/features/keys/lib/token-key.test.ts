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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { withTokenPrefix } from './token-key'

describe('token key prefix normalization', () => {
  test('adds the prefix to a stored suffix', () => {
    assert.equal(withTokenPrefix('abc123'), 'sk-abc123')
  })

  test('does not duplicate an existing prefix', () => {
    assert.equal(withTokenPrefix('sk-abc123'), 'sk-abc123')
    assert.equal(withTokenPrefix('sk-sk-abc123'), 'sk-abc123')
  })

  test('trims surrounding whitespace and preserves empty input', () => {
    assert.equal(withTokenPrefix('  sk-abc123  '), 'sk-abc123')
    assert.equal(withTokenPrefix('  '), '')
  })
})
