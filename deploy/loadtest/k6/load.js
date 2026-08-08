import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const profile = __ENV.LOAD_PROFILE || 'smoke';
const baseURL = __ENV.BASE_URL || 'http://new-api:3000';
const userCount = numberEnv('LOADTEST_USERS', 1000);
const smokeDuration = __ENV.SMOKE_DURATION || '1m';
const smokeVUs = numberEnv('SMOKE_VUS', 10);
const burstMaxDuration = __ENV.BURST_MAX_DURATION || '5m';
const capacityRates = numberListEnv('CAPACITY_RATES', [100, 200, 300, 400, 500, 600, 700, 800]);
const capacityRampDuration = __ENV.CAPACITY_RAMP_DURATION || '30s';
const capacityStageDuration = __ENV.CAPACITY_STAGE_DURATION || '2m';
const capacityPreallocatedVUs = numberEnv('CAPACITY_PREALLOCATED_VUS', 500);
const capacityMaxVUs = numberEnv('CAPACITY_MAX_VUS', 2000);

const applicationErrors = new Rate('new_api_application_errors');
const timeToFirstByte = new Trend('new_api_time_to_first_byte', true);
const completedRequests = new Counter('new_api_completed_requests');
const promptTokens = new Counter('new_api_prompt_tokens');
const completionTokens = new Counter('new_api_completion_tokens');
const totalTokens = new Counter('new_api_total_tokens');
const usageMissing = new Rate('new_api_usage_missing');

const commonThresholds = {
  http_req_failed: ['rate<0.01'],
  new_api_application_errors: ['rate<0.01'],
  http_req_duration: ['p(95)<2000', 'p(99)<5000'],
  new_api_time_to_first_byte: ['p(95)<1000'],
};

export const options = buildOptions(profile);

export function chat() {
  makeChatRequest(false);
  sleep(0.2 + Math.random() * 0.8);
}

export function stream() {
  makeChatRequest(true);
  sleep(0.1 + Math.random() * 0.4);
}

export function modelList() {
  const response = http.get(`${baseURL}/v1/models`, {
    headers: headersForVU(),
    tags: { endpoint: 'models', stream: 'false', profile },
    timeout: '30s',
  });
  recordResponse(response, 'model list');
  sleep(0.5 + Math.random());
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
      model: 'gpt-3.5-turbo',
      messages: [{ role: 'user', content: 'Return a deterministic load-test response.' }],
      stream: streaming,
      stream_options: streaming ? { include_usage: true } : undefined,
      max_tokens: 32,
      temperature: 0,
    }),
    {
      headers: headersForVU(),
      tags: { endpoint: 'chat_completions', stream: String(streaming), profile },
      timeout: streaming ? '2m' : '30s',
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
  const valid = check(response, {
    [`${name}: status is 200`]: (result) => result.status === 200,
    [`${name}: no gateway error`]: (result) => !result.body || !result.body.includes('"error"'),
  });
  applicationErrors.add(!valid, { profile });
  completedRequests.add(1, { profile });
}

function headersForVU() {
  const tokenNumber = ((__VU - 1) % userCount) + 1;
  return {
    Authorization: `Bearer sk-loadtest${String(tokenNumber).padStart(5, '0')}`,
    'Content-Type': 'application/json',
    'X-Load-Test-ID': `${profile}-${__VU}-${__ITER}`,
  };
}

function buildOptions(selectedProfile) {
  const profiles = {
    smoke: {
      scenarios: {
        smoke: { executor: 'constant-vus', exec: 'chat', vus: smokeVUs, duration: smokeDuration, gracefulStop: '30s' },
      },
    },
    step: {
      scenarios: {
        step: {
          executor: 'ramping-vus',
          exec: 'chat',
          startVUs: 0,
          stages: [
            { target: 50, duration: '2m' }, { target: 50, duration: '5m' },
            { target: 100, duration: '1m' }, { target: 100, duration: '5m' },
            { target: 250, duration: '1m' }, { target: 250, duration: '5m' },
            { target: 500, duration: '1m' }, { target: 500, duration: '5m' },
            { target: 750, duration: '1m' }, { target: 750, duration: '5m' },
            { target: 1000, duration: '1m' }, { target: 1000, duration: '10m' },
            { target: 0, duration: '2m' },
          ],
          gracefulRampDown: '30s',
        },
      },
    },
    steady: {
      scenarios: {
        steady: { executor: 'constant-vus', exec: 'chat', vus: 1000, duration: '30m', gracefulStop: '2m' },
      },
    },
    spike: {
      scenarios: {
        spike: {
          executor: 'ramping-vus',
          exec: 'chat',
          startVUs: 0,
          stages: [
            { target: 50, duration: '1m' },
            { target: 1000, duration: '10s' },
            { target: 1000, duration: '5m' },
            { target: 0, duration: '30s' },
          ],
          gracefulRampDown: '30s',
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
          maxDuration: burstMaxDuration,
        },
      },
    },
    stream: {
      scenarios: {
        stream: { executor: 'constant-vus', exec: 'stream', vus: 1000, duration: '15m', gracefulStop: '2m' },
      },
      thresholds: {
        ...commonThresholds,
        http_req_duration: ['p(95)<5000', 'p(99)<10000'],
      },
    },
    mixed: {
      scenarios: {
        stream: { executor: 'constant-vus', exec: 'stream', vus: 700, duration: '20m', gracefulStop: '2m' },
        chat: { executor: 'constant-vus', exec: 'chat', vus: 200, duration: '20m', gracefulStop: '2m' },
        models: { executor: 'constant-vus', exec: 'modelList', vus: 100, duration: '20m', gracefulStop: '30s' },
      },
      thresholds: {
        ...commonThresholds,
        'http_req_duration{stream:true}': ['p(95)<5000'],
      },
    },
    soak: {
      scenarios: {
        soak: { executor: 'constant-vus', exec: 'chat', vus: 500, duration: '2h', gracefulStop: '2m' },
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
            { target: rate, duration: index === 0 ? '10s' : capacityRampDuration },
            { target: rate, duration: capacityStageDuration },
          ]).concat([{ target: 0, duration: '30s' }]),
          gracefulStop: '1m',
        },
      },
      thresholds: {
        ...commonThresholds,
        dropped_iterations: ['count==0'],
        new_api_usage_missing: ['rate<0.001'],
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
