import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { formatChartTime } from '@/lib/time'

import { buildCacheTrendChartValues, summarizeCacheTrend } from './cache-trend'

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
      values.every(
        (value) =>
          value.CacheRate === 0 &&
          value.CacheInputTokens === 0 &&
          value.CacheReadTokens === 0 &&
          value.CacheWriteTokens === 0 &&
          value.CacheHitRequests === 0 &&
          value.CacheEligibleRequests === 0
      ),
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
      values.map((value) => ({
        rate: value.CacheRate,
        inputTokens: value.CacheInputTokens,
        readTokens: value.CacheReadTokens,
        writeTokens: value.CacheWriteTokens,
        hitRequests: value.CacheHitRequests,
        eligibleRequests: value.CacheEligibleRequests,
      })),
      [
        {
          rate: 25,
          inputTokens: 100,
          readTokens: 25,
          writeTokens: 0,
          hitRequests: 1,
          eligibleRequests: 1,
        },
        {
          rate: 0,
          inputTokens: 0,
          readTokens: 0,
          writeTokens: 0,
          hitRequests: 0,
          eligibleRequests: 0,
        },
      ]
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

  test('summarizes token and request cache hit metrics', () => {
    const summary = summarizeCacheTrend([
      {
        Time: '09-02 10:00',
        Model: 'paid',
        CacheRate: 25,
        CacheInputTokens: 100,
        CacheReadTokens: 25,
        CacheWriteTokens: 10,
        CacheHitRequests: 1,
        CacheEligibleRequests: 2,
      },
      {
        Time: '09-02 11:00',
        Model: 'paid',
        CacheRate: 50,
        CacheInputTokens: 300,
        CacheReadTokens: 150,
        CacheWriteTokens: 20,
        CacheHitRequests: 2,
        CacheEligibleRequests: 3,
      },
    ])

    assert.deepEqual(summary, {
      cacheInputTokens: 400,
      cacheReadTokens: 175,
      cacheWriteTokens: 30,
      cacheHitRequests: 3,
      cacheEligibleRequests: 5,
      tokenHitRate: 43.75,
      requestHitRate: 60,
    })
  })

  test('summarizes only the groups or channels selected in the legend', () => {
    const values = [
      {
        Time: '09-02 10:00',
        Model: 'ChatGPT Plus #20',
        CacheRate: 40,
        CacheInputTokens: 100,
        CacheReadTokens: 40,
        CacheWriteTokens: 10,
        CacheHitRequests: 2,
        CacheEligibleRequests: 4,
      },
      {
        Time: '09-02 10:00',
        Model: 'DeepSeek官方 #22',
        CacheRate: 20,
        CacheInputTokens: 200,
        CacheReadTokens: 40,
        CacheWriteTokens: 5,
        CacheHitRequests: 1,
        CacheEligibleRequests: 5,
      },
    ]

    assert.deepEqual(summarizeCacheTrend(values, ['DeepSeek官方 #22']), {
      cacheInputTokens: 200,
      cacheReadTokens: 40,
      cacheWriteTokens: 5,
      cacheHitRequests: 1,
      cacheEligibleRequests: 5,
      tokenHitRate: 20,
      requestHitRate: 20,
    })
  })
})
