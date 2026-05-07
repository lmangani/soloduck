# soloduck

Interactive **DuckDB** shell written in **[Solod](https://github.com/solod-dev/solod)** (So), linked against **libduckdb**.

This repo contains **everything you need** besides upstream Solod and the DuckDB SDK:

- **`main.go`** — CLI (dot-commands, `-c`, output modes, …).
- **`duckdb/`** — Solod package wrapping the DuckDB C API (`duckdb.h`). Stock **`solod.dev`** does not ship DuckDB; this copy lives here so you can use **[solod-dev/solod](https://github.com/solod-dev/solod)** `main` without a fork.

Behavior is **loosely aligned** with the official DuckDB CLI (LTS docs):

- [CLI overview](https://duckdb.org/docs/lts/clients/cli/overview)
- [Command-line arguments](https://duckdb.org/docs/lts/clients/cli/arguments)
- [Dot commands](https://duckdb.org/docs/lts/clients/cli/dot_commands)
- [Output formats](https://duckdb.org/docs/lts/clients/cli/output_formats)

Gaps vs the full CLI: no readline/history/syntax highlighting, no `~/.duckdbrc`, incomplete `.mode` set, `-readonly` not enforced in the C shim, `.complete` is keyword-prefix hints only.

## Prerequisites

1. **Solod** — clone [github.com/solod-dev/solod](https://github.com/solod-dev/solod) (any revision that matches your `solod.dev` replace). You only need the compiler and `so/*` stdlib from that tree.
2. **libduckdb** — headers + library ([installation](https://duckdb.org/install/?environment=c)).

## Layout A — `soloduck` as a submodule (inside `solod/`)

```bash
git clone https://github.com/solod-dev/solod.git
cd solod
git submodule add https://github.com/lmangani/soloduck.git soloduck   # or: submodule update --init
cd soloduck
# go.mod already has: replace solod.dev => ../
make
./soloduck -version
```

## Layout B — sibling directories (`solod/` + `soloduck/`)

```bash
git clone https://github.com/solod-dev/solod.git
git clone https://github.com/lmangani/soloduck.git
cd soloduck
```

Edit `go.mod`: **comment** `replace solod.dev => ../` and **uncomment** `replace solod.dev => ../solod`.

```bash
make SOLID=../solod
```

## Build

```bash
make              # uses $(SOLID) default .. for submodule layout
./soloduck -c 'SELECT version();'
./soloduck :memory: 'SELECT 42 AS answer'
```

Override DuckDB install prefix if needed: `make DUCK_PREFIX=/path/to/libduckdb`.

### Arguments (subset)

| Flag | Meaning |
|------|--------|
| `-help` | Usage summary and exit. |
| `-version` | Print `duckdb_library_version()` and exit. |
| `-c SQL` | Run SQL and exit (same role as official `-c` / `-s`). |
| `-csv` | Initial output mode: CSV. |
| `-json` | Initial output mode: JSON (array of objects). |
| `-readonly` | Reserved; prints a warning (read-only open not wired in the shim). |

Positional: optional **FILENAME** (default `:memory:`), optional **second** SQL string for one-shot execution.

### Interactive

- Startup banner similar to the official shell; prompt `D`; multi-line SQL until a line ends with `;`.
- Dot commands: `.help` `.exit` `.quit` `.open` `.read` `.tables` `.schema` `.mode` `.complete`.

## Static binaries

So emits plain C; fully static linking depends on a static `libduckdb` and your platform (often easier on Linux + musl). See DuckDB’s C install docs.

## License

Same as the projects you combine (Solod is BSD-3-Clause; this repo may use the same or your choice).
