package fireblazer

import (
	"net/http"
	"sync"
	"time"
)

var sharedClient *http.Client

var GetClient = sync.OnceValue(func() *http.Client {
	return &http.Client{
		// Timeouts and all will be as long as they are right now until a (planned) retry pool is implemented
		Transport: &http.Transport{
			MaxConnsPerHost:       1000,
			IdleConnTimeout:       60 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
		},
		Timeout: 30 * time.Second,
	}
})

// GetClient is now a function that returns a *http.Client.
// The internal logic is only executed the very first time it is called.
