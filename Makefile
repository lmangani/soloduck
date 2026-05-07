# Build soloduck: translate with Solod (So), then link against libduckdb.
#
# Requires:
#   ./solod (git submodule update --init, or clone repo with --recurse-submodules)
#   libduckdb (headers + library): https://duckdb.org/install/?environment=c
#
#   make        # translate + link
#   make clean  # removes gen/ and binary only (keeps ./solod submodule)
#
# Override DuckDB install prefix if needed:
#   make DUCK_PREFIX=/path/to/prefix

SOLID ?= ./solod
GEN ?= gen
DUCK_PREFIX ?= $(shell brew --prefix duckdb 2>/dev/null)
RPATH ?= -Wl,-rpath,$(DUCK_PREFIX)/lib
CSRCS = $(shell find $(GEN) -name '*.c' 2>/dev/null)

.PHONY: translate clean build all

all: build

translate:
	@test -d "$(SOLID)/cmd/so" || (echo "Missing Solod checkout. Run: git submodule update --init"; exit 1)
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
