#!/usr/bin/env python3
"""
Compare cold-process wall time: Solod + DuckDB C API (soloduck) vs
Go + database/sql + github.com/duckdb/duckdb-go (CGO + bundled libduckdb).

Each timed invocation is a fresh process: runtime startup, open DB, run one
query, drain results, exit. This isolates “initialization + first query” cost.

Requires:
  • Built soloduck (default ./soloduck)
  • Go toolchain + CGO (see duckdb-go docs): a C compiler is required for the driver.
  • The helper in scripts/benchmark_go_once — build once with --build, or pass --go-binary.

This does not change or depend on the main soloduck module; the harness lives in
scripts/benchmark_go_once as its own tiny go.mod.
"""

from __future__ import annotations

import argparse
import os
import shutil
import statistics
import subprocess
import sys
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_GO_DIR = os.path.join(SCRIPT_DIR, "benchmark_go_once")
DEFAULT_GO_BIN = os.path.join(DEFAULT_GO_DIR, "benchmark_go_once")


def run_soloduck_once(soloduck: str, sql: str) -> float:
    t0 = time.perf_counter()
    subprocess.run(
        [soloduck, "-batch", "-c", sql],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=True,
    )
    return time.perf_counter() - t0


def run_go_once(go_bin: str, sql: str) -> float:
    t0 = time.perf_counter()
    subprocess.run(
        [go_bin, sql],
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


def build_go_harness(go_dir: str, out_bin: str) -> None:
    env = os.environ.copy()
    env.setdefault("CGO_ENABLED", "1")
    subprocess.run(
        ["go", "build", "-o", out_bin, "."],
        cwd=go_dir,
        env=env,
        check=True,
    )


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--soloduck",
        default=os.environ.get("SOLODUCK", "./soloduck"),
        help="Path to soloduck binary (default: ./soloduck or $SOLODUCK)",
    )
    ap.add_argument(
        "--go-binary",
        default=os.environ.get("BENCHMARK_GO_ONCE", DEFAULT_GO_BIN),
        help=f"Path to benchmark_go_once executable (default: {DEFAULT_GO_BIN} or $BENCHMARK_GO_ONCE)",
    )
    ap.add_argument(
        "--go-dir",
        default=DEFAULT_GO_DIR,
        help="Directory with benchmark_go_once Go module (for --build)",
    )
    ap.add_argument(
        "--build",
        action="store_true",
        help="Run `go build` in --go-dir before benchmarking",
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

    solo = args.soloduck
    if not os.path.isfile(solo):
        print(f"error: soloduck binary not found: {solo}", file=sys.stderr)
        return 1

    go_bin = os.path.abspath(args.go_binary)
    go_dir = os.path.abspath(args.go_dir)

    if args.build:
        try:
            build_go_harness(go_dir, go_bin)
        except subprocess.CalledProcessError as e:
            print(f"error: go build failed (need CGO + gcc/clang): {e}", file=sys.stderr)
            return 1

    if not os.path.isfile(go_bin):
        print(
            f"error: Go harness not found: {go_bin}\n"
            "  Run with --build once, or build scripts/benchmark_go_once manually.",
            file=sys.stderr,
        )
        return 1

    solo_abs = os.path.abspath(solo)

    print(
        "Wall time per fresh process (startup + query + exit).\n"
        f"Timed runs per binary (after warmup): {args.runs}\n"
        "Compare: soloduck (Solod + libduckdb C API) vs benchmark_go_once "
        "(Go + database/sql + duckdb-go / CGO).\n"
    )

    for sql in queries:
        label = sql.replace("\n", " ").strip()
        if len(label) > 60:
            label = label[:57] + "…"
        print(f"Query: {label!r}")

        for _ in range(max(0, args.warmup)):
            run_go_once(go_bin, sql)
        go_times = [run_go_once(go_bin, sql) for _ in range(args.runs)]

        for _ in range(max(0, args.warmup)):
            run_soloduck_once(solo_abs, sql)
        solo_times = [run_soloduck_once(solo_abs, sql) for _ in range(args.runs)]

        for name, times in [
            ("duckdb-go", go_times),
            ("soloduck", solo_times),
        ]:
            mean, stdev, vmin, vmax = summarize_ms(times)
            print(
                f"  {name:12}  mean {mean:8.2f} ms  σ {stdev:6.2f} ms  min {vmin:7.2f} ms  max {vmax:7.2f} ms"
            )

        mg, ms = statistics.mean(go_times), statistics.mean(solo_times)
        ratio = ms / mg if mg > 0 else 0.0
        print(f"  ratio (soloduck / duckdb-go mean wall time): {ratio:.2f}x\n")

    print(
        "Notes:\n"
        "  • Lower ms is faster.\n"
        "  • duckdb-go is not “pure Go”: it uses CGO and ships/linksets libduckdb.\n"
        "  • Different DuckDB builds (bundled static vs your linker flags for soloduck) "
        "affect absolute ms; ratios are approximate.\n"
        "  • docs: https://github.com/duckdb/duckdb-go\n"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
