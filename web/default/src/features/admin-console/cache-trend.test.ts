import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { formatChartTime } from '@/lib/time'

import { buildCacheTrendChartValues } from './cache-trend'

describe('admin console cache trend chart values', () => {
  test('fills zero rates when the cache trend has no points', () => {
    const values = buildCacheTrendChartValues(
      [
        { created_at: 1, model_name: 'paid' },
        { created_at: 2, model_name: 'free' },
      ],
      [
        { Time: '09-02 10:00', Model: 'paid' },
        { Time: '09-02 10:00', Model: 'free' },
        { Time: '09-02 11:00', Model: 'paid' },
        { Time: '09-02 11:00', Model: 'free' },
      ],
      [],
      'hour'
    )

    assert.equal(values.length, 4)
    assert.equal(
      values.every((value) => value.CacheRate === 0),
      true
    )
  })

  test('uses reported rates and fills missing dimension buckets with zero', () => {
    const timestamp = 1_756_793_600
    const reportedTime = formatChartTime(timestamp, 'hour')
    const missingTime = formatChartTime(timestamp + 3_600, 'hour')
    const values = buildCacheTrendChartValues(
      [{ created_at: timestamp, model_name: 'paid' }],
      [
        { Time: reportedTime, Model: 'paid' },
        { Time: missingTime, Model: 'paid' },
      ],
      [
        {
          timestamp,
          name: 'paid',
          cache_input_tokens: 100,
          cache_read_tokens: 25,
          cache_write_tokens: 0,
          cache_hit_requests: 1,
          cache_eligible_requests: 1,
          cache_hit_rate: 25,
        },
      ],
      'hour'
    )

    assert.deepEqual(
      values.map((value) => value.CacheRate),
      [25, 0]
    )
  })

  test('keeps same-named channels separate by their display ID', () => {
    const timestamp = 1_756_793_600
    const time = formatChartTime(timestamp, 'hour')
    const values = buildCacheTrendChartValues(
      [
        { created_at: timestamp, model_name: 'ChatGPT Plus #1' },
        { created_at: timestamp, model_name: 'ChatGPT Plus #2' },
      ],
      [
        { Time: time, Model: 'ChatGPT Plus #1' },
        { Time: time, Model: 'ChatGPT Plus #2' },
      ],
      [
        {
          timestamp,
          name: 'ChatGPT Plus',
          channel_id: 1,
          cache_input_tokens: 100,
          cache_read_tokens: 40,
          cache_write_tokens: 0,
          cache_hit_requests: 1,
          cache_eligible_requests: 1,
          cache_hit_rate: 40,
        },
        {
          timestamp,
          name: 'ChatGPT Plus',
          channel_id: 2,
          cache_input_tokens: 200,
          cache_read_tokens: 20,
          cache_write_tokens: 0,
          cache_hit_requests: 1,
          cache_eligible_requests: 1,
          cache_hit_rate: 10,
        },
      ],
      'hour'
    )

    assert.deepEqual(
      values.map((value) => [value.Model, value.CacheRate]),
      [
        ['ChatGPT Plus #1', 40],
        ['ChatGPT Plus #2', 10],
      ]
    )
  })
})
