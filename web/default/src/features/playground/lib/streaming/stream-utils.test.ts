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

import { parseStreamErrorDetails } from './stream-utils'

describe('stream error parsing', () => {
  test('reads request IDs from nested OpenAI errors', () => {
    assert.deepEqual(
      parseStreamErrorDetails(
        JSON.stringify({
          error: {
            code: 'bad_response',
            message: 'temporary failure',
            request_id: 'req_nested',
          },
        })
      ),
      {
        errorCode: 'bad_response',
        errorMessage: 'temporary failure',
        requestId: 'req_nested',
      }
    )
  })

  test('reads request IDs from root Responses errors', () => {
    assert.deepEqual(
      parseStreamErrorDetails(
        JSON.stringify({
          code: 'server_error',
          message: 'temporary failure',
          request_id: 'req_root',
        })
      ),
      {
        errorCode: 'server_error',
        errorMessage: 'temporary failure',
        requestId: 'req_root',
      }
    )
  })
})
