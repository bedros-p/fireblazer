package fireblazer

import (
	"context"
	"crypto/tls"
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
	h3Mutex          sync.Mutex
	sharedTransports = make(map[string]*quic.Transport)
	h3ConnMap        = make(map[string]*http3.ClientConn)
)

func getSharedH3Conn(ctx context.Context, customTransport *http3.Transport, hostname string, apiKey string, useActualResolvedName bool) (*http3.ClientConn, error) {
	h3Mutex.Lock()
	defer h3Mutex.Unlock()

	var destAddr string
	if useActualResolvedName {
		destAddr = hostname + ":443"
	} else {
		destAddr = "googleapis.com:443"
	}

	connKey := apiKey + "|" + destAddr

	if conn, ok := h3ConnMap[connKey]; ok {
		return conn, nil
	}

	resolvedRemote, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		return nil, err
	}

	raddrStr := resolvedRemote.IP.String()

	tr, ok := sharedTransports[raddrStr]
	if !ok {
		resolvedHost, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
		if err != nil {
			log.Println("Failed to resolve local address & port for binding. Try running as admin.")
		}
		host, err := net.ListenUDP("udp", resolvedHost)
		if err != nil {
			return nil, err
		}
		tr = &quic.Transport{
			Conn: host,
		}
		sharedTransports[raddrStr] = tr
	}

	dialer, err := tr.DialEarly(ctx, resolvedRemote, customTransport.TLSClientConfig, customTransport.QUICConfig)
	if err != nil {
		return nil, err
	}

	conn := customTransport.NewClientConn(dialer)

	tHandshake := TrackTime("QUIC Handshake to " + destAddr)
	<-dialer.HandshakeComplete()
	tHandshake()

	h3ConnMap[connKey] = conn
	return conn, nil
}

// For handling errors with a retry for the connection stream itself - otherwise i'd be limited to retrying the domain name resolution / dial
func ReqHeaderOnly(req http.Request, apiKey string, useActualResolvedName bool) (*http.Response, error) {
	hostname := req.URL.Hostname()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	customTransport := GetClient().Transport.(*http3.Transport)
	
	tConn := TrackTime("getSharedH3Conn for " + hostname)
	conn, err := getSharedH3Conn(ctx, customTransport, hostname, apiKey, useActualResolvedName)
	tConn()

	if err != nil {
		if useActualResolvedName {
			log.Printf("Couldn't dial service %v even when resolving with the proper domain.", hostname)
			return nil, err
		} else {
			log.Printf("Failed to dial service %v resolved from googleapis.com", hostname)
			log.Println("Retrying with proper raddr")
			return ReqHeaderOnly(req, apiKey, true)
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
		connKey := apiKey + "|" + destAddr
		delete(h3ConnMap, connKey)
		h3Mutex.Unlock()

		if !useActualResolvedName {
			log.Printf("Failed to open stream to service %v via googleapis.com. Retrying with proper raddr", hostname)
			return ReqHeaderOnly(req, apiKey, true)
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
			return ReqHeaderOnly(req, apiKey, true)
		}
		log.Printf("Failed to read response from stream %v - %v", stream, err)
		return nil, err
	}

	return resp, nil

}
