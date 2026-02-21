#!/usr/bin/env python3
"""
Benchmark script for gemini-web-to-api.
Measures response latency per model over multiple rounds.

Usage:
    python3 scripts/benchmark.py [options]

Examples:
    python3 scripts/benchmark.py
    python3 scripts/benchmark.py --models gemini-2.5-flash gemini-2.5-pro
    python3 scripts/benchmark.py --rounds 10 --base-url http://localhost:4981
    python3 scripts/benchmark.py --prompt "Write a haiku about Go programming"
"""

import argparse
import json
import statistics
import subprocess
import sys
import time

DEFAULT_BASE_URL = "http://localhost:4981"
DEFAULT_MODELS = ["gemini-2.5-flash", "gemini-2.5-pro"]
DEFAULT_ROUNDS = 5
DEFAULT_PROMPT = "reply with just: ok"
TIMEOUT = 180


def parse_args():
    parser = argparse.ArgumentParser(description="Benchmark gemini-web-to-api response latency")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help=f"API base URL (default: {DEFAULT_BASE_URL})")
    parser.add_argument("--models", nargs="+", default=DEFAULT_MODELS, help="Models to benchmark")
    parser.add_argument("--rounds", type=int, default=DEFAULT_ROUNDS, help=f"Rounds per model (default: {DEFAULT_ROUNDS})")
    parser.add_argument("--prompt", default=DEFAULT_PROMPT, help="Prompt to send each round")
    parser.add_argument("--timeout", type=int, default=TIMEOUT, help=f"Request timeout in seconds (default: {TIMEOUT})")
    parser.add_argument("--delay", type=float, default=0, help="Delay in seconds between requests (default: 0)")
    return parser.parse_args()


def check_health(base_url):
    r = subprocess.run(
        ["curl", "-s", "--max-time", "5", f"{base_url}/health"],
        capture_output=True, text=True
    )
    try:
        data = json.loads(r.stdout)
        return data.get("status") == "ok"
    except Exception:
        return False


def run_request(base_url, model, prompt, timeout):
    payload = json.dumps({
        "model": model,
        "messages": [{"role": "user", "content": prompt}]
    })
    start = time.time()
    r = subprocess.run(
        ["curl", "-s", "--max-time", str(timeout),
         f"{base_url}/openai/v1/chat/completions",
         "-H", "Content-Type: application/json",
         "-d", payload],
        capture_output=True, text=True
    )
    elapsed_ms = round((time.time() - start) * 1000)
    try:
        content = json.loads(r.stdout)["choices"][0]["message"]["content"]
        return elapsed_ms, content, None
    except Exception:
        try:
            error = json.loads(r.stdout).get("error", {}).get("message", r.stdout[:120])
        except Exception:
            error = r.stdout[:120] or "no response (timeout?)"
        return elapsed_ms, None, error


def print_header(args):
    print()
    print("=" * 60)
    print("  gemini-web-to-api benchmark")
    print("=" * 60)
    print(f"  Base URL : {args.base_url}")
    print(f"  Models   : {', '.join(args.models)}")
    print(f"  Rounds   : {args.rounds}")
    print(f"  Prompt   : {args.prompt}")
    print(f"  Timeout  : {args.timeout}s")
    print(f"  Delay    : {args.delay}s between requests")
    print("=" * 60)


def benchmark(args):
    print_header(args)

    print("\nChecking server health...", end=" ", flush=True)
    if not check_health(args.base_url):
        print("FAILED")
        print(f"  Server at {args.base_url} is not reachable or unhealthy.")
        sys.exit(1)
    print("OK")

    results = {}

    for model in args.models:
        times = []
        errors = []
        failed_rounds = []
        print(f"\n{model}:")

        for i in range(1, args.rounds + 1):
            print(f"  round {i}/{args.rounds} ...", end=" ", flush=True)
            elapsed, content, error = run_request(args.base_url, model, args.prompt, args.timeout)

            if content is not None:
                print(f"{elapsed}ms ✓")
                times.append(elapsed)
            else:
                print(f"FAILED — {error}")
                errors.append(error)
                failed_rounds.append(i)

            if args.delay > 0 and i < args.rounds:
                time.sleep(args.delay)

        results[model] = {
            "times": times,
            "errors": errors,
            "avg":    round(statistics.mean(times))              if times else None,
            "median": round(statistics.median(times))            if times else None,
            "min":    min(times)                                 if times else None,
            "max":    max(times)                                 if times else None,
            "stdev":  round(statistics.stdev(times))            if len(times) > 1 else 0,
            "success_rate": f"{len(times)}/{len(times) + len(errors)}",
            "failed_rounds": failed_rounds,
        }

    print_results(results)


def print_results(results):
    print()
    print("=" * 60)
    print("  RESULTS")
    print("=" * 60)
    header = f"  {'Model':<22} {'Avg':>7} {'Median':>7} {'Min':>7} {'Max':>7} {'StdDev':>7} {'OK':>5}"
    print(header)
    print("  " + "-" * 56)

    for model, r in results.items():
        if r["avg"] is not None:
            print(
                f"  {model:<22}"
                f" {r['avg']:>6}ms"
                f" {r['median']:>6}ms"
                f" {r['min']:>6}ms"
                f" {r['max']:>6}ms"
                f" {r['stdev']:>6}ms"
                f" {r['success_rate']:>5}"
            )
        else:
            print(f"  {model:<22}  all rounds failed ({r['success_rate']})")

    print("=" * 60)

    # Winner
    scored = {m: r["median"] for m, r in results.items() if r["median"] is not None}
    if scored:
        winner = min(scored, key=scored.get)
        print(f"\n  Fastest (by median): {winner} — {scored[winner]}ms")

    # Failure position analysis
    any_failures = any(r["failed_rounds"] for r in results.values())
    if any_failures:
        print()
        print("  Failure position analysis:")
        for model, r in results.items():
            if r["failed_rounds"]:
                total = len(r["times"]) + len(r["errors"])
                early = [n for n in r["failed_rounds"] if n <= total // 2]
                late  = [n for n in r["failed_rounds"] if n >  total // 2]
                print(f"    {model}: failed at rounds {r['failed_rounds']}  (early={len(early)} late={len(late)})")
            else:
                print(f"    {model}: no failures")
    print()


if __name__ == "__main__":
    benchmark(parse_args())
