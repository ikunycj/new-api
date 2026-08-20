import { readFileSync, writeFileSync } from 'node:fs';

const baseURL = (process.env.BASE_URL || 'https://alltokenapi.com').replace(/\/+$/, '').replace(/\/v1$/, '');
const model = process.env.LOADTEST_MODEL || 'claude-opus-4-8';
const tokenFile = process.env.LOADTEST_TOKEN_FILE;
const vus = numberEnv('TRAJECTORY_VUS', 50);
const turns = numberEnv('TRAJECTORY_TURNS', 16);
const timeoutMs = numberEnv('TRAJECTORY_TIMEOUT_MS', 120_000);
const rampSeconds = numberEnv('TRAJECTORY_RAMP_SECONDS', 0);
const holdSeconds = numberEnv('TRAJECTORY_HOLD_SECONDS', 0);
const batchSize = numberEnv('TRAJECTORY_BATCH_SIZE', 10);
const runID = process.env.LOADTEST_RUN_ID || `trajectory-native-${Date.now()}`;
const resultFile = process.env.LOADTEST_RESULT_FILE;

if (!tokenFile) throw new Error('LOADTEST_TOKEN_FILE is required');

const tokens = readFileSync(tokenFile, 'utf8')
  .split(/\r?\n/)
  .map((line) => line.trim().split(/\s+/).pop())
  .filter(Boolean);

if (tokens.length < vus) {
  throw new Error(`token file has ${tokens.length} tokens, but ${vus} VUs were requested`);
}

const toolSchemas = [
  {
    type: 'function',
    function: {
      name: 'read_fixture',
      description: 'Read a deterministic fixture. This tool has no external side effects.',
      parameters: { type: 'object', properties: { path: { type: 'string' } }, required: ['path'], additionalProperties: false },
    },
  },
  {
    type: 'function',
    function: {
      name: 'inspect_fixture',
      description: 'Inspect deterministic metadata. This tool has no external side effects.',
      parameters: { type: 'object', properties: { query: { type: 'string' } }, required: ['query'], additionalProperties: false },
    },
  },
];

const result = {
  run_id: runID,
  target: baseURL,
  model,
  vus,
  turns,
  started_at: new Date().toISOString(),
  ended_at: null,
  sessions_completed: 0,
  workers_started: 0,
  flows_attempted: 0,
  flows_succeeded: 0,
  flows_failed: 0,
  status_counts: {},
  error_counts: {},
  latencies_ms: [],
};

let stopRequested = false;
const testStart = Date.now();
const testEnd = testStart + (rampSeconds + holdSeconds) * 1000;
const workers = [];
const batches = Math.ceil(vus / batchSize);

for (let batch = 0; batch < batches; batch += 1) {
  const targetStart = testStart + (rampSeconds > 0 ? (rampSeconds * 1000 * batch) / batches : 0);
  await sleep(Math.max(0, targetStart - Date.now()));
  const firstVU = batch * batchSize + 1;
  const lastVU = Math.min(vus, firstVU + batchSize - 1);
  for (let vu = firstVU; vu <= lastVU; vu += 1) {
    result.workers_started += 1;
    workers.push(runWorker(vu, tokens[vu - 1], testEnd));
  }
}
await Promise.all(workers);

result.ended_at = new Date().toISOString();
result.success_rate = result.flows_attempted === 0 ? 0 : result.flows_succeeded / result.flows_attempted;
result.latency = summarizeLatency(result.latencies_ms);
delete result.latencies_ms;
const output = `${JSON.stringify(result, null, 2)}\n`;
if (resultFile) writeFileSync(resultFile, output, { mode: 0o600 });
process.stdout.write(output);

async function runSession(vu, token) {
  const messages = [{ role: 'user', content: `${'Synthetic repository task. '.repeat(14)}\n${'Provide a deterministic review of the supplied fixture. '.repeat(24)}` }];
  for (let turn = 0; turn < turns; turn += 1) {
    if (stopRequested) return;
    result.flows_attempted += 1;
    const startedAt = performance.now();
    const response = await requestFlow(vu, turn, token, messages);
    result.latencies_ms.push(performance.now() - startedAt);
    increment(result.status_counts, String(response.status));
    if (response.status !== 200) {
      result.flows_failed += 1;
      increment(result.error_counts, response.error);
      const failureRate = result.flows_failed / result.flows_attempted;
      if (response.status === 401 || response.status === 403 || (result.flows_attempted >= 50 && failureRate >= 0.25)) {
        stopRequested = true;
      }
      return false;
    }
    result.flows_succeeded += 1;
    messages.push({ role: 'assistant', content: 'Deterministic fixture inspection completed.' });
    messages.push({ role: 'user', content: `Fixture result ${turn + 1}: ${'read-only deterministic result '.repeat(10)}` });
    await sleep(200 + Math.random() * 600);
  }
  result.sessions_completed += 1;
  return true;
}

async function runWorker(vu, token, testEnd) {
  while (!stopRequested && Date.now() < testEnd) {
    const completed = await runSession(vu, token);
    if (!completed) await sleep(2000);
  }
}

async function requestFlow(vu, turn, token, messages) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(`${baseURL}/v1/chat/completions`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', 'X-Load-Test-ID': `${runID}-${vu}-${turn}` },
      body: JSON.stringify({ model, messages, tools: toolSchemas, tool_choice: 'auto', stream: false, max_tokens: 128, temperature: 0 }),
      signal: controller.signal,
    });
    if (response.status === 200) {
      await response.arrayBuffer();
      return { status: response.status, error: 'success' };
    }
    let body = {};
    try { body = await response.json(); } catch (_) { /* normalized status is sufficient */ }
    return { status: response.status, error: normalizeError(response.status, body) };
  } catch (error) {
    return { status: 0, error: error?.name === 'AbortError' ? 'timeout' : 'transport_error' };
  } finally {
    clearTimeout(timeout);
  }
}

function normalizeError(status, body) {
  const error = body?.error || body || {};
  const code = error.source_code || error.code || error.type;
  return code ? `${status}:${String(code).slice(0, 120)}` : `http_${status}`;
}

function summarizeLatency(values) {
  if (values.length === 0) return { count: 0 };
  const sorted = [...values].sort((a, b) => a - b);
  return { count: sorted.length, avg: sorted.reduce((sum, value) => sum + value, 0) / sorted.length, p50: percentile(sorted, 0.5), p95: percentile(sorted, 0.95), p99: percentile(sorted, 0.99), max: sorted.at(-1) };
}

function percentile(sorted, quantile) { return sorted[Math.min(sorted.length - 1, Math.ceil(sorted.length * quantile) - 1)]; }
function increment(target, key) { target[key] = (target[key] || 0) + 1; }
function numberEnv(name, fallback) { const value = Number(process.env[name]); return Number.isFinite(value) && value > 0 ? value : fallback; }
function sleep(ms) { return new Promise((resolve) => setTimeout(resolve, ms)); }
