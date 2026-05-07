package replfd

//so:embed replfd.h
var replfd_h string

//so:embed replfd.c
var replfd_c string

//so:extern
func replfdReadRaw(fd int, buf *byte, n int) (int, error) {
	_, _, _ = fd, buf, n
	return 0, nil
}

//so:extern
func replfd_fflush_stdout()
