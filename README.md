# soloduck

Interactive **DuckDB** shell in **[Solod](https://github.com/solod-dev/solod)** (So), linked against **libduckdb**.

This repository is **self-contained**: it vendors the compiler/stdlib as a **git submodule** (`solod` → [solod-dev/solod](https://github.com/solod-dev/solod)) and ships the DuckDB C bindings in **`duckdb/`**. No fork of Solod is required.

CLI behavior is loosely aligned with the official DuckDB CLI (LTS):

- [CLI overview](https://duckdb.org/docs/lts/clients/cli/overview)
- [Arguments](https://duckdb.org/docs/lts/clients/cli/arguments)
- [Dot commands](https://duckdb.org/docs/lts/clients/cli/dot_commands)

## Build (from this repo only)

```bash
git clone --recurse-submodules https://github.com/lmangani/soloduck.git
cd soloduck
make
./soloduck -version
```

If you cloned without submodules, fetch them once: `git submodule update --init`.

You need **libduckdb** on the machine ([install](https://duckdb.org/install/?environment=c)). On macOS with Homebrew, `make` picks up `$(brew --prefix duckdb)` automatically; elsewhere set `DUCK_PREFIX` if headers/libs are not found:

```bash
make DUCK_PREFIX=/usr/local
```

## CI

Pushes and pull requests against **`main`** run [`.github/workflows/ci.yml`](.github/workflows/ci.yml): Linux (**amd64** + **arm64**) and **macOS** builds using the same DuckDB **static-libs** bundles as [releases](.github/workflows/release.yml), then `./soloduck -version`.

## Layout

| Path | Purpose |
|------|--------|
| `solod/` | Submodule: upstream Solod (`so translate`, `so/*` stdlib). Includes `LICENSE` (BSD-3-Clause). |
| `duckdb/` | Solod package wrapping DuckDB’s C API (`duckdb.h`). |
| `main.go` | CLI entrypoint. |
| `gen/` | Generated C from `make translate` (removed by `make clean`). |

`go.mod` uses `replace solod.dev => ./solod` so imports resolve to the submodule checkout.

## Flags & usage

See `-help`. Common cases:

```bash
./soloduck -c 'SELECT version();'
./soloduck :memory: 'SELECT 42 AS n'
```

Run with no `-c` / second-arg SQL to open the **interactive shell**: type SQL ending with `;`; default **`duckbox`** output follows the DuckDB CLI layout (UTF-8 box borders, column names, **physical type** row, separator, then rows — numeric columns **right-aligned**). Use `-batch` for stdin scripts (`soloduck -batch < script.sql`). `.mode ascii` keeps plain `|` tables without the type row.

## Static binaries

So emits C; fully static linking depends on a static `libduckdb` and your platform.

Published **GitHub Releases** include CI-built **Linux** (x86_64, arm64) and **macOS** (amd64 or arm64, matches the `macos-latest` runner) binaries; see `.github/workflows/release.yml`. CI unpacks DuckDB’s **`static-libs-*.zip`** (many `lib*.a` archives, not only `libduckdb_static.a`). On Linux, GNU `ld` uses `-Wl,--start-group` … `-Wl,--end-group` and `-static-libgcc`/`-static-libstdc++`. On macOS, Apple’s linker uses **`-Wl,-force_load`** per archive (see `Makefile`). The system C/C++ runtime and frameworks stay dynamic as usual on each OS.

## License

Upstream **Solod** is BSD-3-Clause — see `solod/LICENSE`. Combine with your choice for **soloduck**-specific files in this repository.
