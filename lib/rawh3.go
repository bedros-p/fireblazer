package fireblazer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/quic-go/qpack"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/quicvarint"
	"golang.org/x/net/http2/hpack"
)

// rawH3Conn speaks HTTP/3 directly over a QUIC connection. It sends one
// HEADERS frame per request and parses only :status from the response, which
// avoids the http3.ClientConn/RequestStream/Response/MIMEHeader machinery and
// its per-request allocations.
type rawH3Conn struct {
	conn *quic.Conn
	ctrl *quic.SendStream
	enc  *quic.SendStream
	dec  *quic.SendStream
}

func newRawH3Conn(c *quic.Conn) (*rawH3Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Client control stream: stream type 0x00, then an empty SETTINGS frame.
	// Empty SETTINGS means QPACK dynamic table capacity 0 and 0 blocked streams.
	ctrl, err := c.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := ctrl.Write([]byte{0x00, 0x04, 0x00}); err != nil {
		return nil, err
	}

	// QPACK encoder and decoder streams. With capacity 0 they stay empty.
	enc, err := c.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := enc.Write([]byte{0x02}); err != nil {
		return nil, err
	}
	dec, err := c.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Write([]byte{0x03}); err != nil {
		return nil, err
	}

	return &rawH3Conn{conn: c, ctrl: ctrl, enc: enc, dec: dec}, nil
}

func readQUICVarint(r io.Reader) (uint64, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	n := 1 << (b[0] >> 6)
	v := uint64(b[0] & 0x3f)
	if n == 1 {
		return v, nil
	}
	var rest [8]byte
	if _, err := io.ReadFull(r, rest[:n-1]); err != nil {
		return 0, err
	}
	for i := range n - 1 {
		v = v<<8 | uint64(rest[i])
	}
	return v, nil
}

// readStatus reads the first frame on a response stream and extracts :status.
// HEAD responses carry a single HEADERS frame; anything else is an error.
func readStatus(s *quic.Stream, scratch *[]byte) (int, error) {
	typ, err := readQUICVarint(s)
	if err != nil {
		return 0, err
	}
	if typ != 0x01 {
		return 0, fmt.Errorf("http3: expected HEADERS frame, got 0x%x", typ)
	}
	length, err := readQUICVarint(s)
	if err != nil {
		return 0, err
	}
	if length > 1<<20 {
		return 0, fmt.Errorf("http3: HEADERS frame too large: %d", length)
	}
	if cap(*scratch) < int(length) {
		*scratch = make([]byte, length)
	}
	block := (*scratch)[:length]
	if _, err := io.ReadFull(s, block); err != nil {
		return 0, err
	}
	return parseStatus(block)
}

var errDynamicQPACK = errors.New("qpack: dynamic table reference with capacity 0")

// Static-table indices that encode :status directly (RFC 9204 Appendix A).
var staticStatus = map[uint64]int{
	24: 103, 25: 200, 26: 304, 27: 404, 28: 503,
	63: 100, 64: 204, 65: 206, 66: 302,
	67: 400, 68: 403, 69: 421, 70: 425, 71: 500,
}

const staticStatusNameIndex = 24

func readPrefixedInt(b []byte, prefix uint8) (uint64, int, error) {
	if len(b) == 0 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	mask := byte(1<<prefix - 1)
	v := uint64(b[0] & mask)
	if v < uint64(mask) {
		return v, 1, nil
	}
	var shift uint
	for i := 1; i < len(b); i++ {
		x := b[i]
		v += uint64(x&0x7f) << shift
		shift += 7
		if x&0x80 == 0 {
			return v, i + 1, nil
		}
	}
	return 0, 0, io.ErrUnexpectedEOF
}

func readQpackString(b []byte) (string, int, error) {
	if len(b) == 0 {
		return "", 0, io.ErrUnexpectedEOF
	}
	huffman := b[0]&0x80 != 0
	length, n, err := readPrefixedInt(b, 7)
	if err != nil {
		return "", 0, err
	}
	if length > uint64(len(b)-n) {
		return "", 0, io.ErrUnexpectedEOF
	}
	raw := b[n : n+int(length)]
	if huffman {
		s, err := hpack.HuffmanDecodeToString(raw)
		return s, n + int(length), err
	}
	return string(raw), n + int(length), nil
}

// parseStatus decodes a QPACK field section that uses only the static table.
func parseStatus(block []byte) (int, error) {
	if len(block) < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	reqInsert, n, err := readPrefixedInt(block, 8)
	if err != nil {
		return 0, err
	}
	if n >= len(block) {
		return 0, io.ErrUnexpectedEOF
	}
	deltaBase, dn, err := readPrefixedInt(block[n:], 7)
	if err != nil {
		return 0, err
	}
	if reqInsert != 0 || deltaBase != 0 {
		return 0, errDynamicQPACK
	}
	rest := block[n+dn:]

	for len(rest) > 0 {
		b := rest[0]
		switch {
		case b&0x80 != 0: // Indexed Field Line
			if b&0x40 == 0 {
				return 0, errDynamicQPACK
			}
			idx, used, err := readPrefixedInt(rest, 6)
			if err != nil {
				return 0, err
			}
			rest = rest[used:]
			if status, ok := staticStatus[idx]; ok {
				return status, nil
			}

		case b&0x40 != 0: // Literal Field Line with Name Reference
			idx, used, err := readPrefixedInt(rest, 4)
			if err != nil {
				return 0, err
			}
			rest = rest[used:]
			if b&0x10 == 0 { // N bit: never index; T bit is 0x10 here
				// T=0 means dynamic table name reference.
				return 0, errDynamicQPACK
			}
			value, used, err := readQpackString(rest)
			if err != nil {
				return 0, err
			}
			rest = rest[used:]
			if idx == staticStatusNameIndex {
				return parseStatusCode(value)
			}

		case b&0x20 != 0: // Literal Field Line with Literal Name
			name, used, err := readQpackString(rest)
			if err != nil {
				return 0, err
			}
			rest = rest[used:]
			value, used, err := readQpackString(rest)
			if err != nil {
				return 0, err
			}
			rest = rest[used:]
			if name == ":status" {
				return parseStatusCode(value)
			}

		default:
			return 0, errDynamicQPACK
		}
	}
	return 0, errors.New("qpack: response has no :status")
}

func parseStatusCode(s string) (int, error) {
	var code int
	if len(s) != 3 {
		return 0, fmt.Errorf("qpack: bad :status %q", s)
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("qpack: bad :status %q", s)
		}
		code = code*10 + int(s[i]-'0')
	}
	return code, nil
}

// encodeRequest writes a single HEADERS frame for a discovery request.
func encodeRequest(enc *qpack.Encoder, buf *bytes.Buffer, out *[]byte, method, scheme, authority, path string, extra []qpack.HeaderField) []byte {
	buf.Reset()
	_ = enc.WriteField(qpack.HeaderField{Name: ":method", Value: method})
	_ = enc.WriteField(qpack.HeaderField{Name: ":scheme", Value: scheme})
	_ = enc.WriteField(qpack.HeaderField{Name: ":authority", Value: authority})
	_ = enc.WriteField(qpack.HeaderField{Name: ":path", Value: path})
	for _, f := range extra {
		_ = enc.WriteField(f)
	}
	_ = enc.Close()

	block := buf.Bytes()
	*out = (*out)[:0]
	*out = quicvarint.Append(*out, 0x01)
	*out = quicvarint.Append(*out, uint64(len(block)))
	*out = append(*out, block...)
	return *out
}
