module github.com/lmangani/soloduck

go 1.26

require solod.dev v0.0.0

// solod.dev is the module path for github.com/lmangani/solod (or your fork).
// Point "replace" at a checkout that includes so/duckdb on main.
//
// Layout A — submodule inside solod (replace points at repo root):
replace solod.dev => ../

// Layout B — sibling directories solod/ and soloduck/ (uncomment this instead):
// replace solod.dev => ../solod
//
// For layout B, build with: make SOLID=../solod
