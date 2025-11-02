package log

import (
	"fmt"
	"os"
)

func Warn(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
}

func Warnf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, format, v...)
	fmt.Fprintln(os.Stderr)
}
