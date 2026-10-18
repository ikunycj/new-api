#!/usr/bin/env python3
"""Multi-turn Claude session load test for prompt-cache stickiness.

Each session is a real conversation: a fixed system context, a history that
grows by one user/assistant pair per turn, a stable metadata.user_id, and the
next turn sent only after the previous reply returns. Breakpoints sit on the
system block and the latest user message, like Claude Code.

    LOADTEST_TOKENS=sk-xxx python3 multiturn.py --base-url https://alltokenapi.com \
        --model claude-haiku-4-5-20251001 --sessions 20 --rpm 200 --duration 3m
"""

import argparse
import asyncio
import hashlib
import json
import os
import random
import time
import uuid
from collections import defaultdict

import httpx

from loadtest import build_context, parse_duration

QUESTIONS = (
    "Which town in the story sounds the most peaceful, and why?",
    "Pick one scene and describe what the people there might have eaten that evening.",
    "What would a visitor notice first when arriving in one of these towns?",
    "Summarise the mood of the story in two sentences.",
    "Which activity from the story would you most like to join?",
    "Invent a short conversation between two neighbours from one of the towns.",
    "How might the same town look in winter?",
    "What small detail from your previous answer could be expanded into a new scene?",
)


def cache_tokens(usage):
    read = usage.get("cache_read_input_tokens") or 0
    write = usage.get("cache_creation_input_tokens") or 0
    return usage.get("input_tokens") or 0, read, write


async def run_session(args, client, context, stats, deadline, index):
    interval = args.sessions * 60.0 / args.rpm
    # 错开各会话的起点，避免所有会话同一时刻冷启动
    next_at = time.monotonic() + random.random() * interval
    conversation = 0
    while time.monotonic() < deadline:
        conversation += 1
        seed = f"{args.context_seed}-s{index}-c{conversation}"
        user_id = (f"user_{hashlib.sha256(seed.encode()).hexdigest()}_account__session_"
                   f"{uuid.uuid5(uuid.NAMESPACE_URL, seed)}")
        history = []
        prev_prompt = 0
        for turn in range(1, args.max_turns + 1):
            await asyncio.sleep(max(0.0, next_at - time.monotonic()))
            if time.monotonic() >= deadline:
                return
            next_at += interval
            question = f"(turn {turn}) {QUESTIONS[(index + turn) % len(QUESTIONS)]}"
            messages = [{**m, "content": [{"type": "text", "text": m["content"]}]} for m in history]
            messages.append({"role": "user", "content": [
                {"type": "text", "text": question, "cache_control": {"type": "ephemeral"}}]})
            body = {
                "model": args.model,
                "max_tokens": args.max_tokens,
                "system": [{"type": "text", "text": context, "cache_control": {"type": "ephemeral"}}],
                "messages": messages,
                "metadata": {"user_id": user_id},
            }
            start = time.monotonic()
            try:
                response = await client.post("/v1/messages", json=body, timeout=args.timeout)
                status = response.status_code
                payload = response.json() if status == 200 else None
                error = None if status == 200 else response.text[:300]
            except (httpx.HTTPError, ValueError) as exc:
                status, payload, error = 0, None, f"{type(exc).__name__}: {exc}"[:300]
            elapsed = time.monotonic() - start
            bucket = int((start - stats["started"]) // 10)
            stats["timeline"][bucket]["requests"] += 1
            stats["latency"].append(elapsed)
            if payload is None or "usage" not in payload:
                stats["errors"][status] += 1
                stats["timeline"][bucket]["errors"] += 1
                if len(stats["error_samples"]) < 5:
                    stats["error_samples"].append(f"[{start - stats['started']:.1f}s] status={status} {error}")
                # 失败的一轮不写入历史，下一轮重新问
                continue
            uncached, read, write = cache_tokens(payload["usage"])
            prompt = uncached + read + write
            # 理想情况：本轮应读到上一轮的整个前缀（system + 之前所有轮次）
            full_hit = turn > 1 and read >= 0.9 * prev_prompt
            row = stats["by_turn"]["1" if turn == 1 else "2+"]
            row["n"] += 1
            row["hit"] += read > 0
            row["full_hit"] += full_hit
            for key, value in (("uncached", uncached), ("read", read), ("write", write)):
                row[key] += value
                stats["timeline"][bucket][key] += value
            stats["timeline"][bucket]["miss"] += read == 0
            text = "".join(block.get("text", "") for block in payload.get("content", []) if block.get("type") == "text")
            history.append({"role": "user", "content": question})
            history.append({"role": "assistant", "content": text or "(empty)"})
            prev_prompt = prompt


async def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="https://alltokenapi.com")
    parser.add_argument("--model", required=True)
    parser.add_argument("--sessions", type=int, default=20)
    parser.add_argument("--rpm", type=float, default=200, help="target total RPM across all sessions")
    parser.add_argument("--duration", default="3m")
    parser.add_argument("--max-turns", type=int, default=20, help="turns before a session starts a new conversation")
    parser.add_argument("--context-tokens", type=int, default=6000)
    parser.add_argument("--context-seed", default=f"mt-{int(time.time())}")
    parser.add_argument("--max-tokens", type=int, default=256)
    parser.add_argument("--timeout", type=float, default=120)
    parser.add_argument("--summary-out")
    args = parser.parse_args()
    token = os.environ["LOADTEST_TOKENS"].split(",")[0].strip()

    context = build_context(args.context_tokens, args.context_seed)
    stats = {
        "started": time.monotonic(),
        "timeline": defaultdict(lambda: defaultdict(int)),
        "by_turn": defaultdict(lambda: defaultdict(int)),
        "errors": defaultdict(int),
        "error_samples": [],
        "latency": [],
    }
    deadline = stats["started"] + parse_duration(args.duration)
    headers = {"Authorization": f"Bearer {token}", "x-api-key": token, "anthropic-version": "2023-06-01"}
    limits = httpx.Limits(max_connections=args.sessions * 2)
    async with httpx.AsyncClient(base_url=args.base_url.rstrip("/"), headers=headers, limits=limits) as client:
        await asyncio.gather(*(run_session(args, client, context, stats, deadline, i) for i in range(1, args.sessions + 1)))
    elapsed = time.monotonic() - stats["started"]

    print(f"== multiturn against {args.base_url} model={args.model} sessions={args.sessions} "
          f"target_rpm={args.rpm:g} in {elapsed:.1f}s")
    for bucket in sorted(stats["timeline"]):
        t = stats["timeline"][bucket]
        total = t["uncached"] + t["read"] + t["write"]
        print(f"[{(bucket + 1) * 10:>5}s] requests=+{t['requests']} errors=+{t['errors']} miss=+{t['miss']} "
              f"token_hit={t['read'] / total * 100 if total else 0:.1f}%")
    requests = len(stats["latency"])
    latency = sorted(stats["latency"])
    print(f"  requests       {requests} ({requests / elapsed * 60:.0f} RPM actual) errors={dict(stats['errors'])}")
    if latency:
        print(f"  latency s      p50={latency[len(latency) // 2]:.2f} p95={latency[int(len(latency) * 0.95)]:.2f} "
              f"max={latency[-1]:.2f}")
    summary = {"args": vars(args), "elapsed": elapsed, "requests": requests, "errors": dict(stats["errors"]), "by_turn": {}}
    for name in ("1", "2+"):
        row = stats["by_turn"][name]
        if not row["n"]:
            continue
        total = row["uncached"] + row["read"] + row["write"]
        line = (f"  turn {name:<3} n={row['n']} any_hit={row['hit'] / row['n'] * 100:.1f}% "
                f"token_hit={row['read'] / total * 100:.1f}% read={row['read']} write={row['write']} "
                f"uncached={row['uncached']}")
        if name == "2+":
            line += f" full_prefix_hit={row['full_hit'] / row['n'] * 100:.1f}%"
        print(line)
        summary["by_turn"][name] = dict(row)
    all_rows = [stats["by_turn"][n] for n in ("1", "2+")]
    read = sum(r["read"] for r in all_rows)
    total = sum(r["uncached"] + r["read"] + r["write"] for r in all_rows)
    print(f"  overall        token_hit={read / total * 100 if total else 0:.2f}% read={read} "
          f"write={sum(r['write'] for r in all_rows)}")
    for sample in stats["error_samples"]:
        print(f"  error sample   {sample}")
    if args.summary_out:
        with open(args.summary_out, "w") as handle:
            json.dump(summary, handle, indent=2)


if __name__ == "__main__":
    asyncio.run(main())
