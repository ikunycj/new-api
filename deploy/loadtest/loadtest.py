#!/usr/bin/env python3
"""Python port of k6/load.js.

Drives the same profiles, reads the same loadtest.config.json, and checks the
same thresholds, without requiring k6. The target endpoint and API keys can be
set from the command line or the same environment variables load.js reads.

    pip install httpx
    python3 loadtest.py --base-url https://alltokenapi.com --api-key sk-xxx --model gpt-4o-mini smoke

A single Python process is fine for a few hundred concurrent requests. For
capacity measurements beyond that, run several processes on separate hosts.
"""

import argparse
import asyncio
import hashlib
import json
import os
import random
import re
import resource
import signal
import sys
import time
import uuid
from array import array
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, List, Optional

import httpx

PROFILES = ["smoke", "step", "steady", "spike", "burst", "stream", "mixed", "soak", "capacity", "channel-failover"]
DURATION_PART = re.compile(r"(\d+(?:\.\d+)?)(ms|s|m|h)")
DURATION_UNITS = {"ms": 0.001, "s": 1, "m": 60, "h": 3600}


def parse_duration(value) -> float:
    if isinstance(value, (int, float)):
        return float(value)
    text = str(value).strip()
    parts = DURATION_PART.findall(text)
    if not parts or "".join(number + unit for number, unit in parts) != text:
        raise ValueError(f"invalid duration: {value!r}")
    return sum(float(number) * DURATION_UNITS[unit] for number, unit in parts)


def percentile(sorted_values, p: float) -> float:
    if not sorted_values:
        return 0.0
    rank = (len(sorted_values) - 1) * p / 100
    low = int(rank)
    high = min(low + 1, len(sorted_values) - 1)
    return sorted_values[low] + (sorted_values[high] - sorted_values[low]) * (rank - low)


def distribution(values) -> dict:
    ordered = sorted(values)
    if not ordered:
        return {"count": 0}
    return {
        "count": len(ordered),
        "avg": sum(ordered) / len(ordered),
        "min": ordered[0],
        "p50": percentile(ordered, 50),
        "p90": percentile(ordered, 90),
        "p95": percentile(ordered, 95),
        "p99": percentile(ordered, 99),
        "max": ordered[-1],
    }


# Cache misses during the first seconds are the unavoidable cost of writing each
# session's prefix; the steady-state hit rate excludes them.
CACHE_WARMUP_SECONDS = 30


class Metrics:
    def __init__(self):
        self.started = time.monotonic()
        self.cache_hit_min_tokens = 1
        self.durations = {"true": array("d"), "false": array("d")}
        self.ttfb = array("d")
        self.first_chunk = array("d")
        self.responses = Counter()
        self.transport_errors = Counter()
        self.error_samples = []
        self.requests = 0
        self.failed = 0
        self.application_errors = 0
        self.usage_checked = 0
        self.usage_missing = 0
        self.prompt_tokens = 0
        self.completion_tokens = 0
        self.total_tokens = 0
        self.cache_read_tokens = 0
        self.cache_write_tokens = 0
        self.cache_hits = 0
        self.cache_misses = 0
        self.steady_cache_checked = 0
        self.steady_cache_hits = 0
        self.refusals = 0
        self.refusal_categories = Counter()
        self.refusal_request_ids = []
        self.dropped_iterations = 0
        self.interrupted_iterations = 0

    def record(self, name, streaming, status, duration_ms, ttfb_ms, has_error_body):
        self.requests += 1
        self.durations["true" if streaming else "false"].append(duration_ms)
        if ttfb_ms is not None:
            self.ttfb.append(ttfb_ms)
        self.responses[(name, status)] += 1
        if status == 0 or status >= 400:
            self.failed += 1
        if status != 200 or has_error_body:
            self.application_errors += 1

    def record_usage(self, usage):
        self.usage_checked += 1
        valid = isinstance(usage, dict) and all(
            isinstance(usage.get(key), (int, float)) for key in ("prompt_tokens", "completion_tokens", "total_tokens")
        )
        if not valid:
            self.usage_missing += 1
            return
        self.prompt_tokens += usage["prompt_tokens"]
        self.completion_tokens += usage["completion_tokens"]
        self.total_tokens += usage["total_tokens"]
        details = usage.get("prompt_tokens_details") or {}
        cache_read = details.get("cached_tokens") or 0
        self.cache_read_tokens += cache_read
        self.cache_write_tokens += details.get("cache_write_tokens") or 0
        # 号池等上游会在请求前注入固定系统提示词并总能命中，用阈值只统计命中了我们自己内容的请求
        hit = cache_read >= self.cache_hit_min_tokens
        if hit:
            self.cache_hits += 1
        else:
            self.cache_misses += 1
        if time.monotonic() - self.started >= CACHE_WARMUP_SECONDS:
            self.steady_cache_checked += 1
            self.steady_cache_hits += hit

    def value(self, metric, stat, tag=None):
        if metric == "http_req_failed":
            return self.failed / self.requests if self.requests else 0.0
        if metric == "new_api_application_errors":
            return self.application_errors / self.requests if self.requests else 0.0
        if metric == "new_api_usage_missing":
            return self.usage_missing / self.usage_checked if self.usage_checked else 0.0
        if metric == "dropped_iterations":
            return self.dropped_iterations
        if metric == "new_api_time_to_first_byte":
            values = self.ttfb
        elif tag:
            values = self.durations[tag]
        else:
            values = list(self.durations["true"]) + list(self.durations["false"])
        return percentile(sorted(values), float(stat[1:]))


@dataclass
class Scenario:
    name: str
    executor: str
    exec_name: str
    vu_offset: int = 0
    vus: int = 0
    duration: float = 0.0
    graceful: float = 30.0
    start_vus: int = 0
    stages: Optional[List[tuple]] = None
    iterations: Optional[int] = None
    start_rate: int = 0
    max_vus: int = 0
    target: int = 0
    deadline: float = float("inf")

    def vu_active(self, index, stop_event):
        return not stop_event.is_set() and index < self.target and time.monotonic() < self.deadline


@dataclass
class Threshold:
    metric: str
    stat: str
    op: str
    limit: float
    tag: Optional[str] = None

    def describe(self):
        label = f"{self.metric}{{stream:{self.tag}}}" if self.tag else self.metric
        return f"{label} {self.stat}{self.op}{self.limit:g}"


CONTEXT_TOWNS = ("Willowbrook", "Elmsford", "Brightwater", "Ashdale", "Fernhill", "Cloverfield")
CONTEXT_SCENES = (
    "the bakery on the corner filled the street with the smell of warm bread and cinnamon rolls",
    "children gathered at the pond to feed the ducks and race little paper boats",
    "the old librarian read picture books aloud to a circle of sleepy toddlers",
    "farmers brought baskets of apples, pumpkins and honey to the weekend market",
    "neighbours painted the fence of the community garden in soft shades of yellow",
    "a small orchestra practised folk songs in the town square as the sun went down",
)
DEFAULT_CONTEXT_QUESTION = "Please retell your favourite part of this story in a few sentences."


def build_context(target_tokens, seed=""):
    # 按约 4 字符/token 生成确定性的日常小故事，用于模拟较长的输入上下文。
    # 刻意用自然叙述：运维报告、带编号的结构化日志一类文本会被 Opus 5 的输出分类器按一定比例判为 cyber 而拒答
    if target_tokens <= 0:
        return ""
    rng = random.Random(target_tokens)
    parts = []
    while sum(len(part) + 1 for part in parts) < target_tokens * 4:
        parts.append(
            f"In {rng.choice(CONTEXT_TOWNS)}, {rng.choice(CONTEXT_SCENES)}. "
            f"Later that afternoon {rng.choice(CONTEXT_SCENES)}, and everyone agreed it had been a lovely day."
        )
    # seed 放在最前面：不同 seed 的前缀从第一个 token 起就不同，互不共享上游 prompt cache
    header = f"Story collection {seed}. " if seed else ""
    return header + "Here is a gentle story about life in a few small towns.\n\n" + " ".join(parts)


class LoadTest:
    def __init__(self, args, config):
        self.args = args
        self.config = config
        workload = config.get("workload", {})
        self.pacing = workload.get("pacing", {})
        request = workload.get("request", {})
        self.base_url = args.base_url.rstrip("/")
        self.base_url = re.sub(r"/v1$", "", self.base_url)
        self.model = args.model or workload.get("model", "gpt-3.5-turbo")
        self.users = args.users or workload.get("users", 1000)
        self.max_tokens = args.max_tokens or workload.get("max_output_tokens", 256)
        # 默认不发 temperature：Opus 4.7+/Opus 5/Sonnet 5 等模型只接受默认采样参数，传了会被上游 400
        self.temperature = args.temperature if args.temperature is not None else request.get("temperature")
        self.context_tokens = args.context_tokens if args.context_tokens is not None else request.get("context_tokens", 0)
        if self.context_tokens:
            self.message = args.message or DEFAULT_CONTEXT_QUESTION
        else:
            self.message = args.message or request.get("message", "Return a deterministic load-test response.")
        self.context = build_context(self.context_tokens, args.context_seed)
        self.request_timeout = parse_duration(args.timeout or workload.get("request_timeout", "120s"))
        self.model_list_timeout = parse_duration(workload.get("model_list_timeout", "30s"))
        self.tokens = args.api_keys
        self.api_format = args.api_format
        self.metrics = Metrics()
        self.metrics.cache_hit_min_tokens = args.cache_hit_min_tokens
        self.started_at = time.monotonic()
        self.stop_event: Optional[asyncio.Event] = None
        self.client: Optional[httpx.AsyncClient] = None

    def headers(self, vu, iteration):
        if self.tokens:
            token = self.tokens[(vu - 1) % len(self.tokens)]
        else:
            token = f"sk-loadtest{((vu - 1) % self.users) + 1:05d}"
        headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "User-Agent": f"new-api-loadtest/{self.args.profile}",
            "X-Load-Test-ID": f"{self.args.profile}-{vu}-{iteration}",
            "X-Alltoken-Failover-Mode": self.args.failover_mode,
        }
        if self.api_format == "claude":
            headers["x-api-key"] = token
            headers["anthropic-version"] = "2023-06-01"
        return headers

    async def pace(self, min_key, min_fallback, max_key, max_fallback):
        low = float(self.pacing.get(min_key, min_fallback))
        high = float(self.pacing.get(max_key, max_fallback))
        if low < 0 or high < low:
            raise ValueError(f"{min_key}/{max_key} must define a valid range")
        await asyncio.sleep(low + random.random() * (high - low))

    def messages(self, vu, iteration):
        if not self.context:
            return [{"role": "user", "content": self.message}]
        # 共享上下文放最前面，唯一编号放在后面：开启缓存时所有请求前缀一致，可以命中前缀缓存
        suffix = f"(reader {self.args.profile}-{vu}-{iteration}) {self.message}"
        if not self.args.context_cache:
            # 关闭缓存时把唯一编号放到最前面，让每个请求的前缀都不同
            return [{"role": "user", "content": f"{suffix}\n\n{self.context}"}]
        if self.api_format == "claude":
            # Claude 需要显式 cache_control 才会缓存；断点打在不变的上下文块上
            return [{"role": "user", "content": [
                {"type": "text", "text": self.context, "cache_control": {"type": "ephemeral"}},
                {"type": "text", "text": suffix},
            ]}]
        # OpenAI 格式把共享上下文放进独立的 system 消息：部分上游（如 Azure 部署）按整条消息匹配缓存，
        # 上下文和唯一编号拼在同一条消息里时每次都会整段重写，拆开后与真实业务的 system + 问题一致
        return [{"role": "system", "content": self.context}, {"role": "user", "content": suffix}]

    async def chat_request(self, vu, iteration, streaming):
        body = {
            "model": self.model,
            "messages": self.messages(vu, iteration),
            "stream": streaming,
            self.args.max_tokens_field: self.max_tokens,
        }
        if self.temperature is not None:
            body["temperature"] = self.temperature
        session = (vu - 1) % self.args.sessions + 1 if self.args.sessions else vu
        if self.api_format == "responses":
            # Responses API：input 代替 messages，输出上限用 max_output_tokens
            body = {"model": self.model, "input": self.messages(vu, iteration), "stream": streaming,
                    "max_output_tokens": self.max_tokens}
            if self.temperature is not None:
                body["temperature"] = self.temperature
            url = f"{self.base_url}/v1/responses"
            if self.args.prompt_cache_key:
                # new-api 内置的 "codex cli trace" 亲和规则按 /v1/responses 的 prompt_cache_key 粘住渠道
                body["prompt_cache_key"] = f"loadtest-{self.args.context_seed or 'ctx'}-s{session}"
        elif self.api_format == "claude":
            url = f"{self.base_url}/v1/messages"
            if self.context and self.args.context_cache:
                # 网关内置的 "claude cli trace" 渠道亲和规则按 metadata.user_id 粘住渠道，每个 VU 模拟一个会话
                user_id = f"loadtest-{self.args.profile}-vu{session}"
                if self.args.claude_code_user_id:
                    # Claude Code 的 user_id 形如 user_<64hex>_account__session_<uuid>，号池常按其中的 session uuid 做粘性
                    seed = f"{user_id}-{self.args.context_seed}"
                    user_id = (f"user_{hashlib.sha256(seed.encode()).hexdigest()}_account__session_"
                               f"{uuid.uuid5(uuid.NAMESPACE_URL, seed)}")
                body["metadata"] = {"user_id": user_id}
        else:
            url = f"{self.base_url}/v1/chat/completions"
            if streaming:
                body["stream_options"] = {"include_usage": True}
            if self.args.prompt_cache_key:
                # OpenAI 按 prompt_cache_key + 前缀路由缓存；上游 new-api 也可按它做渠道亲和
                body["prompt_cache_key"] = f"loadtest-{self.args.context_seed or 'ctx'}-s{session}"
        name = "streaming chat" if streaming else "chat"
        start = time.perf_counter()
        result = {"status": 0, "ttfb": None, "error_body": False, "usage": None}
        try:
            await asyncio.wait_for(
                self.send_chat(url, body, self.headers(vu, iteration), streaming, start, result),
                timeout=self.request_timeout,
            )
        except asyncio.TimeoutError:
            self.metrics.transport_errors["RequestTimeout"] += 1
            result["error_detail"] = "RequestTimeout"
            # 流可能在 200 之后才中断，此时响应不完整，不能算成功
            result["status"] = 0
        except httpx.HTTPError as error:
            self.metrics.transport_errors[type(error).__name__] += 1
            result["error_detail"] = f"{type(error).__name__}: {error}"
            result["status"] = 0
        duration_ms = (time.perf_counter() - start) * 1000
        self.metrics.record(name, streaming, result["status"], duration_ms, result["ttfb"], result["error_body"])
        if (result["status"] != 200 or result["error_body"]) and len(self.metrics.error_samples) < 20:
            self.metrics.error_samples.append({
                "elapsed_s": round(time.monotonic() - self.started_at, 1),
                "status": result["status"],
                "request_id": result.get("request_id"),
                "detail": (result.get("error_detail") or "")[:500],
            })
        if result.get("refusal"):
            self.metrics.refusals += 1
            self.metrics.refusal_categories[result["refusal"]] += 1
            if len(self.metrics.refusal_request_ids) < 50:
                self.metrics.refusal_request_ids.append({"request_id": result.get("request_id"), "category": result["refusal"]})
        if result["status"] == 200:
            self.metrics.record_usage(result["usage"])
        elif not streaming:
            self.metrics.record_usage(None)

    async def send_chat(self, url, body, headers, streaming, start, result):
        async with self.client.stream("POST", url, json=body, headers=headers) as response:
            result["status"] = response.status_code
            result["request_id"] = response.headers.get("x-oneapi-request-id")
            result["ttfb"] = (time.perf_counter() - start) * 1000
            if not streaming or response.status_code != 200:
                text = (await response.aread()).decode("utf-8", errors="replace")
                if self.api_format == "responses":
                    # Responses 正常响应里也带 "error": null，只有非空 error 才算错误
                    try:
                        result["error_body"] = bool(json.loads(text).get("error"))
                    except (ValueError, AttributeError):
                        result["error_body"] = True
                else:
                    result["error_body"] = '"error"' in text
                if response.status_code != 200 or result["error_body"]:
                    result["error_detail"] = text
                try:
                    result["usage"] = self.normalize_usage(json.loads(text).get("usage"))
                except (ValueError, AttributeError):
                    result["usage"] = None
                return
            async for line in response.aiter_lines():
                # Responses 的 response.created 等事件里带 "error": null，错误改由事件类型判断
                if self.api_format != "responses" and '"error"' in line:
                    result["error_body"] = True
                    result["error_detail"] = line
                if not line.startswith("data:"):
                    continue
                payload = line[5:].strip()
                if payload == "[DONE]":
                    continue
                if result.get("first_chunk") is None:
                    result["first_chunk"] = True
                    self.metrics.first_chunk.append((time.perf_counter() - start) * 1000)
                try:
                    chunk = json.loads(payload)
                except ValueError:
                    continue
                if not isinstance(chunk, dict):
                    continue
                if self.api_format == "responses":
                    event = chunk.get("type", "")
                    if event in ("error", "response.failed") or (event == "response.completed" and (chunk.get("response") or {}).get("error")):
                        result["error_body"] = True
                        result["error_detail"] = payload
                    if event in ("response.completed", "response.incomplete"):
                        result["usage"] = self.normalize_usage((chunk.get("response") or {}).get("usage"))
                elif self.api_format == "claude":
                    delta = chunk.get("delta") or {}
                    if delta.get("stop_reason") == "refusal":
                        result["refusal"] = (delta.get("stop_details") or {}).get("category") or "unknown"
                    # Claude 流式 usage 分两段：message_start 带 input，message_delta 带最终 output
                    usage = chunk.get("usage") or (chunk.get("message") or {}).get("usage")
                    if usage:
                        claude_usage = result.setdefault("claude_usage", {})
                        claude_usage.update({k: v for k, v in usage.items() if v is not None})
                        result["usage"] = self.normalize_usage(claude_usage)
                elif chunk.get("usage"):
                    result["usage"] = chunk["usage"]

    def normalize_usage(self, usage):
        # 把 Claude usage 统一成 prompt/completion/total_tokens，便于和 OpenAI 格式共用统计
        if self.api_format == "openai" or not isinstance(usage, dict):
            return usage
        if self.api_format == "responses":
            if not isinstance(usage.get("input_tokens"), (int, float)) or not isinstance(usage.get("output_tokens"), (int, float)):
                return None
            details = usage.get("input_tokens_details") or {}
            return {
                "prompt_tokens": usage["input_tokens"],
                "completion_tokens": usage["output_tokens"],
                "total_tokens": usage.get("total_tokens") or usage["input_tokens"] + usage["output_tokens"],
                "prompt_tokens_details": {"cached_tokens": details.get("cached_tokens") or 0,
                                          "cache_write_tokens": details.get("cache_write_tokens") or 0},
            }
        if not isinstance(usage.get("input_tokens"), (int, float)) or not isinstance(usage.get("output_tokens"), (int, float)):
            return None
        cache_read = usage.get("cache_read_input_tokens") or 0
        cache_write = usage.get("cache_creation_input_tokens") or 0
        prompt = usage["input_tokens"] + cache_write + cache_read
        completion = usage["output_tokens"]
        return {
            "prompt_tokens": prompt,
            "completion_tokens": completion,
            "total_tokens": prompt + completion,
            "prompt_tokens_details": {"cached_tokens": cache_read, "cache_write_tokens": cache_write},
        }

    async def model_list(self, vu, iteration):
        start = time.perf_counter()
        status, ttfb, error_body = 0, None, False
        try:
            response = await self.client.get(
                f"{self.base_url}/v1/models", headers=self.headers(vu, iteration), timeout=self.model_list_timeout
            )
            status = response.status_code
            ttfb = response.elapsed.total_seconds() * 1000
            error_body = '"error"' in response.text
        except httpx.HTTPError as error:
            self.metrics.transport_errors[type(error).__name__] += 1
        self.metrics.record("model list", False, status, (time.perf_counter() - start) * 1000, ttfb, error_body)

    async def iterate(self, exec_name, vu, iteration):
        if exec_name == "chat":
            await self.chat_request(vu, iteration, False)
            await self.pace("chat_min_seconds", 0.2, "chat_max_seconds", 1.0)
        elif exec_name == "stream":
            await self.chat_request(vu, iteration, True)
            await self.pace("stream_min_seconds", 0.1, "stream_max_seconds", 0.5)
        elif exec_name == "modelList":
            await self.model_list(vu, iteration)
            await self.pace("model_list_min_seconds", 0.5, "model_list_max_seconds", 1.5)
        else:
            await self.chat_request(vu, iteration, self.args.stream)

    async def run_vu(self, scenario: Scenario, index: int):
        iteration = 0
        while scenario.vu_active(index, self.stop_event):
            if scenario.iterations is not None and iteration >= scenario.iterations:
                return
            await self.iterate(scenario.exec_name, scenario.vu_offset + index + 1, iteration)
            iteration += 1

    async def finish(self, tasks, timeout):
        """Wait for in-flight iterations up to the graceful window, then interrupt the rest."""
        tasks = [task for task in tasks if task is not None]
        if tasks:
            waiter = asyncio.gather(*tasks, return_exceptions=True)
            stopper = asyncio.ensure_future(self.stop_event.wait())
            await asyncio.wait({waiter, stopper}, timeout=timeout, return_when=asyncio.FIRST_COMPLETED)
            stopper.cancel()
            for task in tasks:
                if not task.done():
                    task.cancel()
                    self.metrics.interrupted_iterations += 1
            await asyncio.gather(*tasks, return_exceptions=True)

    async def wait_or_stop(self, seconds):
        try:
            await asyncio.wait_for(self.stop_event.wait(), timeout=max(seconds, 0))
        except asyncio.TimeoutError:
            pass

    async def run_scenario(self, scenario: Scenario):
        if scenario.executor == "constant-vus":
            scenario.target = scenario.vus
            scenario.deadline = time.monotonic() + scenario.duration
            tasks = [asyncio.ensure_future(self.run_vu(scenario, index)) for index in range(scenario.vus)]
            await self.wait_or_stop(scenario.duration)
            await self.finish(tasks, scenario.graceful)
        elif scenario.executor == "per-vu-iterations":
            scenario.target = scenario.vus
            scenario.deadline = time.monotonic() + scenario.duration
            tasks = [asyncio.ensure_future(self.run_vu(scenario, index)) for index in range(scenario.vus)]
            await self.finish(tasks, scenario.duration)
        elif scenario.executor == "ramping-vus":
            await self.run_ramping_vus(scenario)
        elif scenario.executor == "ramping-arrival-rate":
            await self.run_arrival_rate(scenario)

    async def run_ramping_vus(self, scenario: Scenario):
        tasks = {}
        start = time.monotonic()
        total = sum(duration for _, duration in scenario.stages)
        while not self.stop_event.is_set():
            elapsed = time.monotonic() - start
            if elapsed >= total:
                break
            scenario.target = round(stage_value(scenario.start_vus, scenario.stages, elapsed))
            for index in range(scenario.target):
                if index not in tasks or tasks[index].done():
                    tasks[index] = asyncio.ensure_future(self.run_vu(scenario, index))
            await self.wait_or_stop(0.2)
        scenario.target = scenario.stages[-1][0] if scenario.stages else 0
        await self.finish(list(tasks.values()), scenario.graceful)

    async def run_arrival_rate(self, scenario: Scenario):
        in_flight = set()
        iteration = 0
        pending = 0.0
        start = previous = time.monotonic()
        total = sum(duration for _, duration in scenario.stages)
        while not self.stop_event.is_set():
            now = time.monotonic()
            elapsed = now - start
            if elapsed >= total:
                break
            pending += stage_value(scenario.start_rate, scenario.stages, elapsed) * (now - previous)
            previous = now
            while pending >= 1:
                pending -= 1
                if len(in_flight) >= scenario.max_vus:
                    self.metrics.dropped_iterations += 1
                    continue
                task = asyncio.ensure_future(self.iterate(scenario.exec_name, iteration % scenario.max_vus + 1, iteration))
                in_flight.add(task)
                task.add_done_callback(in_flight.discard)
                iteration += 1
            await asyncio.sleep(0.005)
        await self.finish(list(in_flight), scenario.graceful)

    async def reset_mock_channels(self):
        mock_channels = {item.get("id"): item for item in self.config.get("mock", {}).get("channels", [])}
        targets = [(self.args.mock_control_a, 1, 300), (self.args.mock_control_b, 2, 3000), (self.args.mock_control_c, 3, 3000)]
        for base, channel_id, fallback in targets:
            tokens = mock_channels.get(channel_id, {}).get("tokens", fallback)
            response = await self.client.post(f"{base.rstrip('/')}/control/reset", params={"tokens": tokens})
            if response.status_code != 200:
                raise RuntimeError(f"failed to reset mock channel {channel_id}: HTTP {response.status_code}")

    async def report_progress(self, started):
        last_requests = 0
        last_misses = 0
        while not self.stop_event.is_set():
            await self.wait_or_stop(10)
            metrics = self.metrics
            elapsed = time.monotonic() - started
            print(
                f"[{elapsed:7.1f}s] requests={metrics.requests} (+{metrics.requests - last_requests}/10s) "
                f"failed={metrics.failed} app_errors={metrics.application_errors} dropped={metrics.dropped_iterations} "
                f"cache_miss=+{metrics.cache_misses - last_misses}/10s",
                flush=True,
            )
            last_requests = metrics.requests
            last_misses = metrics.cache_misses

    async def run(self, scenarios, thresholds):
        limits = httpx.Limits(max_connections=None, max_keepalive_connections=None)
        timeout = httpx.Timeout(self.request_timeout)
        self.stop_event = asyncio.Event()
        loop = asyncio.get_running_loop()
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, self.stop_event.set)
        async with httpx.AsyncClient(limits=limits, timeout=timeout, verify=not self.args.insecure) as client:
            self.client = client
            if self.args.profile == "channel-failover":
                await self.reset_mock_channels()
            started = self.started_at = time.monotonic()
            progress = asyncio.ensure_future(self.report_progress(started))
            await asyncio.gather(*(self.run_scenario(scenario) for scenario in scenarios))
            progress.cancel()
            elapsed = time.monotonic() - started
        return self.summarize(elapsed, thresholds)

    def summarize(self, elapsed, thresholds):
        metrics = self.metrics
        all_durations = list(metrics.durations["true"]) + list(metrics.durations["false"])
        results = []
        for threshold in thresholds:
            actual = metrics.value(threshold.metric, threshold.stat, threshold.tag)
            passed = actual < threshold.limit if threshold.op == "<" else actual == threshold.limit
            results.append({"threshold": threshold.describe(), "actual": actual, "passed": passed})
        return {
            "profile": self.args.profile,
            "base_url": self.base_url,
            "model": self.model,
            "elapsed_seconds": elapsed,
            "requests": metrics.requests,
            "success_rate": 1 - metrics.value("new_api_application_errors", "rate") if metrics.requests else 0.0,
            "rps": metrics.requests / elapsed if elapsed else 0.0,
            "http_req_failed_rate": metrics.value("http_req_failed", "rate"),
            "application_error_rate": metrics.value("new_api_application_errors", "rate"),
            "responses": {f"{name} {status}": count for (name, status), count in sorted(metrics.responses.items())},
            "transport_errors": dict(metrics.transport_errors),
            "error_samples": metrics.error_samples,
            "http_req_duration_ms": distribution(all_durations),
            "stream_duration_ms": distribution(metrics.durations["true"]),
            "time_to_first_byte_ms": distribution(metrics.ttfb),
            "time_to_first_chunk_ms": distribution(metrics.first_chunk),
            "tokens": {
                "prompt": metrics.prompt_tokens,
                "completion": metrics.completion_tokens,
                "total": metrics.total_tokens,
                "output_tps": metrics.completion_tokens / elapsed if elapsed else 0.0,
                "usage_missing_rate": metrics.value("new_api_usage_missing", "rate"),
                "cache_read": metrics.cache_read_tokens,
                "cache_write": metrics.cache_write_tokens,
                "cache_hit_requests": metrics.cache_hits,
                "refusals": metrics.refusals,
                "refusal_categories": dict(metrics.refusal_categories),
                "refusal_samples": metrics.refusal_request_ids,
                "cache_hit_rate": metrics.cache_hits / metrics.usage_checked if metrics.usage_checked else 0.0,
                "cache_hit_rate_after_warmup": (
                    metrics.steady_cache_hits / metrics.steady_cache_checked if metrics.steady_cache_checked else 0.0
                ),
            },
            "dropped_iterations": metrics.dropped_iterations,
            "interrupted_iterations": metrics.interrupted_iterations,
            "thresholds": results,
        }


def stage_value(start_value, stages, elapsed):
    previous = start_value
    for target, duration in stages:
        if elapsed < duration:
            return previous + (target - previous) * (elapsed / duration if duration else 1)
        elapsed -= duration
        previous = target
    return previous


def parse_stages(stages):
    return [(int(stage["target"]), parse_duration(stage["duration"])) for stage in stages]


def build_scenarios(args, config):
    workload = config.get("workload", {})
    profiles = workload.get("profiles", {})
    capacity = workload.get("capacity", {})
    smoke = profiles.get("smoke") or workload.get("smoke", {})
    burst = profiles.get("burst") or workload.get("burst", {})
    users = args.users or workload.get("users", 1000)
    profile = args.profile

    if profile == "smoke":
        return [Scenario("smoke", "constant-vus", "chat", vus=args.smoke_vus or smoke.get("vus", 10),
                         duration=parse_duration(args.smoke_duration or smoke.get("duration", "1m")),
                         graceful=parse_duration(smoke.get("graceful_stop", "30s")))]
    if profile in ("step", "spike"):
        stage_config = profiles.get(profile, {})
        return [Scenario(profile, "ramping-vus", "chat", start_vus=stage_config.get("start_vus", 0),
                         stages=parse_stages(stage_config.get("stages", [])),
                         graceful=parse_duration(stage_config.get("graceful_ramp_down", "30s")))]
    if profile in ("steady", "soak", "stream", "channel-failover"):
        key = "channel_failover" if profile == "channel-failover" else profile
        defaults = {"steady": (1000, "30m", "2m"), "soak": (500, "2h", "2m"), "stream": (1000, "15m", "2m"),
                    "channel_failover": (5, "30s", "30s")}[key]
        profile_config = profiles.get(key, {})
        return [Scenario(profile, "constant-vus", "stream" if profile == "stream" else "chat",
                         vus=profile_config.get("vus", defaults[0]),
                         duration=parse_duration(profile_config.get("duration", defaults[1])),
                         graceful=parse_duration(profile_config.get("graceful_stop", defaults[2])))]
    if profile == "burst":
        return [Scenario("burst", "per-vu-iterations", "chat", vus=users, iterations=1,
                         duration=parse_duration(burst.get("max_duration", "5m")))]
    if profile == "mixed":
        mixed = profiles.get("mixed", {})
        duration = parse_duration(mixed.get("duration", "20m"))
        graceful = parse_duration(mixed.get("graceful_stop", "2m"))
        stream_vus = mixed.get("stream_vus", 700)
        chat_vus = mixed.get("chat_vus", 200)
        return [
            Scenario("stream", "constant-vus", "stream", vu_offset=0, vus=stream_vus, duration=duration, graceful=graceful),
            Scenario("chat", "constant-vus", "chat", vu_offset=stream_vus, vus=chat_vus, duration=duration, graceful=graceful),
            Scenario("models", "constant-vus", "modelList", vu_offset=stream_vus + chat_vus,
                     vus=mixed.get("model_vus", 100), duration=duration,
                     graceful=parse_duration(mixed.get("model_graceful_stop", "30s"))),
        ]
    if profile == "capacity" and args.rpm:
        rate = args.rpm / 60
        return [Scenario("capacity", "ramping-arrival-rate", "capacityChat", start_rate=rate,
                         stages=[(rate, parse_duration(args.duration))],
                         max_vus=capacity.get("max_vus", 2000),
                         graceful=parse_duration(args.timeout or workload.get("request_timeout", "120s")))]
    if profile == "capacity":
        rates = args.capacity_rates or capacity.get("rates", [100, 200, 300, 400, 500, 600, 700, 800])
        stages = []
        for index, rate in enumerate(rates):
            ramp = capacity.get("first_ramp_duration", "10s") if index == 0 else capacity.get("ramp_duration", "30s")
            stages.append((rate, parse_duration(ramp)))
            stages.append((rate, parse_duration(capacity.get("stage_duration", "2m"))))
        stages.append((0, parse_duration(capacity.get("cooldown_duration", "30s"))))
        return [Scenario("capacity", "ramping-arrival-rate", "capacityChat", start_rate=rates[0], stages=stages,
                         max_vus=capacity.get("max_vus", 2000),
                         graceful=parse_duration(capacity.get("graceful_stop", "1m")))]
    raise ValueError(f"Unknown profile: {profile}")


def build_thresholds(profile, config):
    limits = config.get("thresholds", {})
    error_rate = limits.get("error_rate", 0.01)
    common = [
        Threshold("http_req_failed", "rate", "<", error_rate),
        Threshold("new_api_application_errors", "rate", "<", error_rate),
        Threshold("http_req_duration", "p95", "<", limits.get("request_p95_ms", 2000)),
        Threshold("http_req_duration", "p99", "<", limits.get("request_p99_ms", 5000)),
        Threshold("new_api_time_to_first_byte", "p95", "<", limits.get("ttfb_p95_ms", 1000)),
    ]
    if profile == "stream":
        return common[:2] + [
            Threshold("http_req_duration", "p95", "<", limits.get("stream_p95_ms", 5000)),
            Threshold("http_req_duration", "p99", "<", limits.get("stream_p99_ms", 10000)),
            common[4],
        ]
    if profile == "mixed":
        return common + [Threshold("http_req_duration", "p95", "<", limits.get("stream_p95_ms", 5000), tag="true")]
    if profile == "capacity":
        return common + [
            Threshold("dropped_iterations", "count", "==", 0),
            Threshold("new_api_usage_missing", "rate", "<", limits.get("usage_missing_rate", 0.001)),
        ]
    if profile == "channel-failover":
        return common[:1] + [Threshold("new_api_application_errors", "rate", "<", 0.01)] + common[2:]
    return common


def print_summary(summary):
    def row(label, stats):
        if not stats.get("count"):
            return f"  {label:<22} -"
        return (f"  {label:<22} avg={stats['avg']:.0f} p50={stats['p50']:.0f} p95={stats['p95']:.0f} "
                f"p99={stats['p99']:.0f} max={stats['max']:.0f} (n={stats['count']})")

    print()
    print(f"== {summary['profile']} against {summary['base_url']} model={summary['model']} "
          f"in {summary['elapsed_seconds']:.1f}s")
    print(f"  requests               {summary['requests']} ({summary['rps']:.2f}/s)")
    print(f"  http_req_failed        {summary['http_req_failed_rate']:.2%}")
    print(f"  success rate           {summary['success_rate']:.2%}")
    print(f"  application errors     {summary['application_error_rate']:.2%}")
    for key, count in summary["responses"].items():
        print(f"  status {key:<15} {count}")
    for name, count in summary["transport_errors"].items():
        print(f"  transport {name:<12} {count}")
    for sample in summary["error_samples"][:5]:
        print(f"  error sample [{sample['elapsed_s']}s] status={sample['status']} "
              f"request_id={sample['request_id']} {sample['detail'][:200]}")
    print(row("duration ms", summary["http_req_duration_ms"]))
    print(row("ttfb ms", summary["time_to_first_byte_ms"]))
    print(row("first chunk ms", summary["time_to_first_chunk_ms"]))
    tokens = summary["tokens"]
    print(f"  tokens                 prompt={tokens['prompt']} completion={tokens['completion']} "
          f"total={tokens['total']} output_tps={tokens['output_tps']:.1f} usage_missing={tokens['usage_missing_rate']:.2%}")
    print(f"  prompt cache           read={tokens['cache_read']} write={tokens['cache_write']} "
          f"hit_requests={tokens['cache_hit_requests']} hit_rate={tokens['cache_hit_rate']:.2%} "
          f"hit_rate_after_{CACHE_WARMUP_SECONDS}s={tokens['cache_hit_rate_after_warmup']:.2%}")
    if tokens["refusals"]:
        categories = ", ".join(f"{k}={v}" for k, v in tokens["refusal_categories"].items())
        print(f"  refusals               {tokens['refusals']} (stop_reason=refusal; {categories})")
    print(f"  dropped / interrupted  {summary['dropped_iterations']} / {summary['interrupted_iterations']}")
    print("  thresholds:")
    for result in summary["thresholds"]:
        mark = "PASS" if result["passed"] else "FAIL"
        print(f"    [{mark}] {result['threshold']} (actual {result['actual']:.4g})")


def split_list(raw, convert: Callable = str):
    if not raw:
        return []
    return [convert(value.strip()) for value in raw.split(",") if value.strip()]


def parse_args():
    env = os.environ.get
    here = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(description="Python port of the new-api k6 load test.")
    parser.add_argument("profile", nargs="?", default=env("LOAD_PROFILE", "smoke"), choices=PROFILES)
    parser.add_argument("--base-url", default=env("BASE_URL", "http://127.0.0.1:3000"),
                        help="gateway base URL; a trailing /v1 is stripped (env BASE_URL)")
    parser.add_argument("--api-key", action="append", dest="api_keys", default=[],
                        help="API key, repeatable or comma-separated; VUs rotate through them (env LOADTEST_TOKENS)")
    parser.add_argument("--config", default=env("LOADTEST_CONFIG_FILE", str(here / "loadtest.config.json")))
    parser.add_argument("--model", default=env("LOADTEST_MODEL"))
    parser.add_argument("--users", type=int, default=int(env("LOADTEST_USERS", 0)) or None)
    parser.add_argument("--max-tokens", type=int, default=int(env("MAX_OUTPUT_TOKENS", 0)) or None)
    parser.add_argument("--max-tokens-field", choices=("max_tokens", "max_completion_tokens"), default="max_tokens",
                        help="request field for the output limit; OpenAI reasoning models require max_completion_tokens")
    parser.add_argument("--temperature", type=float,
                        default=float(env("REQUEST_TEMPERATURE")) if env("REQUEST_TEMPERATURE") else None)
    parser.add_argument("--message", default=env("LOADTEST_MESSAGE"))
    parser.add_argument("--context-tokens", type=int,
                        default=int(env("LOADTEST_CONTEXT_TOKENS")) if env("LOADTEST_CONTEXT_TOKENS") else None,
                        help="prepend a generated document of roughly this many input tokens (env LOADTEST_CONTEXT_TOKENS)")
    parser.add_argument("--context-seed", default="",
                        help="tag added at the start of the context so separate runs/keys get distinct prefixes and do not share prompt cache")
    parser.add_argument("--context-cache", action=argparse.BooleanOptionalAction, default=True,
                        help="keep the context as a shared prefix so prompt caching can hit; claude adds cache_control "
                             "(default on; --no-context-cache makes every prefix unique)")
    parser.add_argument("--claude-code-user-id", action="store_true",
                        help="claude + context cache only: send metadata.user_id in Claude Code format (user_<hex>_account__session_<uuid>)")
    parser.add_argument("--sessions", type=int, default=None,
                        help="rotate metadata.user_id (claude) or prompt_cache_key (openai) over this many sessions so "
                             "channel affinity keeps each session on one channel (default: one session per VU)")
    parser.add_argument("--cache-hit-min-tokens", type=int, default=1,
                        help="count a request as a cache hit only when cached tokens reach this value; raise it above the "
                             "upstream-injected prefix (e.g. 5000 for pools that add ~4.4K tokens) to measure hits on your own prompt")
    parser.add_argument("--prompt-cache-key", action="store_true",
                        help="openai only: send prompt_cache_key per session (see --sessions)")
    parser.add_argument("--timeout", default=None, help="per-request timeout, e.g. 120s (default from config)")
    parser.add_argument("--failover-mode", default=env("FAILOVER_MODE", "balanced"))
    parser.add_argument("--smoke-vus", type=int, default=int(env("SMOKE_VUS", 0)) or None)
    parser.add_argument("--smoke-duration", default=env("SMOKE_DURATION"))
    parser.add_argument("--capacity-rates", default=env("CAPACITY_RATES"), help="comma-separated RPS steps")
    parser.add_argument("--rpm", type=float, default=None,
                        help="capacity profile only: hold a fixed arrival rate in requests per minute")
    parser.add_argument("--duration", default="5m", help="how long to hold --rpm, e.g. 5m (default 5m)")
    parser.add_argument("--stream", action="store_true", help="capacity profile only: send streaming requests")
    parser.add_argument("--api-format", choices=("openai", "claude", "responses"), default=env("LOADTEST_API_FORMAT", "openai"),
                        help="openai: /v1/chat/completions; claude: /v1/messages; responses: /v1/responses (env LOADTEST_API_FORMAT)")
    parser.add_argument("--mock-control-a", default=env("MOCK_CONTROL_A", "http://127.0.0.1:18080"))
    parser.add_argument("--mock-control-b", default=env("MOCK_CONTROL_B", "http://127.0.0.1:18081"))
    parser.add_argument("--mock-control-c", default=env("MOCK_CONTROL_C", "http://127.0.0.1:18082"))
    parser.add_argument("--summary-out", default=None, help="write the summary JSON to this path")
    parser.add_argument("--insecure", action="store_true", help="skip TLS certificate verification")
    args = parser.parse_args()
    args.api_keys = [key for raw in args.api_keys for key in split_list(raw)] or split_list(env("LOADTEST_TOKENS"))
    args.capacity_rates = split_list(args.capacity_rates, int) or None
    return args


def main():
    args = parse_args()
    config = json.loads(Path(args.config).read_text())
    scenarios = build_scenarios(args, config)
    thresholds = build_thresholds(args.profile, config)

    soft, hard = resource.getrlimit(resource.RLIMIT_NOFILE)
    wanted = hard if hard != resource.RLIM_INFINITY else 65536
    if soft < wanted:
        try:
            resource.setrlimit(resource.RLIMIT_NOFILE, (wanted, hard))
        except (ValueError, OSError):
            pass

    test = LoadTest(args, config)
    summary = asyncio.run(test.run(scenarios, thresholds))
    print_summary(summary)
    if args.summary_out:
        Path(args.summary_out).parent.mkdir(parents=True, exist_ok=True)
        Path(args.summary_out).write_text(json.dumps(summary, indent=2, ensure_ascii=False))
    sys.exit(0 if all(result["passed"] for result in summary["thresholds"]) else 99)


if __name__ == "__main__":
    main()
