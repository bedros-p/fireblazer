//go:build !wireshark

package trace

import "io"

func KeyLogWriter() io.Writer {
	return nil
}

func Close() error {
	return nil
}
