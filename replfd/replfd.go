// Package replfd provides stdin input via read(2) for interactive CLIs built with Solod.
//
// Stock solod.dev/so/os uses FILE* + fread, which can fully-buffer tty reads so bufio
// does not see lines until the stdio buffer fills. Wrapping fd 0 with [Stdin] avoids that.
//
// Use [Stdin] as the io.Reader for bufio.NewReader in REPL mode only; pipes and -batch
// can keep using [os.Stdin].
package replfd

import "unsafe"

// Stdin is an io.Reader over file descriptor 0 implemented with read(2).
type Stdin struct {
	// pad avoids invalid `{0}` initializers for empty structs in generated C.
	pad uint8
}

// singleton avoids zero-initializer quirks for empty structs in translated C.
var singleton Stdin

// Ptr returns the shared [*Stdin] used by the interactive REPL.
func Ptr() *Stdin {
	return &singleton
}

// FlushStdout flushes libc stdout so prompts appear before read(2) blocks (stdio buffering).
func FlushStdout() {
	replfd_fflush_stdout()
}

func stdinRead(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return replfdReadRaw(0, unsafe.SliceData(p), len(p))
}

// Read implements [io.Reader] for interactive terminal input.
// Pointer receiver is required so Solod emits an io.Reader compatible with bufio (void* self).
func (*Stdin) Read(p []byte) (int, error) {
	return stdinRead(p)
}
