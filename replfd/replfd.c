//go:build ignore

#include "replfd.h"
#include <stdio.h>
#include <unistd.h>
#include "so/builtin/builtin.h"
#include "so/io/io.h"

void replfd_fflush_stdout(void) {
    (void)fflush(stdout);
}

so_R_int_err replfdReadRaw(so_int fd, so_byte* buf, so_int n) {
    if (n <= 0 || buf == NULL) {
        return (so_R_int_err){.val = 0, .err = NULL};
    }
    ssize_t r = read((int)fd, buf, (size_t)n);
    if (r < 0) {
        return (so_R_int_err){.val = 0, .err = io_ErrUnexpectedEOF};
    }
    if (r == 0) {
        return (so_R_int_err){.val = 0, .err = io_EOF};
    }
    return (so_R_int_err){.val = (so_int)r, .err = NULL};
}
