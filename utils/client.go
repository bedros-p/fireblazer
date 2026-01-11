package fireblazer

import (
	"context"
	"log"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
)

var GetClient = sync.OnceValue(func() *http.Client {
	// tr := &quic.Transport{}

	return &http.Client{
		// Timeouts and all will be as long as they are right now until a (planned) retry pool is implemented
		Transport: &http3.Transport{
			// TLSClientConfig: &tls.Config{},
			// QUICConfig:      &quic.Config{},
			// Dial: func(ctx context.Context, addr string, tlsConf *tls.Config, quicConf *quic.Config) (*quic.Conn, error) {
			// 	log.Printf("----Beginning Resolve----\nctx\t%v\nresolvedRemote.String()\t%v\nTLSClientConfig\t%v\nQUICConfig\t%v\n----Dial Info End----", ctx, addr, tlsConf, quicConf)
			// 	resolvedAddr, err := net.ResolveUDPAddr("udp", addr)

			// 	resolvedHost, _ := net.ResolveUDPAddr("udp", "0.0.0.0:0")
			// 	host, _ := net.ListenUDP("udp", resolvedHost)

			// 	resolvedRemote, err := net.ResolveUDPAddr("udp", hostname+":443")
			// 	if err != nil {
			// 		return nil, err
			// 	}
			// 	log.Printf("----Beginning tr DialEarly----\nctx\t%v\nresolvedAddr\t%v\ntlsConf\t%v\nquicConf\t%v\n----Dial Info End----", ctx, resolvedAddr, tlsConf, quicConf)

			// 	return tr.Dial(ctx, resolvedAddr, tlsConf, quicConf)
			// },
			EnableDatagrams: true,
		},
		Timeout: 5 * time.Second,
	}
})

func ReqWithBackoff(url string, client *http.Client) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := range 5 {
		resp, err = client.Get(url)
		if err == nil {
			return resp, nil
		}
		time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
	}

	return nil, err
}

// In case I want to pursue this route later.
func ReqHeaderOnly(req http.Request) *http.Response {
	hostname := req.URL.Hostname()
	ctx, cancel := context.WithCancel(context.Background())

	// resolvedHost, _ := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	// host, _ := net.ListenUDP("udp", resolvedHost)

	resolvedRemote, err := net.ResolveUDPAddr("udp", hostname+":443")

	if err != nil {
		log.Printf("Failed to resolve host %s", hostname)
	}

	customTransport := GetClient().Transport.(*http3.Transport)
	log.Println(customTransport.QUICConfig)

	log.Printf("----Beginning Dial----\nctx\t%v\nresolvedRemote.String()\t%v\nTLSClientConfig\t%v\nQUICConfig\t%v\n----Dial Info End----", ctx, hostname, customTransport.TLSClientConfig, customTransport.QUICConfig)
	dialer, err := customTransport.Dial(ctx, resolvedRemote.String(), http3.ConfigureTLSConfig(nil), customTransport.QUICConfig)

	// dialer, err := quic.DialEarly(ctx, host, resolvedRemote, customTransport.TLSClientConfig, customTransport.QUICConfig) // No protocol, raw quic. Maybe later. Like way later.

	if err != nil {
		log.Printf("Failed to dial raddr %v", hostname)
		log.Printf("%v----", err)
		// log.Panic(err)
	}

	conn := customTransport.NewClientConn(dialer)
	log.Printf("Conn state : %v", conn.Conn().ConnectionState())
	conn.Conn().HandshakeComplete()

	stream, err := conn.OpenRequestStream(ctx)

	if err != nil {
		log.Println("Failed to open request stream at raddr %v", resolvedRemote)
	}

	conn.RoundTrip(&req)
	resp, err := stream.ReadResponse()
	if err != nil {
		log.Printf("Failed to read response from stream %v - %v", stream, err)
	}
	defer cancel()
	return resp

}
