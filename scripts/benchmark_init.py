#!/usr/bin/env python3
"""
Compare wall-clock time for one-shot CLI runs: each invocation measures full process
startup, minimal DuckDB work, and exit. This approximates "initialization + first query"
latency for tools used as `duckdb -batch -c '…'` / `./soloduck -batch -c '…'`.

Requires both binaries on PATH or passed via --duckdb / --soloduck.
"""

from __future__ import annotations

import argparse
import os
import shutil
import statistics
import subprocess
import sys
import time


def run_once(cmd: list[str], sql: str) -> float:
    t0 = time.perf_counter()
    subprocess.run(
        cmd + ["-batch", "-c", sql],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=True,
    )
    return time.perf_counter() - t0


def summarize_ms(times: list[float]) -> tuple[float, float, float, float]:
    ms = [t * 1000.0 for t in times]
    return (
        statistics.mean(ms),
        statistics.stdev(ms) if len(ms) > 1 else 0.0,
        min(ms),
        max(ms),
    )


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--duckdb",
        default=os.environ.get("DUCKDB_CLI", "duckdb"),
        help="Path to official DuckDB CLI (default: duckdb or $DUCKDB_CLI)",
    )
    ap.add_argument(
        "--soloduck",
        default=os.environ.get("SOLODUCK", "./soloduck"),
        help="Path to soloduck binary (default: ./soloduck or $SOLODUCK)",
    )
    ap.add_argument("--runs", type=int, default=25, help="Timed iterations per binary (default: 25)")
    ap.add_argument(
        "--warmup",
        type=int,
        default=2,
        help="Warmup runs per binary before timing (default: 2)",
    )
    ap.add_argument(
        "--sql",
        action="append",
        dest="queries",
        metavar="SQL",
        help="SQL to run (repeatable). Default: SELECT 1; and SELECT version();",
    )
    args = ap.parse_args()

    queries = args.queries or ["SELECT 1;", "SELECT version();"]

    duckdb_bin = shutil.which(args.duckdb) if os.path.dirname(args.duckdb) == "" else args.duckdb
    if not duckdb_bin or not os.path.isfile(duckdb_bin):
        print(f"error: DuckDB CLI not found: {args.duckdb}", file=sys.stderr)
        return 1

    solo = args.soloduck
    if not os.path.isfile(solo):
        print(f"error: soloduck binary not found: {solo}", file=sys.stderr)
        return 1

    duck_cmd = [duckdb_bin]
    solo_cmd = [os.path.abspath(solo)]

    print(
        "Wall time per fresh process (startup + query + exit).\n"
        f"Timed runs per binary (after warmup): {args.runs}\n"
    )

    for sql in queries:
        label = sql.replace("\n", " ").strip()
        if len(label) > 60:
            label = label[:57] + "…"
        print(f"Query: {label!r}")

        for _ in range(max(0, args.warmup)):
            run_once(duck_cmd, sql)
        duck_times = [run_once(duck_cmd, sql) for _ in range(args.runs)]

        for _ in range(max(0, args.warmup)):
            run_once(solo_cmd, sql)
        solo_times = [run_once(solo_cmd, sql) for _ in range(args.runs)]

        for name, times in [("duckdb", duck_times), ("soloduck", solo_times)]:
            mean, stdev, vmin, vmax = summarize_ms(times)
            print(
                f"  {name:10}  mean {mean:8.2f} ms  σ {stdev:6.2f} ms  min {vmin:7.2f} ms  max {vmax:7.2f} ms"
            )

        md, sd = statistics.mean(duck_times), statistics.mean(solo_times)
        ratio = sd / md if md > 0 else 0.0
        print(f"  ratio (soloduck / duckdb mean wall time): {ratio:.2f}x\n")

    print(
        "Notes:\n"
        "  • Lower ms is faster.\n"
        "  • Results vary with disk cache and CPU governor; use ≥ 25 runs on a quiet machine.\n"
        "  • soloduck links the same libduckdb but adds the Solod-compiled runtime.\n"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
