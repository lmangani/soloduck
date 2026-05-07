# soloduck

Interactive **DuckDB** shell written in **[Solod](https://github.com/lmangani/solod)** (So): the same Go subset the `solod` compiler translates to C, linked against **libduckdb**. This repo is both a small end-user CLI and a demo of [`solod.dev/so/duckdb`](https://github.com/lmangani/solod/tree/main/so/duckdb).

Design goal: stay close in spirit to the [official DuckDB CLI](https://duckdb.org/docs/current/clients/cli/overview) (dot-commands, lightweight SQL keyword hints), without pretending to match every feature (no embedded readline; “autocomplete” is `.complete` / prefix lists).

## Submodule layout

Checked out under the `solod` tree as a **git submodule** (temporary, for co-development):

```bash
git submodule update --init soloduck
```

`go.mod` uses `replace solod.dev => ../` so imports resolve to the parent checkout.

## Build

Requires [libduckdb](https://duckdb.org/install/?environment=c) (headers + shared or static library), same as any So program using `so/duckdb`.

```bash
make          # translate with `so translate`, then cc + -lduckdb
./soloduck -version
./soloduck -e 'SELECT version();'
```

Options:

| Flag | Meaning |
|------|--------|
| `-db path` | Database file or `:memory:` (default). |
| `-e SQL` | Run one statement and exit (non-interactive). |
| `-version` | Print `duckdb_library_version()` and exit. |

Interactive hints:

- End statements with `;` (multi-line input is accumulated until a line ends with `;`).
- `.help`, `.tables`, `.schema [table]`, `.mode line|csv`, `.complete [prefix]`.

## Static binaries

So emits plain C; “static” is a **linker** concern. DuckDB ships `libduckdb_static.a` on many platforms; on **Linux** you can experiment with `-static` and the static archive (often easiest with **musl**). **macOS** rarely produces a portable fully static binary; prefer a normal dynamic link against `libduckdb` or codesign an app bundle.

## License

Same as the parent project you combine with (Solod + your choice for this repo).
