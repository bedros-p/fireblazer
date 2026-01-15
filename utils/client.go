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

var StoredResolvedAddr *net.UDPAddr

var GetClient = sync.OnceValue(func() *http.Client {
	// KeyLogFile, _ := os.Create("ssl_keys.log") // If you want to read the traffic and debug issues with Wireshark, uncomment this.

	return &http.Client{
		Transport: &http3.Transport{
			EnableDatagrams: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
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

// For handling errors with a retry for the connection stream itself - otherwise i'd be limited to retrying the domain name resolution / dial
func ReqHeaderOnly(req http.Request, useActualResolvedName bool) (*http.Response, error) {
	hostname := req.URL.Hostname()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resolvedHost, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		log.Println("Failed to resolve local address & port for binding. Try running as admin.")
	}
	host, _ := net.ListenUDP("udp", resolvedHost)

	resolvedRemote, err := net.ResolveUDPAddr("udp", "googleapis.com:443")
	if err != nil {
		if useActualResolvedName {
			log.Printf("Failed to resolve host %s - ensure you're connected to the internet & can resolve domain names.", hostname)
		} else {
			log.Println("Failed to resolve googleapis.com - falling back to proper execution..")
		}
	}

	customTransport := GetClient().Transport.(*http3.Transport)

	dialer, err := quic.DialEarly(ctx, host, resolvedRemote, customTransport.TLSClientConfig, customTransport.QUICConfig) // No protocol, raw quic. Maybe later. Like way later.
	if err != nil {
		if useActualResolvedName {
			log.Printf("Couldn't dial service %v even when resolving with the proper domain.", hostname)
			return nil, err
		} else {
			log.Printf("Failed to dial service %v resolved from googleapis.com", hostname)
			log.Println("Retrying with proper raddr")
			return ReqHeaderOnly(req, true)
		}
	}

	conn := customTransport.NewClientConn(dialer)
	conn.Conn().HandshakeComplete()

	stream, err := conn.OpenRequestStream(ctx)
	stream.SendRequestHeader(&req)

	if err != nil {
		log.Printf("Failed to open request stream at raddr %v", resolvedRemote)
	}

	resp, err := stream.ReadResponse()
	if err != nil {
		log.Printf("Failed to read response from stream %v - %v", stream, err)
		return nil, err
	}

	return resp, nil

}
