#!/usr/bin/env python3
"""Probe reasoning_items / encrypted_content on /v1/responses, then test turn-2 decryption.

Turn 1: POST /v1/responses with store=false, include=["reasoning.encrypted_content"].
        Records whether output has reasoning items and whether they carry encrypted_content.
Turn 2: Sends turn-1's output items (reasoning + message) back as input.
        The upstream must decrypt encrypted_content; if turn 1 and turn 2 land on
        channels/accounts that do not share a key, this fails
        (typically 400 invalid_encrypted_content, surfaced as 503 by a gateway).

Rounds run concurrently so routing spreads across every channel serving the model.
A mix of turn-2 successes and failures means more than one channel is in rotation.

  LOADTEST_TOKENS=sk-xxx python3 reasoning_probe.py \\
      --base-url https://upstream.example.com --insecure \\
      --model gpt-6-luna --rounds 20 --concurrency 10 \\
      --out results/reasoning-probe.json
"""

import argparse
import asyncio
import collections
import json
import os
import sys
import time

import httpx

DECRYPT_MARKERS = (
    "encrypt", "decrypt", "could not be verified", "invalid_encrypted_content"
)
REQUEST_ID_HEADERS = ("x-oneapi-request-id", "x-request-id", "request-id")


def get_request_id(resp):
    for h in REQUEST_ID_HEADERS:
        v = resp.headers.get(h)
        if v:
            return v
    return None


def inspect_output(body):
    output = body.get("output") or []
    reasoning = [o for o in output if o.get("type") == "reasoning"]
    encrypted = [o for o in reasoning if o.get("encrypted_content")]
    usage = body.get("usage") or {}
    return {
        "output_types": [o.get("type") for o in output],
        "reasoning_items": len(reasoning),
        "encrypted_items": len(encrypted),
        "encrypted_len": sum(len(o["encrypted_content"]) for o in encrypted),
        "has_summary": any(o.get("summary") for o in reasoning),
        "reasoning_tokens": (usage.get("output_tokens_details") or {}).get(
            "reasoning_tokens"
        ),
    }


def is_decrypt_error(text):
    t = (text or "").lower()
    return any(m in t for m in DECRYPT_MARKERS)


async def post_responses(client, base_url, payload):
    t0 = time.perf_counter()
    try:
        resp = await client.post(f"{base_url}/v1/responses", json=payload)
    except httpx.HTTPError as e:
        return {
            "status": 0,
            "error": f"{type(e).__name__}: {e}",
            "ms": (time.perf_counter() - t0) * 1000,
        }
    res = {
        "status": resp.status_code,
        "request_id": get_request_id(resp),
        "ms": (time.perf_counter() - t0) * 1000,
    }
    try:
        res["body"] = resp.json()
    except ValueError:
        res["error"] = resp.text[:500]
        return res
    if resp.status_code != 200:
        res["error"] = json.dumps(res.pop("body"), ensure_ascii=False)[:500]
    return res


async def run_round(i, client, base_url, args, sem):
    prompt = (
        f"[probe {i}] A snail climbs a 30-foot pole. "
        "Each day it climbs 3 feet, each night it slides back 2 feet. "
        "On which day does it reach the top? Think step by step."
    )
    base_payload = {
        "model": args.model,
        "store": False,
        "include": ["reasoning.encrypted_content"],
        "reasoning": {"effort": args.effort, "summary": "auto"},
        "max_output_tokens": args.max_output_tokens,
    }

    async with sem:
        t1 = await post_responses(
            client,
            base_url,
            {**base_payload, "input": [{"role": "user", "content": prompt}]},
        )
        rec = {"round": i, "turn1": {k: v for k, v in t1.items() if k != "body"}}

        if t1["status"] != 200:
            return rec

        rec["turn1"].update(inspect_output(t1["body"]))

        if not rec["turn1"]["encrypted_items"] or args.no_turn2:
            return rec

        # Turn 2: send reasoning + message items back to test decryption
        items = [
            o
            for o in t1["body"]["output"]
            if o.get("type") in ("reasoning", "message")
        ]
        t2_input = [
            {"role": "user", "content": prompt},
            *items,
            {"role": "user", "content": "Now give the answer as a single number only."},
        ]
        t2 = await post_responses(
            client, base_url, {**base_payload, "input": t2_input}
        )
        rec["turn2"] = {k: v for k, v in t2.items() if k != "body"}
        if t2["status"] == 200:
            rec["turn2"].update(inspect_output(t2["body"]))
        rec["turn2"]["decrypt_error"] = is_decrypt_error(t2.get("error") or "")
        return rec


def summarize(records):
    c = collections.Counter()
    for r in records:
        t1 = r["turn1"]
        c[f"turn1 status {t1['status']}"] += 1
        if t1["status"] == 200:
            c["turn1 has reasoning_items"] += t1.get("reasoning_items", 0) > 0
            c["turn1 has encrypted_content"] += t1.get("encrypted_items", 0) > 0
        t2 = r.get("turn2")
        if t2:
            c[f"turn2 status {t2['status']}"] += 1
            c["turn2 decrypt errors"] += bool(t2.get("decrypt_error"))
    return dict(sorted(c.items()))


async def main():
    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    p.add_argument("--base-url", default=os.environ.get("BASE_URL", "https://api.openai.com"))
    p.add_argument("--api-key", dest="api_key", default=os.environ.get("LOADTEST_TOKENS"))
    p.add_argument("--model", default="gpt-6-luna")
    p.add_argument("--rounds", type=int, default=20)
    p.add_argument("--concurrency", type=int, default=10)
    p.add_argument("--effort", default="low", choices=["minimal", "low", "medium", "high"])
    p.add_argument("--max-output-tokens", type=int, default=512)
    p.add_argument("--timeout", type=float, default=120)
    p.add_argument("--no-turn2", action="store_true", help="only check turn-1 output")
    p.add_argument("--insecure", action="store_true", help="skip TLS verification")
    p.add_argument("--out", help="write per-round records + summary as JSON")
    # legacy alias so old invocations still work
    p.add_argument("--json-out", dest="out", help=argparse.SUPPRESS)
    args = p.parse_args()

    if not args.api_key:
        sys.exit("need --api-key or LOADTEST_TOKENS env var")

    base_url = args.base_url.rstrip("/").removesuffix("/v1")
    headers = {
        "Authorization": f"Bearer {args.api_key}",
        "Content-Type": "application/json",
    }
    sem = asyncio.Semaphore(args.concurrency)

    async with httpx.AsyncClient(
        headers=headers,
        timeout=args.timeout,
        verify=not args.insecure,
    ) as client:
        records = await asyncio.gather(
            *(run_round(i, client, base_url, args, sem) for i in range(args.rounds))
        )

    for r in sorted(records, key=lambda x: x["round"]):
        t1 = r["turn1"]
        line = (
            f"#{r['round']:>3}  t1={t1['status']}  rid={t1.get('request_id')}  "
            f"reasoning={t1.get('reasoning_items', '-')}  "
            f"encrypted={t1.get('encrypted_items', '-')}({t1.get('encrypted_len', 0)}B)  "
            f"rtok={t1.get('reasoning_tokens')}"
        )
        t2 = r.get("turn2")
        if t2:
            line += f"  |  t2={t2['status']}  rid={t2.get('request_id')}"
            if t2.get("decrypt_error"):
                line += "  DECRYPT_ERROR"
        err = (t2 or {}).get("error") or t1.get("error")
        if err:
            line += f"\n      {err[:300]}"
        print(line)

    summary = summarize(records)
    print("\n== summary")
    for k, v in summary.items():
        print(f"  {k:<36} {v}")

    t2_total = sum(1 for r in records if r.get("turn2"))
    t2_fail = sum(1 for r in records if r.get("turn2") and r["turn2"]["status"] != 200)
    if t2_total and 0 < t2_fail < t2_total:
        print(
            "  -> turn-2 partly fails: likely multiple channels/accounts in rotation "
            "for this model (encrypted_content keys differ between channels)"
        )

    if args.out:
        with open(args.out, "w") as f:
            json.dump({"summary": summary, "records": records}, f, ensure_ascii=False, indent=2)
        print(f"\nresults written to {args.out}")


if __name__ == "__main__":
    asyncio.run(main())
