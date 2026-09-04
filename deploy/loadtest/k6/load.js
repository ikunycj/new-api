import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const profile = __ENV.LOAD_PROFILE || 'smoke';
const loadTestConfig = JSON.parse(open(__ENV.LOADTEST_CONFIG_FILE || '/config/loadtest.config.json'));
const workloadConfig = loadTestConfig.workload || {};
const thresholdConfig = loadTestConfig.thresholds || {};
const mockConfig = loadTestConfig.mock || {};
const profileConfig = workloadConfig.profiles || {};
const pacingConfig = workloadConfig.pacing || {};
const requestConfig = workloadConfig.request || {};
const baseURL = normalizeBaseURL(__ENV.BASE_URL || 'http://new-api:3000');
const loadTestModel = __ENV.LOADTEST_MODEL || workloadConfig.model || 'gpt-3.5-turbo';
const userCount = numberEnv('LOADTEST_USERS', workloadConfig.users || 1000);
const loadTestTokens = tokenListEnv('LOADTEST_TOKENS');
const failoverMode = __ENV.FAILOVER_MODE || 'balanced';
const mockControlA = normalizeBaseURL(__ENV.MOCK_CONTROL_A || 'http://mock-upstream:8080');
const mockControlB = normalizeBaseURL(__ENV.MOCK_CONTROL_B || 'http://mock-upstream-b:8080');
const mockControlC = normalizeBaseURL(__ENV.MOCK_CONTROL_C || 'http://mock-upstream-c:8080');
const smokeDuration = __ENV.SMOKE_DURATION || workloadConfig.smoke?.duration || '1m';
const smokeVUs = numberEnv('SMOKE_VUS', workloadConfig.smoke?.vus || 10);
const burstMaxDuration = __ENV.BURST_MAX_DURATION || workloadConfig.burst?.max_duration || '5m';
const capacityRates = numberListEnv('CAPACITY_RATES', workloadConfig.capacity?.rates || [100, 200, 300, 400, 500, 600, 700, 800]);
const capacityRampDuration = __ENV.CAPACITY_RAMP_DURATION || workloadConfig.capacity?.ramp_duration || '30s';
const capacityStageDuration = __ENV.CAPACITY_STAGE_DURATION || workloadConfig.capacity?.stage_duration || '2m';
const capacityPreallocatedVUs = numberEnv('CAPACITY_PREALLOCATED_VUS', workloadConfig.capacity?.preallocated_vus || 500);
const capacityMaxVUs = numberEnv('CAPACITY_MAX_VUS', workloadConfig.capacity?.max_vus || 2000);
const capacityFirstRampDuration = workloadConfig.capacity?.first_ramp_duration || '10s';
const capacityCooldownDuration = workloadConfig.capacity?.cooldown_duration || '30s';
const capacityGracefulStop = workloadConfig.capacity?.graceful_stop || '1m';
const requestTimeout = workloadConfig.request_timeout || '120s';
const modelListTimeout = workloadConfig.model_list_timeout || '30s';
const maxOutputTokens = numberEnv('MAX_OUTPUT_TOKENS', workloadConfig.max_output_tokens || 256);
const requestTemperature = numberValue('REQUEST_TEMPERATURE', requestConfig.temperature ?? 0);
const requestMessage = __ENV.LOADTEST_MESSAGE || requestConfig.message || 'Return a deterministic load-test response.';

const applicationErrors = new Rate('new_api_application_errors');
const timeToFirstByte = new Trend('new_api_time_to_first_byte', true);
const completedRequests = new Counter('new_api_completed_requests');
const httpResponses = new Counter('new_api_http_responses');
const promptTokens = new Counter('new_api_prompt_tokens');
const completionTokens = new Counter('new_api_completion_tokens');
const totalTokens = new Counter('new_api_total_tokens');
const usageMissing = new Rate('new_api_usage_missing');

const commonThresholds = {
  http_req_failed: [`rate<${thresholdConfig.error_rate ?? 0.01}`],
  new_api_application_errors: [`rate<${thresholdConfig.error_rate ?? 0.01}`],
  http_req_duration: [`p(95)<${thresholdConfig.request_p95_ms ?? 2000}`, `p(99)<${thresholdConfig.request_p99_ms ?? 5000}`],
  new_api_time_to_first_byte: [`p(95)<${thresholdConfig.ttfb_p95_ms ?? 1000}`],
};

export const options = buildOptions(profile);

export function setup() {
  if (profile !== 'channel-failover') {
    return;
  }
  const responses = http.batch([
    ['POST', `${mockControlA}/control/reset?tokens=${mockChannelTokens(1, 300)}`, null],
    ['POST', `${mockControlB}/control/reset?tokens=${mockChannelTokens(2, 3000)}`, null],
    ['POST', `${mockControlC}/control/reset?tokens=${mockChannelTokens(3, 3000)}`, null],
  ]);
  const reset = check(responses, {
    'mock channels reset': (items) => items.every((item) => item.status === 200),
  });
  if (!reset) {
    throw new Error('failed to reset mock channels');
  }
}

export function chat() {
  makeChatRequest(false);
  sleep(randomPacing('chat_min_seconds', 0.2, 'chat_max_seconds', 1.0));
}

export function stream() {
  makeChatRequest(true);
  sleep(randomPacing('stream_min_seconds', 0.1, 'stream_max_seconds', 0.5));
}

export function modelList() {
  const response = http.get(`${baseURL}/v1/models`, {
    headers: headersForVU(),
    tags: { endpoint: 'models', stream: 'false', profile },
    timeout: modelListTimeout,
  });
  recordResponse(response, 'model list');
  sleep(randomPacing('model_list_min_seconds', 0.5, 'model_list_max_seconds', 1.5));
}

export function capacityChat() {
  makeChatRequest(false);
}

export default function () {
  makeChatRequest(false);
}

function makeChatRequest(streaming) {
  const response = http.post(
    `${baseURL}/v1/chat/completions`,
    JSON.stringify({
      model: loadTestModel,
      messages: [{ role: 'user', content: requestMessage }],
      stream: streaming,
      stream_options: streaming ? { include_usage: true } : undefined,
      max_tokens: maxOutputTokens,
      temperature: requestTemperature,
    }),
    {
      headers: headersForVU(),
      tags: { endpoint: 'chat_completions', stream: String(streaming), profile },
      timeout: requestTimeout,
    },
  );
  timeToFirstByte.add(response.timings.waiting, { stream: String(streaming), profile });
  recordResponse(response, streaming ? 'streaming chat' : 'chat');
  if (!streaming) {
    recordUsage(response);
  }
}

function recordUsage(response) {
  let usage;
  if (response.status === 200) {
    try {
      usage = response.json().usage;
    } catch (_) {
      usage = null;
    }
  }
  const valid = usage
    && Number.isFinite(usage.prompt_tokens)
    && Number.isFinite(usage.completion_tokens)
    && Number.isFinite(usage.total_tokens);
  usageMissing.add(!valid, { profile });
  if (!valid) {
    return;
  }
  const tags = { profile };
  promptTokens.add(usage.prompt_tokens, tags);
  completionTokens.add(usage.completion_tokens, tags);
  totalTokens.add(usage.total_tokens, tags);
}

function recordResponse(response, name) {
  httpResponses.add(1, { profile, endpoint: name, status: String(response.status) });
  const valid = check(response, {
    [`${name}: status is 200`]: (result) => result.status === 200,
    [`${name}: no gateway error`]: (result) => !result.body || !result.body.includes('"error"'),
  });
  applicationErrors.add(!valid, { profile });
  completedRequests.add(1, { profile });
}

function headersForVU() {
  const tokenNumber = ((__VU - 1) % userCount) + 1;
  const token = loadTestTokens.length > 0
    ? loadTestTokens[(__VU - 1) % loadTestTokens.length]
    : `sk-loadtest${String(tokenNumber).padStart(5, '0')}`;
  return {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Load-Test-ID': `${profile}-${__VU}-${__ITER}`,
    'X-Alltoken-Failover-Mode': failoverMode,
  };
}

function buildOptions(selectedProfile) {
  const smoke = profileConfig.smoke || workloadConfig.smoke || {};
  const step = profileConfig.step || {};
  const steady = profileConfig.steady || {};
  const spike = profileConfig.spike || {};
  const burst = profileConfig.burst || workloadConfig.burst || {};
  const streamProfile = profileConfig.stream || {};
  const mixed = profileConfig.mixed || {};
  const soak = profileConfig.soak || {};
  const channelFailover = profileConfig.channel_failover || {};
  const profiles = {
    smoke: {
      scenarios: {
        smoke: { executor: 'constant-vus', exec: 'chat', vus: smokeVUs, duration: smokeDuration, gracefulStop: smoke.graceful_stop || '30s' },
      },
    },
    step: {
      scenarios: {
        step: {
          executor: 'ramping-vus',
          exec: 'chat',
          startVUs: step.start_vus ?? 0,
          stages: step.stages || [],
          gracefulRampDown: step.graceful_ramp_down || '30s',
        },
      },
    },
    steady: {
      scenarios: {
        steady: { executor: 'constant-vus', exec: 'chat', vus: steady.vus || 1000, duration: steady.duration || '30m', gracefulStop: steady.graceful_stop || '2m' },
      },
    },
    spike: {
      scenarios: {
        spike: {
          executor: 'ramping-vus',
          exec: 'chat',
          startVUs: spike.start_vus ?? 0,
          stages: spike.stages || [],
          gracefulRampDown: spike.graceful_ramp_down || '30s',
        },
      },
    },
    burst: {
      scenarios: {
        burst: {
          executor: 'per-vu-iterations',
          exec: 'chat',
          vus: userCount,
          iterations: 1,
          maxDuration: burst.max_duration || burstMaxDuration,
        },
      },
    },
    stream: {
      scenarios: {
        stream: { executor: 'constant-vus', exec: 'stream', vus: streamProfile.vus || 1000, duration: streamProfile.duration || '15m', gracefulStop: streamProfile.graceful_stop || '2m' },
      },
      thresholds: {
        ...commonThresholds,
        http_req_duration: [`p(95)<${thresholdConfig.stream_p95_ms ?? 5000}`, `p(99)<${thresholdConfig.stream_p99_ms ?? 10000}`],
      },
    },
    mixed: {
      scenarios: {
        stream: { executor: 'constant-vus', exec: 'stream', vus: mixed.stream_vus || 700, duration: mixed.duration || '20m', gracefulStop: mixed.graceful_stop || '2m' },
        chat: { executor: 'constant-vus', exec: 'chat', vus: mixed.chat_vus || 200, duration: mixed.duration || '20m', gracefulStop: mixed.graceful_stop || '2m' },
        models: { executor: 'constant-vus', exec: 'modelList', vus: mixed.model_vus || 100, duration: mixed.duration || '20m', gracefulStop: mixed.model_graceful_stop || '30s' },
      },
      thresholds: {
        ...commonThresholds,
        'http_req_duration{stream:true}': [`p(95)<${thresholdConfig.stream_p95_ms ?? 5000}`],
      },
    },
    soak: {
      scenarios: {
        soak: { executor: 'constant-vus', exec: 'chat', vus: soak.vus || 500, duration: soak.duration || '2h', gracefulStop: soak.graceful_stop || '2m' },
      },
    },
    capacity: {
      scenarios: {
        capacity: {
          executor: 'ramping-arrival-rate',
          exec: 'capacityChat',
          startRate: capacityRates[0],
          timeUnit: '1s',
          preAllocatedVUs: capacityPreallocatedVUs,
          maxVUs: capacityMaxVUs,
          stages: capacityRates.flatMap((rate, index) => [
            { target: rate, duration: index === 0 ? capacityFirstRampDuration : capacityRampDuration },
            { target: rate, duration: capacityStageDuration },
          ]).concat([{ target: 0, duration: capacityCooldownDuration }]),
          gracefulStop: capacityGracefulStop,
        },
      },
      thresholds: {
        ...commonThresholds,
        dropped_iterations: ['count==0'],
        new_api_usage_missing: [`rate<${thresholdConfig.usage_missing_rate ?? 0.001}`],
      },
    },
    'channel-failover': {
      scenarios: {
        channelFailover: {
          executor: 'constant-vus',
          exec: 'chat',
          vus: channelFailover.vus || 5,
          duration: channelFailover.duration || '30s',
          gracefulStop: channelFailover.graceful_stop || '30s',
        },
      },
      thresholds: {
        ...commonThresholds,
        new_api_application_errors: ['rate<0.01'],
      },
    },
  };

  const selected = profiles[selectedProfile];
  if (!selected) {
    throw new Error(`Unknown LOAD_PROFILE: ${selectedProfile}`);
  }
  const selectedScenarios = selected.scenarios;
  return {
    discardResponseBodies: false,
    userAgent: `new-api-loadtest/${selectedProfile}`,
    thresholds: selected.thresholds || commonThresholds,
    scenarios: selectedScenarios,
  };
}

function numberEnv(name, fallback) {
  const value = Number(__ENV[name] || fallback);
  if (!Number.isInteger(value) || value < 1) {
    throw new Error(`${name} must be a positive integer`);
  }
  return value;
}

function numberValue(name, fallback) {
  const value = Number(__ENV[name] ?? fallback);
  if (!Number.isFinite(value)) {
    throw new Error(`${name} must be a number`);
  }
  return value;
}

function randomPacing(minKey, minFallback, maxKey, maxFallback) {
  const min = Number(pacingConfig[minKey] ?? minFallback);
  const max = Number(pacingConfig[maxKey] ?? maxFallback);
  if (!Number.isFinite(min) || !Number.isFinite(max) || min < 0 || max < min) {
    throw new Error(`${minKey}/${maxKey} must define a valid range`);
  }
  return min + Math.random() * (max - min);
}

function numberListEnv(name, fallback) {
  const raw = __ENV[name];
  if (!raw) {
    return fallback;
  }
  const values = raw.split(',').map((value) => Number(value.trim()));
  if (values.length === 0 || values.some((value) => !Number.isInteger(value) || value < 1)) {
    throw new Error(`${name} must be a comma-separated list of positive integers`);
  }
  return values;
}

function tokenListEnv(name) {
  const raw = __ENV[name];
  if (!raw) {
    return [];
  }
  const values = raw.split(',').map((value) => value.trim()).filter(Boolean);
  if (values.length === 0) {
    throw new Error(`${name} must contain at least one non-empty token`);
  }
  return values;
}

function normalizeBaseURL(value) {
  return value.replace(/\/+$/, '').replace(/\/v1$/, '');
}

function mockChannelTokens(id, fallback) {
  const channel = (mockConfig.channels || []).find((item) => item.id === id);
  return channel && Number.isInteger(channel.tokens) && channel.tokens >= 0 ? channel.tokens : fallback;
}
