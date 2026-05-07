<img width="200" height="200" alt="image" src="https://github.com/user-attachments/assets/55ddc237-5b36-4c5f-8f8e-a7552aa3e6fa" />

# SoloDuckDB CLI

Interactive **DuckDB** shell implemented in **[Solod](https://github.com/solod-dev/solod)** (So) and linked against **libduckdb**. 

> **Experimental research code.** SoloDuckDB is a proof-of-concept to explore embedding DuckDB with Solod. It is **not** a supported project, **not** audited for security or correctness, and **not** suitable for production workloads. Prefer the [official DuckDB CLI](https://duckdb.org/docs/stable/clients/cli/overview) for anything that matters.

This research repository is **self-contained** and vendors requirements as **git submodules**

CLI behavior is loosely aligned with the official DuckDB CLI (LTS):

- [CLI overview](https://duckdb.org/docs/stable/clients/cli/overview)
- [Arguments](https://duckdb.org/docs/stable/clients/cli/arguments)
- [Dot commands](https://duckdb.org/docs/stable/clients/cli/dot_commands)

```
% soloduck -c "SELECT version()"
┌-------------┐
│ "version"() │
│   varchar   │
├-------------┤
│ v1.5.2      │
└-------------┘
```

## Build

From a clone of this repo (with submodules):

```bash
git clone --recurse-submodules https://github.com/lmangani/soloduck.git
cd soloduck
make
./soloduck -version
```

If you cloned without submodules:

```bash
git submodule update --init --recursive
```

You need **libduckdb** on the machine ([install](https://duckdb.org/install/?environment=c)). On macOS with Homebrew, `make` picks up `$(brew --prefix duckdb)` automatically. Else point the linker at headers and libraries:

```bash
make DUCK_PREFIX=/usr/local
```

`make` runs Solod `so translate` into **`gen/`**, then compiles and links **`soloduck`** next to the `Makefile`. Use `make clean` to remove generated C.

## Usage

Show flags:

```bash
./soloduck -help
```

One-shot SQL (same idea as official `duckdb -c`):

```bash
./soloduck -batch -c "SELECT version();"
./soloduck :memory: 'SELECT 42 AS n'
```

- **Interactive REPL** (no `-c` and no SQL argument)
- **`-batch`**: read stdin as a script with no prompts (pipes and files).
- **Output modes**: `-csv`, `-json`, or `.mode …`. Default **`duckbox`** tries to mirror the official CLI layout

## Benchmarks (startup + first query)

Each script measures **wall time per fresh process**: startup, run one query via `-batch -c`, exit. Absolute milliseconds depend on machine and DuckDB linkage; **ratios** are the useful headline.

### vs official DuckDB CLI

Build `soloduck`, install the official CLI, then:

```bash
python3 scripts/benchmark_init.py --runs 25 --warmup 2
```

Optional: `DUCKDB_CLI`, `SOLODUCK`.

```bash
DUCKDB_CLI=/opt/homebrew/bin/duckdb SOLODUCK=./soloduck python3 scripts/benchmark_init.py
```

Reference benchmark (`python3 scripts/benchmark_init.py --runs 25 --warmup 2` on Apple Silicon M4; numbers vary by machine and load):

```
Wall time per fresh process (startup + query + exit).
Timed runs per binary (after warmup): 25

Query: 'SELECT 1;'
  duckdb      mean    15.11 ms  σ   0.64 ms  min   14.19 ms  max   16.55 ms
  soloduck    mean    14.54 ms  σ   0.62 ms  min   13.45 ms  max   15.85 ms
  ratio (soloduck / duckdb mean wall time): 0.96x

Query: 'SELECT version();'
  duckdb      mean    15.10 ms  σ   0.68 ms  min   13.92 ms  max   16.74 ms
  soloduck    mean    14.26 ms  σ   0.57 ms  min   13.44 ms  max   15.17 ms
  ratio (soloduck / duckdb mean wall time): 0.94x
```

### vs Go (`database/sql` + [duckdb-go](https://github.com/duckdb/duckdb-go), CGO)

The driver is **not** pure Go (CGO + bundled libduckdb). This compares **Go runtime + `database/sql` + duckdb-go** against **Solod + DuckDB’s C API** in `soloduck`. Build the harness once, then benchmark:

```bash
python3 scripts/benchmark_init_go.py --build --runs 25 --warmup 2
```

Reference benchmark (`python3 scripts/benchmark_init_go.py --runs 25 --warmup 2` on the same machine, harness already built):

```
Wall time per fresh process (startup + query + exit).
Timed runs per binary (after warmup): 25
Compare: soloduck (Solod + libduckdb C API) vs benchmark_go_once (Go + database/sql + duckdb-go / CGO).

Query: 'SELECT 1;'
  duckdb-go     mean    17.01 ms  σ   0.75 ms  min   15.91 ms  max   18.62 ms
  soloduck      mean    14.55 ms  σ   0.48 ms  min   13.77 ms  max   15.40 ms
  ratio (soloduck / duckdb-go mean wall time): 0.86x

Query: 'SELECT version();'
  duckdb-go     mean    17.18 ms  σ   1.28 ms  min   15.71 ms  max   21.64 ms
  soloduck      mean    14.83 ms  σ   0.78 ms  min   13.79 ms  max   16.74 ms
  ratio (soloduck / duckdb-go mean wall time): 0.86x
```

## CI

Pushes and pull requests against **`main`** run [`.github/workflows/ci.yml`](.github/workflows/ci.yml): Linux (**amd64** + **arm64**) and **macOS** builds using the same DuckDB **static-libs** bundles as [releases](.github/workflows/release.yml), then `./soloduck -version`.

## Layout

| Path | Purpose |
|------|---------|
| `solod/` | Submodule: upstream Solod (`so translate`, `so/*` stdlib). Includes `LICENSE` (BSD-3-Clause). |
| `duckdb/` | Solod package wrapping DuckDB’s C API (`duckdb.h`). |
| `main.go` | CLI entrypoint. |
| `gen/` | Generated C from `make translate` (removed by `make clean`). |
| `scripts/benchmark_init.py` | Startup/latency vs official `duckdb` CLI. |
| `scripts/benchmark_init_go.py` | Startup/latency vs Go `database/sql` + duckdb-go (CGO). |

`go.mod` uses `replace solod.dev => ./solod` so imports resolve to the submodule checkout.

## Static binaries

[Solod](https://github.com/solod-dev/solod) emits `C`; fully static linking depends on a static `libduckdb` and your platform.

