module github.com/lmangani/soloduck

go 1.26

require solod.dev v0.0.0

// Vendored compiler + stdlib (see ./solod submodule → github.com/solod-dev/solod).
// DuckDB bindings live in ./duckdb in this repo (not upstream Solod).
replace solod.dev => ./solod
