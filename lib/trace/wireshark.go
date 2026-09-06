//go:build wireshark

package trace

import (
	"flag"
	"io"
	"log"
	"os"
)

var (
	keylogPath       string
	wiresharkEnabled bool
	keyLogFile       *os.File
)

func init() {
	flag.StringVar(&keylogPath, "keylog", "", "Path to write TLS keylog file for Wireshark decryption (also respects SSLKEYLOGFILE env var).")
	flag.BoolVar(&wiresharkEnabled, "wireshark", false, "Enable TLS key logging to ssl_keys.log for Wireshark inspection.")
}

func KeyLogWriter() io.Writer {
	target := keylogPath
	if target == "" && wiresharkEnabled {
		target = "ssl_keys.log"
	}
	if target == "" {
		target = os.Getenv("SSLKEYLOGFILE")
	}
	if target == "" {
		return nil
	}

	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Printf("Failed to open keylog file %s: %v", target, err)
		return nil
	}
	keyLogFile = f
	return f
}

func Close() error {
	if keyLogFile != nil {
		return keyLogFile.Close()
	}
	return nil
}
