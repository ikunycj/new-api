#!/usr/bin/env python3
"""Compare prompt-cache hits for three request layouts built from a real task prompt.

The prompt file is split into a fixed instruction part and a per-task payload: by default
the payload starts at the first line beginning with `{"case_id"`. Each request uses a
different payload, the way a batch job reviews many cases with the same instructions.

  A  identical body every time (cache upper bound)
  B  instruction + payload concatenated into one user message
  C  instruction as a system message, payload as the user message

Some upstreams (e.g. some Azure-backed relays) match cache per whole message,
so B rewrites the full prompt on every request while C keeps hitting the instruction part.

    pip install httpx
    python3 cache_layout_demo.py --base-url https://alltokenapi.com --api-key sk-xxx \\
        --prompt-file detect-financial_task_auto_1880.md --requests 8
"""

import argparse
import asyncio
import json
import os
import re
import sys
import time
from pathlib import Path

import httpx

LAYOUTS = {
    "A": "identical body",
    "B": "instruction + payload in one user message",
    "C": "instruction as system message, payload as user message",
}


def split_prompt(text, marker):
    index = text.find(marker)
    if index < 0:
        raise SystemExit(f"payload marker {marker!r} not found; pass --payload-marker or --payload-file")
    return text[:index].rstrip(), text[index:].strip()


def payload_variants(files, marker, count):
    payloads = []
    for path in files:
        _, payload = split_prompt(Path(path).read_text(encoding="utf-8"), marker)
        payloads.append(payload)
    if len(payloads) >= count:
        return payloads[:count]
    # 只有一份任务数据时，改写 case_id 生成不同的任务；位置靠前，保证每个请求的 payload 都不同
    base = payloads[0]
    variants = list(payloads)
    while len(variants) < count:
        tag = f"_v{len(variants)}"
        changed, replaced = re.subn(r'("case_id"\s*:\s*")([^"]*)(")', lambda m: m.group(1) + m.group(2) + tag + m.group(3), base, count=1)
        variants.append(changed if replaced else f"[{tag}] {base}")
    return variants


def build_messages(layout, instruction, payload, first_payload):
    if layout == "A":
        return [{"role": "user", "content": f"{instruction}\n\n{first_payload}"}]
    if layout == "B":
        return [{"role": "user", "content": f"{instruction}\n\n{payload}"}]
    return [{"role": "system", "content": instruction}, {"role": "user", "content": payload}]


def read_usage(api, usage):
    usage = usage or {}
    if api == "responses":
        details = usage.get("input_tokens_details") or {}
        return usage.get("input_tokens"), details.get("cached_tokens") or 0, details.get("cache_write_tokens") or 0
    details = usage.get("prompt_tokens_details") or {}
    return usage.get("prompt_tokens"), details.get("cached_tokens") or 0, details.get("cache_write_tokens") or 0


async def send(client, args, layout, index, messages, cache_key):
    if args.api == "responses":
        url = f"{args.base_url}/v1/responses"
        body = {"model": args.model, "input": messages, "max_output_tokens": args.max_tokens}
    else:
        url = f"{args.base_url}/v1/chat/completions"
        body = {"model": args.model, "messages": messages, args.max_tokens_field: args.max_tokens}
    if cache_key:
        body["prompt_cache_key"] = cache_key
    start = time.perf_counter()
    try:
        response = await client.post(url, json=body)
        elapsed = time.perf_counter() - start
        request_id = response.headers.get("x-oneapi-request-id")
        if response.status_code != 200:
            return {"layout": layout, "index": index, "status": response.status_code, "elapsed": elapsed,
                    "request_id": request_id, "error": response.text[:200]}
        prompt, cached, written = read_usage(args.api, response.json().get("usage"))
        return {"layout": layout, "index": index, "status": 200, "elapsed": elapsed, "request_id": request_id,
                "prompt": prompt, "cached": cached, "write": written}
    except httpx.HTTPError as error:
        return {"layout": layout, "index": index, "status": 0, "elapsed": time.perf_counter() - start,
                "request_id": None, "error": f"{type(error).__name__}: {error}"}


async def run_layout(client, args, layout, instruction, payloads, run_tag):
    cache_key = f"cache-demo-{run_tag}-{layout}" if args.prompt_cache_key else None
    semaphore = asyncio.Semaphore(args.concurrency)
    results = [None] * args.requests

    async def one(index):
        async with semaphore:
            messages = build_messages(layout, instruction, payloads[index], payloads[0])
            results[index] = await send(client, args, layout, index, messages, cache_key)
            row = results[index]
            if row["status"] == 200:
                print(f"  {layout} #{index + 1:<3} {row['elapsed']:5.1f}s prompt={row['prompt']} "
                      f"cached={row['cached']} write={row['write']} rid={row['request_id']}", flush=True)
            else:
                print(f"  {layout} #{index + 1:<3} status={row['status']} rid={row['request_id']} {row['error']}", flush=True)

    await asyncio.gather(*(one(index) for index in range(args.requests)))
    return results


def summarize(layout, rows):
    ok = [row for row in rows if row["status"] == 200]
    # 第一条是冷启动，命中率按之后的请求计算
    warm = [row for row in ok if row["index"] > 0]
    hits = sum(1 for row in warm if row["cached"] > 0)
    prompt = sum(row["prompt"] or 0 for row in ok)
    cached = sum(row["cached"] for row in ok)
    written = sum(row["write"] for row in ok)
    return {
        "layout": layout,
        "description": LAYOUTS[layout],
        "requests": len(rows),
        "success": len(ok),
        "warm_hit_requests": hits,
        "warm_requests": len(warm),
        "warm_hit_rate": hits / len(warm) if warm else 0.0,
        "prompt_tokens": prompt,
        "cached_tokens": cached,
        "cache_write_tokens": written,
        "token_hit_rate": cached / prompt if prompt else 0.0,
    }


async def main_async(args):
    files = args.payload_file or [args.prompt_file]
    instruction, _ = split_prompt(Path(args.prompt_file).read_text(encoding="utf-8"), args.payload_marker)
    payloads = payload_variants(files, args.payload_marker, args.requests)
    # 每次运行给指令加一个唯一标记，避免读到上一次运行留下的缓存
    run_tag = args.run_tag or time.strftime("%Y%m%d%H%M%S")
    instruction = f"[cache-demo run {run_tag}]\n{instruction}"

    print(f"target={args.base_url} api={args.api} model={args.model} requests/layout={args.requests} "
          f"concurrency={args.concurrency} prompt_cache_key={'on' if args.prompt_cache_key else 'off'}")
    print(f"instruction={len(instruction)} chars, payload≈{len(payloads[0])} chars, distinct payloads={len(set(payloads))}")

    headers = {"Authorization": f"Bearer {args.api_key}", "Content-Type": "application/json"}
    summaries, all_rows = [], []
    async with httpx.AsyncClient(headers=headers, timeout=args.timeout) as client:
        for layout in args.layouts:
            print(f"\n== {layout}: {LAYOUTS[layout]}")
            rows = await run_layout(client, args, layout, instruction, payloads, run_tag)
            all_rows.extend(rows)
            summaries.append(summarize(layout, rows))

    print("\n--- summary (hit rate excludes the first, cold request) ---")
    print(f"  {'layout':<7}{'ok':>6}{'warm hit':>12}{'hit rate':>10}{'token hit':>11}{'cached':>10}{'written':>10}")
    for s in summaries:
        print(f"  {s['layout']:<7}{s['success']:>3}/{s['requests']:<3}{s['warm_hit_requests']:>6}/{s['warm_requests']:<5}"
              f"{s['warm_hit_rate']:>9.0%}{s['token_hit_rate']:>11.0%}{s['cached_tokens']:>10}{s['cache_write_tokens']:>10}")
    if args.json_out:
        Path(args.json_out).parent.mkdir(parents=True, exist_ok=True)
        Path(args.json_out).write_text(json.dumps({"summary": summaries, "requests": all_rows}, indent=2, ensure_ascii=False))
        print(f"\nwritten to {args.json_out}")


def parse_args():
    env = os.environ.get
    parser = argparse.ArgumentParser(description="Compare prompt-cache hits for A/B/C request layouts.")
    parser.add_argument("--base-url", default=env("BASE_URL", "https://alltokenapi.com"))
    parser.add_argument("--api-key", default=env("API_KEY") or env("LOADTEST_TOKENS"), help="env API_KEY")
    parser.add_argument("--model", default="gpt-6-luna")
    parser.add_argument("--prompt-file", required=True, help="file with the fixed instruction followed by one task payload")
    parser.add_argument("--payload-file", action="append",
                        help="more task files (repeatable); their payloads are used in order instead of synthetic variants")
    parser.add_argument("--payload-marker", default='{"case_id"', help="text where the per-task payload starts")
    parser.add_argument("--layouts", default="ABC", help="which layouts to run, e.g. BC")
    parser.add_argument("--requests", type=int, default=8, help="requests per layout")
    parser.add_argument("--concurrency", type=int, default=1, help="1 = strictly sequential")
    parser.add_argument("--api", choices=("chat", "responses"), default="chat")
    parser.add_argument("--max-tokens", type=int, default=32, help="small output keeps the demo cheap")
    parser.add_argument("--max-tokens-field", choices=("max_tokens", "max_completion_tokens"), default="max_completion_tokens")
    parser.add_argument("--prompt-cache-key", action=argparse.BooleanOptionalAction, default=True)
    parser.add_argument("--run-tag", default=None, help="reuse a tag to test a warm cache; default is a fresh tag")
    parser.add_argument("--timeout", type=float, default=180)
    parser.add_argument("--json-out", default=None)
    args = parser.parse_args()
    if not args.api_key:
        parser.error("provide --api-key or set API_KEY")
    args.base_url = re.sub(r"/v1/?$", "", args.base_url.rstrip("/"))
    args.layouts = [layout for layout in args.layouts.upper() if layout in LAYOUTS]
    return args


if __name__ == "__main__":
    try:
        asyncio.run(main_async(parse_args()))
    except KeyboardInterrupt:
        sys.exit(130)
