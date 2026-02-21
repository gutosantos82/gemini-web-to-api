#!/usr/bin/env python3
"""
Sweep different inter-request delays to find the minimum interval
that achieves a target success rate.

Usage:
    python3 scripts/delay_sweep.py [options]

Examples:
    python3 scripts/delay_sweep.py
    python3 scripts/delay_sweep.py --models gemini-2.5-flash --rounds 10
    python3 scripts/delay_sweep.py --delays 0 2 5 10 15 30
"""

import argparse
import json
import statistics
import subprocess
import sys
import time

DEFAULT_BASE_URL  = "http://localhost:4981"
DEFAULT_MODELS    = ["gemini-2.5-flash", "gemini-2.5-pro"]
DEFAULT_ROUNDS    = 8
DEFAULT_DELAYS    = [0, 2, 5, 10, 15, 30]
DEFAULT_TARGET    = 100   # % success rate considered "sweet spot"
DEFAULT_PROMPT    = "Write a detailed technical explanation of how garbage collection works in Go, covering the tri-color mark-and-sweep algorithm, write barriers, and how the GC interacts with goroutines. Include practical implications for developers writing high-performance Go code."
TIMEOUT           = 180


def parse_args():
    parser = argparse.ArgumentParser(description="Sweep inter-request delays to find the reliability sweet spot")
    parser.add_argument("--base-url",  default=DEFAULT_BASE_URL)
    parser.add_argument("--models",    nargs="+", default=DEFAULT_MODELS)
    parser.add_argument("--rounds",    type=int,   default=DEFAULT_ROUNDS,  help=f"Requests per delay value (default: {DEFAULT_ROUNDS})")
    parser.add_argument("--delays",    nargs="+",  type=float, default=DEFAULT_DELAYS, help="Delay values in seconds to test")
    parser.add_argument("--target",    type=int,   default=DEFAULT_TARGET,  help=f"Target success %% (default: {DEFAULT_TARGET})")
    parser.add_argument("--prompt",    default=DEFAULT_PROMPT)
    parser.add_argument("--timeout",   type=int,   default=TIMEOUT)
    return parser.parse_args()


def check_health(base_url):
    r = subprocess.run(
        ["curl", "-s", "--max-time", "5", f"{base_url}/health"],
        capture_output=True, text=True
    )
    try:
        return json.loads(r.stdout).get("status") == "ok"
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
        return elapsed_ms, True
    except Exception:
        return elapsed_ms, False


def run_delay(base_url, model, delay, rounds, prompt, timeout):
    successes, times = 0, []
    for i in range(1, rounds + 1):
        elapsed, ok = run_request(base_url, model, prompt, timeout)
        status = "✓" if ok else "✗"
        print(f"    [{status}] round {i}/{rounds} — {elapsed}ms", flush=True)
        if ok:
            successes += 1
            times.append(elapsed)
        if delay > 0 and i < rounds:
            time.sleep(delay)
    success_rate = round(successes / rounds * 100)
    avg = round(statistics.mean(times)) if times else None
    return success_rate, avg


def sweep(args):
    print()
    print("=" * 62)
    print("  gemini-web-to-api — delay sweep")
    print("=" * 62)
    print(f"  Base URL : {args.base_url}")
    print(f"  Models   : {', '.join(args.models)}")
    print(f"  Delays   : {args.delays}s")
    print(f"  Rounds   : {args.rounds} per delay")
    print(f"  Target   : {args.target}% success rate")
    print("=" * 62)

    print("\nChecking server health...", end=" ", flush=True)
    if not check_health(args.base_url):
        print("FAILED")
        sys.exit(1)
    print("OK\n")

    # results[model][delay] = (success_rate, avg_ms)
    results = {m: {} for m in args.models}
    sweet_spots = {m: None for m in args.models}

    for model in args.models:
        print(f"{'─' * 62}")
        print(f"  Model: {model}")
        print(f"{'─' * 62}")
        for delay in args.delays:
            label = f"{delay}s delay"
            print(f"\n  {label}:")
            success_rate, avg = run_delay(args.base_url, model, delay, args.rounds, args.prompt, args.timeout)
            results[model][delay] = (success_rate, avg)
            avg_str = f"{avg}ms avg" if avg else "—"
            print(f"  → {success_rate}% success  {avg_str}")
            if success_rate >= args.target and sweet_spots[model] is None:
                sweet_spots[model] = delay
                print(f"  ★ Sweet spot reached at {delay}s delay!")

    print_summary(args, results, sweet_spots)


def print_summary(args, results, sweet_spots):
    print()
    print("=" * 62)
    print("  SUMMARY")
    print("=" * 62)

    for model in args.models:
        print(f"\n  {model}:")
        print(f"  {'Delay':>8}  {'Success':>8}  {'Avg latency':>12}")
        print(f"  {'─'*8}  {'─'*8}  {'─'*12}")
        for delay, (rate, avg) in results[model].items():
            avg_str = f"{avg}ms" if avg else "—"
            flag = " ★" if delay == sweet_spots[model] else ""
            print(f"  {str(delay)+'s':>8}  {str(rate)+'%':>8}  {avg_str:>12}{flag}")

        spot = sweet_spots[model]
        if spot is not None:
            print(f"\n  Sweet spot: {spot}s delay → {args.target}%+ success rate")
        else:
            print(f"\n  No delay in tested range reached {args.target}% success rate.")

    print()
    print("=" * 62)


if __name__ == "__main__":
    sweep(parse_args())
