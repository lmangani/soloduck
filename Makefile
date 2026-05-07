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
#
# Static link: use DuckDB’s “static-libs” release zip (all lib*.a under lib/ + duckdb.h).
#   make DUCK_LINK_STATIC=1 DUCK_PREFIX=/path/to/prefix
# See: https://github.com/duckdb/duckdb/releases (static-libs-*.zip)

SOLID ?= ./solod
GEN ?= gen
DUCK_PREFIX ?= $(shell brew --prefix duckdb 2>/dev/null)
DUCK_LINK_STATIC ?= 0
RPATH ?= -Wl,-rpath,$(DUCK_PREFIX)/lib
CSRCS = $(shell find $(GEN) -name '*.c' 2>/dev/null)
UNAME_S := $(shell uname -s)
# All vendored archives (DuckDB splits symbols across many .a files).
DUCK_STATIC_ARCHIVES = $(sort $(wildcard $(DUCK_PREFIX)/lib/lib*.a))

ifeq ($(DUCK_LINK_STATIC),1)
  DUCK_RPATH :=
  ifeq ($(UNAME_S),Darwin)
    DUCK_LD_LIBS := $(foreach lib,$(DUCK_STATIC_ARCHIVES),-Wl,-force_load,$(lib))
    DUCK_SYS_LIBS := -lm -lz -lc++ -lpthread -framework CoreFoundation -framework SystemConfiguration
  else
    DUCK_LD_LIBS := -Wl,--start-group $(DUCK_STATIC_ARCHIVES) -Wl,--end-group
    DUCK_SYS_LIBS := -lm -lpthread -ldl -lstdc++ -lz
  endif
else
  DUCK_RPATH := $(RPATH)
  DUCK_LD_LIBS := -L$(DUCK_PREFIX)/lib -lduckdb
  DUCK_SYS_LIBS := -lm
endif

.PHONY: translate clean build all

all: build

translate:
	@test -d "$(SOLID)/cmd/so" || (echo "Missing Solod checkout. Run: git submodule update --init"; exit 1)
	@mkdir -p $(GEN)
	cd $(SOLID) && go run ./cmd/so translate -o "$(CURDIR)/$(GEN)" "$(CURDIR)"

build: translate
	@test -n "$(DUCK_PREFIX)" || (echo "Install libduckdb or set DUCK_PREFIX."; exit 1)
	@test -n "$(CSRCS)" || (echo "No generated C sources under $(GEN); translate failed?"; exit 1)
	@if [ "$(DUCK_LINK_STATIC)" = 1 ] && [ -z "$(DUCK_STATIC_ARCHIVES)" ]; then \
		echo "DUCK_LINK_STATIC=1 but no $(DUCK_PREFIX)/lib/lib*.a (use DuckDB static-libs-*.zip)."; exit 1; \
	fi
	$(CC) -O2 -g -std=gnu11 -Wall -Wextra -Werror -Wno-shadow -fno-strict-aliasing \
		-I$(CURDIR)/$(GEN) -I$(DUCK_PREFIX)/include \
		$(CSRCS) -o soloduck \
		$(LDFLAGS) $(DUCK_LD_LIBS) $(DUCK_RPATH) $(DUCK_SYS_LIBS)

clean:
	rm -rf $(GEN) soloduck
