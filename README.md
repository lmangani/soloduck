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

## Layout

| Path | Purpose |
|------|--------|
| `solod/` | Submodule: upstream Solod (`so translate`, `so/*` stdlib). |
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

## Static binaries

So emits C; fully static linking depends on a static `libduckdb` and your platform.

Published **GitHub Releases** include CI-built Linux **x86_64** and **arm64** binaries (see `.github/workflows/release.yml`): CI unpacks DuckDB’s **`static-libs-linux-*.zip`** (many `lib*.a` archives, not only `libduckdb_static.a`) and links them with `-Wl,--start-group` … `-Wl,--end-group`, plus `-static-libgcc`/`-static-libstdc++`. The C library (`libc`) stays dynamically linked unless you use a musl or fully `-static` toolchain yourself.

## License

Combine Solod’s BSD-3-Clause with your choice for this repo.
