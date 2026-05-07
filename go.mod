module github.com/lmangani/soloduck

go 1.26

require solod.dev v0.0.0

// Stock Solod (no DuckDB package): https://github.com/solod-dev/solod
// The DuckDB C shim lives in ./duckdb in this repository.
//
// Layout A — submodule soloduck inside a solod checkout:
replace solod.dev => ../

// Layout B — sibling clones solod/ + soloduck/ (uncomment):
// replace solod.dev => ../solod
//
// Layout B build: make SOLID=../solod
