import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { formatChannelDisplayName } from './channel-display'

describe('admin console channel display names', () => {
  test('adds the channel ID to named channels', () => {
    assert.equal(
      formatChannelDisplayName('ChatGPT Plus', 12),
      'ChatGPT Plus #12'
    )
  })

  test('normalizes deleted and unrecorded channel labels', () => {
    assert.equal(formatChannelDisplayName('channel-12', 12), '渠道 #12')
    assert.equal(formatChannelDisplayName(undefined, 12), '渠道 #12')
    assert.equal(formatChannelDisplayName(undefined, 0), '未记录渠道')
  })

  test('does not duplicate an existing ID suffix', () => {
    assert.equal(
      formatChannelDisplayName('ChatGPT Plus #12', 12),
      'ChatGPT Plus #12'
    )
  })
})
