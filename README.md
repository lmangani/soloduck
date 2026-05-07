# soloduck

Interactive **DuckDB** shell written in **[Solod](https://github.com/lmangani/solod)** (So): the same Go subset the `solod` compiler translates to C, linked against **libduckdb**. This repo is both a small end-user CLI and a demo of [`solod.dev/so/duckdb`](https://github.com/lmangani/solod/tree/main/so/duckdb).

Behavior is **loosely aligned** with the official CLI described in the DuckDB docs (LTS):

- [CLI overview](https://duckdb.org/docs/lts/clients/cli/overview)
- [Command-line arguments](https://duckdb.org/docs/lts/clients/cli/arguments)
- [Dot commands](https://duckdb.org/docs/lts/clients/cli/dot_commands)
- [Output formats](https://duckdb.org/docs/lts/clients/cli/output_formats)

Gaps vs the full CLI include: no readline/history/syntax highlighting, no `~/.duckdbrc`, incomplete `.mode` set, `-readonly` not enforced at the C API layer, and `.complete` is keyword-prefix hints only—not engine-backed autocomplete.

## Submodule layout

Checked out under the `solod` tree as a **git submodule** (temporary, for co-development):

```bash
git submodule update --init soloduck
```

`go.mod` uses `replace solod.dev => ../` so imports resolve to the parent checkout.

## Build

Requires [libduckdb](https://duckdb.org/install/?environment=c) (headers + shared or static library), same as any So program using `so/duckdb`.

```bash
make              # translate with `so translate`, then cc + -lduckdb
./soloduck -version
./soloduck -c 'SELECT version();'
./soloduck :memory: 'SELECT 42 AS answer'
```

### Arguments (subset)

Matches the spirit of `duckdb [OPTIONS] [FILENAME] [SQL]` from the [overview](https://duckdb.org/docs/lts/clients/cli/overview#usage):

| Flag | Meaning |
|------|--------|
| `-help` | Usage summary and exit. |
| `-version` | Print `duckdb_library_version()` and exit. |
| `-c SQL` | Run SQL and exit (same role as official `-c` / `-s`). |
| `-csv` | Initial output mode: CSV. |
| `-json` | Initial output mode: JSON (array of objects). |
| `-readonly` | Reserved; prints a warning (read-only open not wired in `so/duckdb`). |

Positional: optional **FILENAME** (default in-memory: `:memory:`), optional **second** SQL string for one-shot execution.

### Interactive

- Startup banner lines similar to the official shell (“Enter `.help`…”, in-memory connection hint, `.open` hint).
- Prompt `D` with continuation lines for multi-line SQL until a line ends with `;`.
- Dot commands: `.help` `.exit` `.quit` `.open` `.read` `.tables` `.schema` `.mode` `.complete` (see `.help` in-app).

## Static binaries

So emits plain C; “static” is a **linker** concern. DuckDB ships `libduckdb_static.a` on many platforms; on **Linux** you can experiment with `-static` and the static archive (often easiest with **musl**). **macOS** rarely produces a portable fully static binary; prefer a normal dynamic link against `libduckdb` or codesign an app bundle.

## License

Same as the parent project you combine with (Solod + your choice for this repo).
