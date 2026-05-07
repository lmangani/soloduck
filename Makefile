# Build soloduck: translate with Solod (So), then link against libduckdb.
#
# Requires a Solod checkout whose main branch includes so/duckdb (see README).
#
# Layout A (submodule): default SOLID=..  → parent solod repo root.
# Layout B (sibling):   make SOLID=../solod
#
#   make        # translate + link
#   make clean
#
# If Homebrew DuckDB is not on PATH, set:
#   make DUCK_PREFIX=/path/to/duckdb
#
# Static linking: DuckDB distribute libduckdb_static.a; on Linux you can try
#   make LDFLAGS='-static -lduckdb_static ...'
# after consulting the DuckDB C installation docs. macOS rarely supports
# fully static executables the same way.

SOLID ?= ..
GEN ?= gen
DUCK_PREFIX ?= $(shell brew --prefix duckdb 2>/dev/null)
RPATH ?= -Wl,-rpath,$(DUCK_PREFIX)/lib
CSRCS = $(shell find $(GEN) -name '*.c' 2>/dev/null)

.PHONY: translate clean build all

all: build

translate:
	@mkdir -p $(GEN)
	cd $(SOLID) && go run ./cmd/so translate -o "$(CURDIR)/$(GEN)" "$(CURDIR)"

build: translate
	@test -n "$(DUCK_PREFIX)" || (echo "Install libduckdb or set DUCK_PREFIX."; exit 1)
	@test -n "$(CSRCS)" || (echo "No generated C sources under $(GEN); translate failed?"; exit 1)
	$(CC) -O2 -g -std=gnu11 -Wall -Wextra -Werror -Wno-shadow \
		-I$(CURDIR)/$(GEN) -I$(DUCK_PREFIX)/include \
		$(CSRCS) -o soloduck \
		-L$(DUCK_PREFIX)/lib -lduckdb $(RPATH) $(LDFLAGS) -lm

clean:
	rm -rf $(GEN) soloduck
