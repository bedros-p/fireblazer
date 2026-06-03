package fireblazer

import (
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

var KeyLogFile os.File

// TODO: experiment with quic-go params https://quic-go.net/docs/quic/flowcontrol/#configuring-limits
// If we're consistently sending 439 endpoint discovery reqs and expecting all those responses all the time, we might be able to tweak it a little...
// But at this stage it's lowk a gamble ?

var StoredResolvedAddr *net.UDPAddr

var GetClient = sync.OnceValue(func() *http.Client {
	// TODO: Buildtime / runtime flags to enable snooping. I want a dev branch one day that has all sorts of logging eventually but idw pollute the code yet
	// KeyLogFile, _ := os.Create("ssl_keys.log") // If you want to read the traffic and debug issues with Wireshark, uncomment this.

	return &http.Client{
		Transport: &http3.Transport{
			EnableDatagrams: true,
			TLSClientConfig: &tls.Config{
				// InsecureSkipVerify: true,
				// KeyLogWriter:       KeyLogFile,
				ServerName: "googleapis.com",
				NextProtos: []string{http3.NextProtoH3},
			},
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
				hostAddr, _ := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
				listener, err := net.ListenUDP("udp", hostAddr)

				if err != nil {
					log.Printf("Failed to listen on local port - try raising ulimit? Error: %v", err)
				}

				var udpAddr *net.UDPAddr
				udpAddr, err = net.ResolveUDPAddr("udp", addr)

				if err != nil {
					log.Printf("Failed to resolve %s", addr)
					return nil, err
				}

				StoredResolvedAddr = udpAddr

				return quic.Dial(ctx, listener, udpAddr, tlsCfg, cfg)
			},
		},
		Timeout: 20 * time.Second,
	}
})

func ReqWithBackoff(req *http.Request, client *http.Client) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := range 5 {
		resp, err = client.Do(req)
		if err == nil {
			return resp, nil
		}
		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
	}

	return nil, err
}

var (
	h3Mutex           sync.Mutex
	sharedTransports  = make(map[string]*quic.Transport)
	h3ConnMap         = make(map[string]*http3.ClientConn)
	CachedGoogleApiIp string
)

func getSharedH3Conn(ctx context.Context, customTransport *http3.Transport, hostname string, poolIndex int, useActualResolvedName bool) (*http3.ClientConn, error) {
	var destAddr string
	var actualIp string
	if useActualResolvedName {
		destAddr = hostname + ":443"
		actualIp = destAddr
	} else {
		destAddr = "googleapis.com:443"
		// Protect the IP cache check/set
		h3Mutex.Lock()
		if CachedGoogleApiIp == "" {
			resolved, err := net.ResolveUDPAddr("udp", destAddr)
			if err != nil {
				h3Mutex.Unlock()
				return nil, err
			}
			CachedGoogleApiIp = resolved.IP.String() + ":443"
		}
		actualIp = CachedGoogleApiIp
		h3Mutex.Unlock()
	}

	connKey := fmt.Sprintf("%d|%s", poolIndex, destAddr)

	// Check connection cache safely
	h3Mutex.Lock()
	if conn, ok := h3ConnMap[connKey]; ok {
		h3Mutex.Unlock()
		return conn, nil
	}
	h3Mutex.Unlock() // Unlock for network IO!

	resolvedRemote, err := net.ResolveUDPAddr("udp", actualIp)
	if err != nil {
		return nil, err
	}

	raddrStr := resolvedRemote.IP.String()

	// Protect transport map safely
	h3Mutex.Lock()
	tr, ok := sharedTransports[raddrStr]
	if !ok {
		resolvedHost, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
		if err != nil {
			log.Println("Failed to resolve local address & port for binding. Try running as admin.")
		}
		host, err := net.ListenUDP("udp", resolvedHost)
		if err != nil {
			h3Mutex.Unlock()
			return nil, err
		}
		tr = &quic.Transport{
			Conn: host,
		}
		sharedTransports[raddrStr] = tr
	}
	h3Mutex.Unlock() // Unlock for network IO!

	// Perform the blocking dial concurrently!
	dialer, err := tr.DialEarly(ctx, resolvedRemote, customTransport.TLSClientConfig, customTransport.QUICConfig)
	if err != nil {
		return nil, err
	}

	conn := customTransport.NewClientConn(dialer)

	tHandshake := TrackTime("QUIC Handshake to " + destAddr)
	<-dialer.HandshakeComplete()
	tHandshake()

	// Safely insert the completed connection
	h3Mutex.Lock()
	// Double-check someone else didn't beat us to it while we were dialing
	if existingConn, ok := h3ConnMap[connKey]; ok {
		h3Mutex.Unlock()
		// no conn.Close() for http3.ClientConn, let it drop
		return existingConn, nil
	}
	h3ConnMap[connKey] = conn
	h3Mutex.Unlock()

	return conn, nil
}

// For handling errors with a retry for the connection stream itself - otherwise i'd be limited to retrying the domain name resolution / dial
func ReqHeaderOnly(req http.Request, poolIndex int, useActualResolvedName bool) (*http.Response, error) {
	hostname := req.URL.Hostname()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	customTransport := GetClient().Transport.(*http3.Transport)

	tConn := TrackTime("getSharedH3Conn for " + hostname)
	conn, err := getSharedH3Conn(ctx, customTransport, hostname, poolIndex, useActualResolvedName)
	tConn()

	if err != nil {
		if useActualResolvedName {
			log.Printf("Couldn't dial service %v even when resolving with the proper domain.", hostname)
			return nil, err
		} else {
			log.Printf("Failed to dial service %v resolved from googleapis.com", hostname)
			log.Println("Retrying with proper raddr")
			return ReqHeaderOnly(req, poolIndex, true)
		}
	}

	stream, err := conn.OpenRequestStream(ctx)
	if err != nil { // i think this handling is really hacky mannnnn like i ran into some unreproducible errors where it fails. I need to test on diff network speeds and see if the address gets changed by google if it takes too long
		h3Mutex.Lock()
		var destAddr string
		if useActualResolvedName {
			destAddr = hostname + ":443"
		} else {
			destAddr = "googleapis.com:443"
		}
		connKey := fmt.Sprintf("%d|%s", poolIndex, destAddr)
		delete(h3ConnMap, connKey)
		h3Mutex.Unlock()

		if !useActualResolvedName {
			log.Printf("Failed to open stream to service %v via googleapis.com. Retrying with proper raddr", hostname)
			return ReqHeaderOnly(req, poolIndex, true)
		}
		return nil, err
	}

	err = stream.SendRequestHeader(&req)
	if err != nil {
		log.Printf("Failed to send request header to stream %v", err)
	}

	stream.SetDeadline(time.Now().Add(10 * time.Second))
	resp, err := stream.ReadResponse()
	if err != nil {
		if !useActualResolvedName {
			log.Printf("Failed to read response from stream %v - %v. Retrying with proper raddr.", stream, err)
			return ReqHeaderOnly(req, poolIndex, true)
		}
		log.Printf("Failed to read response from stream %v - %v", stream, err)
		return nil, err
	}

	return resp, nil

}

// WarmupConnections concurrently establishes QUIC connections to googleapis.com to prevent mutex deadlocks and DNS latency during the concurrent scan.
func WarmupConnections(ctx context.Context, numConns int) {
	log.Printf("Warming up %d QUIC connection(s) to googleapis.com...", numConns)
	customTransport := GetClient().Transport.(*http3.Transport)

	var warmupGroup errgroup.Group
	for i := 0; i < numConns; i++ {
		i := i // capture loop variable for goroutine
		warmupGroup.Go(func() error {
			_, err := getSharedH3Conn(ctx, customTransport, "googleapis.com", i, false)
			if err != nil {
				log.Printf("Failed to warmup connection %d: %v", i, err)
			}
			return nil
		})
	}
	warmupGroup.Wait()
}
