#pragma once

#include "so/builtin/builtin.h"
#include "so/io/io.h"

// replfdReadRaw reads up to n bytes into buf using read(2).
so_R_int_err replfdReadRaw(so_int fd, so_byte* buf, so_int n);

void replfd_fflush_stdout(void);
