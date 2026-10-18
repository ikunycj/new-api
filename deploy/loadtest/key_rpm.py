#!/usr/bin/env python3
"""按 key 分别打固定 RPM，对比每个 key 的成功率、限流和延迟。

每个 key 独立按开环节奏发请求（第 i 个请求在 start + i*60/rpm 秒发出，不等上一个返回），
所以上游变慢不会降低实际发送速率，429 和超时会如实暴露出来。

keys 文件每行一个 key，格式：

    name  sk-xxx  [rpm]  [model]

rpm、model 省略时用命令行的 --rpm、--model。# 开头的行是注释。keys 文件不要提交到仓库
（keys*.txt 已在 .gitignore 中）。

    pip install httpx
    python3 key_rpm.py --keys-file keys.txt --model gpt-4o-mini --rpm 60 --duration 2m
    python3 key_rpm.py --keys-file keys.txt --api-format claude --model claude-sonnet-4-6 --stream

打的是线上网关和真实上游，会产生真实费用；非本地地址需要 --yes 确认。
"""

import argparse
import asyncio
import json
import re
import sys
import time
from collections import Counter
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import urlparse

import httpx

DURATION_PART = re.compile(r"(\d+(?:\.\d+)?)(ms|s|m|h)")
DURATION_UNITS = {"ms": 0.001, "s": 1, "m": 60, "h": 3600}


def parse_duration(text: str) -> float:
    parts = DURATION_PART.findall(text)
    if not parts or "".join(n + u for n, u in parts) != text:
        raise argparse.ArgumentTypeError(f"invalid duration: {text!r}")
    return sum(float(n) * DURATION_UNITS[u] for n, u in parts)


def percentile(values, p):
    if not values:
        return 0.0
    ordered = sorted(values)
    rank = (len(ordered) - 1) * p / 100
    low = int(rank)
    high = min(low + 1, len(ordered) - 1)
    return ordered[low] + (ordered[high] - ordered[low]) * (rank - low)


@dataclass
class KeyStats:
    name: str
    key: str
    rpm: float
    model: str
    sent: int = 0
    ok: int = 0
    dropped: int = 0
    inflight: int = 0
    statuses: Counter = field(default_factory=Counter)
    latencies: list = field(default_factory=list)
    ttfb: list = field(default_factory=list)
    ok_per_minute: Counter = field(default_factory=Counter)
    input_tokens: int = 0
    output_tokens: int = 0
    error_samples: list = field(default_factory=list)


def load_keys(path: Path, default_rpm: float, default_model: str):
    keys = []
    for number, line in enumerate(path.read_text().splitlines(), 1):
        parts = line.split("#", 1)[0].split()
        if not parts:
            continue
        if len(parts) == 1:
            parts = [f"key{len(keys) + 1}", parts[0]]
        name, key = parts[0], parts[1]
        rpm = float(parts[2]) if len(parts) > 2 else default_rpm
        model = parts[3] if len(parts) > 3 else default_model
        if rpm <= 0 or not model:
            sys.exit(f"{path}:{number}: rpm must be > 0 and model must be set (use --model)")
        keys.append(KeyStats(name=name, key=key, rpm=rpm, model=model))
    if not keys:
        sys.exit(f"{path}: no keys")
    return keys


def mask(key: str) -> str:
    return key[:6] + "..." + key[-4:] if len(key) > 12 else "***"


def collect_usage(stats: KeyStats, payload: dict):
    usage = payload.get("usage") or (payload.get("message") or {}).get("usage")
    if not isinstance(usage, dict):
        return
    # OpenAI: prompt/completion_tokens；Claude: input/output_tokens（流式时分散在 message_start 和 message_delta）
    stats.input_tokens += usage.get("prompt_tokens") or usage.get("input_tokens") or 0
    stats.output_tokens += usage.get("completion_tokens") or usage.get("output_tokens") or 0


async def send_one(client: httpx.AsyncClient, args, stats: KeyStats, seq: int, started: float):
    if args.api_format == "claude":
        url = args.base_url + "/v1/messages"
        headers = {"x-api-key": stats.key, "anthropic-version": "2023-06-01"}
    else:
        url = args.base_url + "/v1/chat/completions"
        headers = {"Authorization": f"Bearer {stats.key}"}
    headers["X-Load-Test-ID"] = f"key-rpm-{stats.name}-{seq}"
    body = {
        "model": stats.model,
        "messages": [{"role": "user", "content": f"{args.message} (#{stats.name}-{seq})"}],
        "max_tokens": args.max_tokens,
        "stream": args.stream,
    }
    if args.stream and args.api_format == "openai":
        body["stream_options"] = {"include_usage": True}

    stats.sent += 1
    stats.inflight += 1
    t0 = time.monotonic()
    status = 0
    error_text = ""
    try:
        async with client.stream("POST", url, headers=headers, json=body) as resp:
            status = resp.status_code
            if status != 200:
                error_text = (await resp.aread()).decode("utf-8", "replace")[:300]
            elif not args.stream:
                stats.ttfb.append((time.monotonic() - t0) * 1000)
                collect_usage(stats, json.loads(await resp.aread()))
            else:
                first_chunk = True
                async for line in resp.aiter_lines():
                    if not line.startswith("data:"):
                        continue
                    data = line[5:].strip()
                    if data == "[DONE]":
                        break
                    payload = json.loads(data)
                    if "error" in payload:
                        # 流内错误（200 后上游中断）单独记为 -1
                        status, error_text = -1, data[:300]
                        break
                    if first_chunk:
                        stats.ttfb.append((time.monotonic() - t0) * 1000)
                        first_chunk = False
                    collect_usage(stats, payload)
    except Exception as exc:  # 超时、连接断开、流中断都记成 status 0
        status, error_text = 0, f"{type(exc).__name__}: {exc}"[:300]
    finally:
        stats.inflight -= 1

    elapsed_ms = (time.monotonic() - t0) * 1000
    stats.statuses[status] += 1
    if status == 200:
        stats.ok += 1
        stats.latencies.append(elapsed_ms)
        stats.ok_per_minute[int((time.monotonic() - started) // 60)] += 1
    elif len(stats.error_samples) < 5:
        stats.error_samples.append(f"{status} {error_text}")


async def drive_key(client, args, stats: KeyStats, started: float, deadline: float, tasks: set):
    interval = 60 / stats.rpm
    seq = 0
    while True:
        due = started + seq * interval
        if due >= deadline:
            return
        await asyncio.sleep(max(0.0, due - time.monotonic()))
        seq += 1
        if stats.inflight >= args.max_inflight:
            stats.dropped += 1
            continue
        task = asyncio.create_task(send_one(client, args, stats, seq, started))
        tasks.add(task)
        task.add_done_callback(tasks.discard)


async def report_progress(keys, started: float, interval: float):
    while True:
        await asyncio.sleep(interval)
        elapsed = time.monotonic() - started
        line = "  ".join(
            f"{k.name}: sent={k.sent} ok={k.ok} 429={k.statuses[429]} err={k.sent - k.ok - k.statuses[429] - k.inflight} inflight={k.inflight}"
            for k in keys
        )
        print(f"[{elapsed:6.0f}s] {line}", flush=True)


def print_summary(keys, duration: float):
    minutes = duration / 60
    print()
    header = f"{'key':<12} {'model':<22} {'目标RPM':>7} {'实发RPM':>7} {'成功RPM':>7} {'成功率':>7} {'429':>5} {'其他错':>6} {'丢弃':>5} {'p50ms':>7} {'p95ms':>7} {'p99ms':>7} {'首字p50':>7} {'输入tok':>9} {'输出tok':>9}"
    print(header)
    print("-" * len(header))
    for k in keys:
        other = k.sent - k.ok - k.statuses[429]
        rate = k.ok / k.sent * 100 if k.sent else 0
        print(
            f"{k.name:<12} {k.model[:22]:<22} {k.rpm:>7.0f} {k.sent / minutes:>7.1f} {k.ok / minutes:>7.1f} {rate:>6.1f}% "
            f"{k.statuses[429]:>5} {other:>6} {k.dropped:>5} {percentile(k.latencies, 50):>7.0f} "
            f"{percentile(k.latencies, 95):>7.0f} {percentile(k.latencies, 99):>7.0f} {percentile(k.ttfb, 50):>7.0f} "
            f"{k.input_tokens:>9} {k.output_tokens:>9}"
        )
    for k in keys:
        per_minute = " ".join(f"{k.ok_per_minute[m]}" for m in range(int(minutes + 0.999)))
        print(f"\n[{k.name}] key={mask(k.key)} 状态码={dict(k.statuses)} 每分钟成功数=[{per_minute}]")
        for sample in k.error_samples:
            print(f"  error: {sample}")


async def main_async(args):
    keys = load_keys(Path(args.keys_file), args.rpm, args.model)
    host = urlparse(args.base_url).hostname or ""
    if host not in ("127.0.0.1", "localhost") and not args.yes:
        total = sum(k.rpm for k in keys) * args.duration / 60
        sys.exit(f"目标 {args.base_url} 不是本地地址，预计发送约 {total:.0f} 个真实请求。确认后加 --yes 再跑。")

    print(f"target={args.base_url} format={args.api_format} stream={args.stream} duration={args.duration:.0f}s")
    for k in keys:
        print(f"  {k.name:<12} key={mask(k.key)} rpm={k.rpm:g} model={k.model}")

    limits = httpx.Limits(max_connections=None, max_keepalive_connections=200)
    timeout = httpx.Timeout(args.timeout, connect=10)
    tasks: set = set()
    async with httpx.AsyncClient(limits=limits, timeout=timeout, verify=not args.insecure) as client:
        started = time.monotonic()
        deadline = started + args.duration
        progress = asyncio.create_task(report_progress(keys, started, 10))
        await asyncio.gather(*(drive_key(client, args, k, started, deadline, tasks) for k in keys))
        if tasks:
            print(f"发送结束，等待 {len(tasks)} 个在途请求返回...", flush=True)
            await asyncio.gather(*tasks, return_exceptions=True)
        progress.cancel()

    print_summary(keys, args.duration)
    if args.summary_out:
        result = {
            "base_url": args.base_url,
            "duration_seconds": args.duration,
            "keys": [
                {
                    "name": k.name, "key": mask(k.key), "model": k.model, "target_rpm": k.rpm,
                    "sent": k.sent, "ok": k.ok, "dropped": k.dropped,
                    "statuses": {str(s): c for s, c in k.statuses.items()},
                    "latency_ms": {p: percentile(k.latencies, int(p[1:])) for p in ("p50", "p95", "p99")},
                    "ttfb_p50_ms": percentile(k.ttfb, 50),
                    "ok_per_minute": [k.ok_per_minute[m] for m in sorted(k.ok_per_minute)],
                    "input_tokens": k.input_tokens, "output_tokens": k.output_tokens,
                    "error_samples": k.error_samples,
                }
                for k in keys
            ],
        }
        Path(args.summary_out).write_text(json.dumps(result, ensure_ascii=False, indent=2))
        print(f"\nsummary 写入 {args.summary_out}")


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--keys-file", required=True, help="每行 name key [rpm] [model]")
    parser.add_argument("--base-url", default="https://alltokenapi.com")
    parser.add_argument("--model", default="", help="keys 文件未指定 model 时使用")
    parser.add_argument("--rpm", type=float, default=60, help="keys 文件未指定 rpm 时使用（每个 key 各自的 RPM）")
    parser.add_argument("--duration", type=parse_duration, default=parse_duration("2m"), help="例如 30s、2m")
    parser.add_argument("--api-format", choices=("openai", "claude"), default="openai")
    parser.add_argument("--stream", action="store_true")
    parser.add_argument("--max-tokens", type=int, default=32)
    parser.add_argument("--message", default="Reply with one short sentence.")
    parser.add_argument("--timeout", type=float, default=120, help="单请求超时秒数")
    parser.add_argument("--max-inflight", type=int, default=500, help="单个 key 在途请求上限，超过的请求记为丢弃")
    parser.add_argument("--summary-out", default=None)
    parser.add_argument("--insecure", action="store_true", help="跳过 TLS 证书校验")
    parser.add_argument("--yes", action="store_true", help="确认对非本地地址发真实请求")
    args = parser.parse_args()
    args.base_url = args.base_url.rstrip("/").removesuffix("/v1")
    try:
        asyncio.run(main_async(args))
    except KeyboardInterrupt:
        print("\ninterrupted")


if __name__ == "__main__":
    main()
