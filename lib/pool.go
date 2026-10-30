package fireblazer

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// GFE advertises initial_max_streams_bidi = 100 on every QUIC connection.
// A single connection therefore cannot keep more than 100 requests in flight.
const streamsPerConn = 100

const (
	maxPoolConns = 250 // 25k instant streams; more just adds handshake cost
	minPoolConns = 4
	maxPoolPorts = 4
)

// H3ConnectionPool is a fixed set of pre-warmed HTTP/3 connections to
// googleapis.com. Requests for every scanned key share it, so a batch pays
// for handshakes once instead of once per key, and no single connection is
// limited to 100 concurrent streams.
type H3ConnectionPool struct {
	conns      []*rawH3Conn
	quicConns  []*quic.Conn
	transports []*quic.Transport
	sockets    []*net.UDPConn
	next       atomic.Uint64
}

func (p *H3ConnectionPool) GetNextConn() *rawH3Conn {
	if p == nil || len(p.conns) == 0 {
		return nil
	}
	idx := (p.next.Add(1) - 1) % uint64(len(p.conns))
	return p.conns[idx]
}

func (p *H3ConnectionPool) Close() {
	if p == nil {
		return
	}
	for _, c := range p.quicConns {
		if c != nil {
			c.CloseWithError(0, "pool closed")
		}
	}
	for _, tr := range p.transports {
		if tr != nil {
			tr.Close()
		}
	}
	for _, s := range p.sockets {
		if s != nil {
			s.Close()
		}
	}
}

// CalculatePoolTopology sizes the pool so that the whole request set fits in
// one credit window when possible.
func CalculatePoolTopology(totalRequests int) (conns, ports int) {
	if totalRequests < 1 {
		totalRequests = 1
	}
	conns = (totalRequests + streamsPerConn - 1) / streamsPerConn
	if conns > maxPoolConns {
		conns = maxPoolConns
	}
	if conns < minPoolConns {
		conns = minPoolConns
	}

	ports = (conns + 24) / 25 // ~25 connections per UDP socket
	if ports > maxPoolPorts {
		ports = maxPoolPorts
	}
	if ports > runtime.NumCPU() {
		ports = runtime.NumCPU()
	}
	if ports < 1 {
		ports = 1
	}
	return conns, ports
}

var (
	globalPool   *H3ConnectionPool
	globalPoolMu sync.Mutex
)

// GetH3ConnectionPool returns the process-wide pool, creating it on first use.
func GetH3ConnectionPool(totalRequests int) (*H3ConnectionPool, error) {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()
	if globalPool != nil {
		return globalPool, nil
	}
	p, err := NewH3ConnectionPool(totalRequests)
	if err != nil {
		return nil, err
	}
	globalPool = p
	return p, nil
}

func resolveGoogleIPv4() (*net.UDPAddr, error) {
	var lastErr error
	for i := range 4 {
		addr, err := net.ResolveUDPAddr("udp4", "googleapis.com:443")
		if err == nil {
			return addr, nil
		}
		lastErr = err
		time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
	}
	return nil, lastErr
}

// NewH3ConnectionPool dials `conns` QUIC connections over a small number of
// UDP sockets and waits for all handshakes to complete.
func NewH3ConnectionPool(totalRequests int) (*H3ConnectionPool, error) {
	conns, ports := CalculatePoolTopology(totalRequests)

	udpAddr, err := resolveGoogleIPv4()
	if err != nil {
		return nil, err
	}

	tlsConf := &tls.Config{
		ServerName:         "googleapis.com",
		NextProtos:         []string{http3.NextProtoH3},
		ClientSessionCache: tls.NewLRUClientSessionCache(conns),
	}
	quicConf := &quic.Config{
		HandshakeIdleTimeout: 10 * time.Second,
	}

	pool := &H3ConnectionPool{
		transports: make([]*quic.Transport, ports),
		sockets:    make([]*net.UDPConn, ports),
	}

	for i := range ports {
		sock, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
		if err != nil {
			pool.Close()
			return nil, err
		}
		_ = sock.SetReadBuffer(7 * 1024 * 1024)
		_ = sock.SetWriteBuffer(7 * 1024 * 1024)
		pool.sockets[i] = sock
		pool.transports[i] = &quic.Transport{Conn: sock}
	}

	type dialResult struct {
		idx  int
		conn *quic.Conn
	}
	results := make(chan dialResult, conns)
	var dialWg sync.WaitGroup
	for i := range conns {
		dialWg.Add(1)
		go func(idx int) {
			defer dialWg.Done()
			tr := pool.transports[idx%ports]
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c, err := tr.DialEarly(ctx, udpAddr, tlsConf, quicConf)
			if err != nil {
				results <- dialResult{idx: idx}
				return
			}
			select {
			case <-c.HandshakeComplete():
			case <-ctx.Done():
				c.CloseWithError(0, "handshake timeout")
				results <- dialResult{idx: idx}
				return
			}
			results <- dialResult{idx: idx, conn: c}
		}(i)
	}
	dialWg.Wait()
	close(results)

	ordered := make([]*quic.Conn, conns)
	ok := 0
	for r := range results {
		if r.conn != nil {
			ordered[r.idx] = r.conn
			ok++
		}
	}
	if ok == 0 {
		pool.Close()
		return nil, errNoPoolConns
	}

	pool.conns = make([]*rawH3Conn, 0, ok)
	pool.quicConns = make([]*quic.Conn, 0, ok)
	for _, c := range ordered {
		if c == nil {
			continue
		}
		raw, err := newRawH3Conn(c)
		if err != nil {
			c.CloseWithError(0, "raw HTTP/3 init failed")
			continue
		}
		pool.conns = append(pool.conns, raw)
		pool.quicConns = append(pool.quicConns, c)
	}
	if len(pool.conns) == 0 {
		pool.Close()
		return nil, errNoPoolConns
	}
	if ok < conns || len(pool.conns) < ok {
		log.Printf("HTTP/3 pool: %d/%d connections established, continuing with %d", len(pool.conns), conns, len(pool.conns))
	}
	return pool, nil
}

type poolError string

func (e poolError) Error() string { return string(e) }

const errNoPoolConns poolError = "failed to establish any HTTP/3 connection to googleapis.com"
