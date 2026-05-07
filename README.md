# SoloDuckDB (`soloduck`)

Interactive **DuckDB** shell implemented in **[Solod](https://github.com/solod-dev/solod)** (So) and linked against **libduckdb**. This repository builds the **`soloduck`** binary.

> **Experimental research code.** SoloDuckDB is a proof-of-concept to explore embedding DuckDB behind a Solod-compiled CLI. It is **not** a supported product, **not** audited for security or correctness, and **not** suitable for production workloads. Prefer the [official DuckDB CLI](https://duckdb.org/docs/stable/clients/cli/overview) for anything that matters.

This repository is **self-contained**: it vendors the compiler/stdlib as a **git submodule** (`solod` → [solod-dev/solod](https://github.com/solod-dev/solod)) and ships the DuckDB C bindings in **`duckdb/`**. No fork of Solod is required.

CLI behavior is loosely aligned with the official DuckDB CLI (LTS):

- [CLI overview](https://duckdb.org/docs/stable/clients/cli/overview)
- [Arguments](https://duckdb.org/docs/stable/clients/cli/arguments)
- [Dot commands](https://duckdb.org/docs/stable/clients/cli/dot_commands)

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

- **Interactive REPL** (no `-c` and no SQL argument): prompt `D`, type SQL ending with **`;`** then **Enter**; dot-commands (`.help`, `.mode`, `.open`, `.read`, …). Lines are read with **`bufio.Reader.ReadString('\\n')`** on **`os.Stdin`** (same pattern as typical Go CLIs). Use **`-batch`** when stdin is a pipe or file script.
- **`-batch`**: read stdin as a script with no prompts (pipes and files).
- **Output modes**: `-csv`, `-json`, or `.mode …`. Default **`duckbox`** tries to mirror the official CLI table layout (UTF-8 box drawing, column names, physical type row, separator, data rows; numeric columns right-aligned). Use `.mode ascii` for plain `|` tables without the type row.

## Benchmarks (startup + first query)

To compare **process startup + one `-batch -c` query + exit** between the official CLI and `soloduck` (useful as a rough “initialization” cost for scripted use), build `soloduck` and run:

```bash
python3 scripts/benchmark_init.py --runs 25 --warmup 2
```

Optional environment variables: `DUCKDB_CLI`, `SOLODUCK`. Example:

```bash
DUCKDB_CLI=/opt/homebrew/bin/duckdb SOLODUCK=./soloduck python3 scripts/benchmark_init.py
```

Example output (numbers vary by machine and load):

```
Wall time per fresh process (startup + query + exit).
Timed runs per binary (after warmup): 25

Query: 'SELECT 1;'
  duckdb      mean    13.41 ms  σ   0.22 ms  min   13.06 ms  max   13.63 ms
  soloduck    mean    14.03 ms  σ   0.25 ms  min   13.67 ms  max   14.34 ms
  ratio (soloduck / duckdb mean wall time): 1.05x
```

Lower milliseconds are faster. The script measures wall time only; CPU time is not split between DuckDB and the Solod runtime.

## CI

Pushes and pull requests against **`main`** run [`.github/workflows/ci.yml`](.github/workflows/ci.yml): Linux (**amd64** + **arm64**) and **macOS** builds using the same DuckDB **static-libs** bundles as [releases](.github/workflows/release.yml), then `./soloduck -version`.

## Layout

| Path | Purpose |
|------|---------|
| `solod/` | Submodule: upstream Solod (`so translate`, `so/*` stdlib). Includes `LICENSE` (BSD-3-Clause). |
| `duckdb/` | Solod package wrapping DuckDB’s C API (`duckdb.h`). |
| `main.go` | CLI entrypoint. |
| `gen/` | Generated C from `make translate` (removed by `make clean`). |
| `scripts/benchmark_init.py` | Optional startup/latency comparison vs official `duckdb`. |

`go.mod` uses `replace solod.dev => ./solod` so imports resolve to the submodule checkout.

## Static binaries

So emits C; fully static linking depends on a static `libduckdb` and your platform.

Published **GitHub Releases** include CI-built **Linux** (x86_64, arm64) and **macOS** (amd64 or arm64, matches the `macos-latest` runner) binaries; see `.github/workflows/release.yml`. CI unpacks DuckDB’s **`static-libs-*.zip`** (many `lib*.a` archives, not only `libduckdb_static.a`). On Linux, GNU `ld` uses `-Wl,--start-group` … `-Wl,--end-group` and `-static-libgcc`/`-static-libstdc++`. On macOS, Apple’s linker uses **`-Wl,-force_load`** per archive (see `Makefile`). The system C/C++ runtime and frameworks stay dynamic as usual on each OS.

## License

Upstream **Solod** is BSD-3-Clause — see `solod/LICENSE`. Combine with your choice for **soloduck**-specific files in this repository.
