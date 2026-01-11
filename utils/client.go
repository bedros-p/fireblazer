package fireblazer

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
)

var GetClient = sync.OnceValue(func() *http.Client {
	return &http.Client{
		Transport: &http3.Transport{
			EnableDatagrams: true,
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
