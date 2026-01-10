package fireblazer

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
)

var sharedClient *http.Client

var GetClient = sync.OnceValue(func() *http.Client {
	return &http.Client{
		// Timeouts and all will be as long as they are right now until a (planned) retry pool is implemented
		Transport: &http3.Transport{
			EnableDatagrams: true,
		},
		Timeout: 30 * time.Second,
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
