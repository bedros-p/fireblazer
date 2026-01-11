package fireblazer

import (
	"log"
	"net/http"
)

func TestKeyServicePair(apiKey string, service string) (bool, error) {

	authenticatedDiscovery := AppendAPIKeyToURL(service, apiKey)
	sharedClient := GetClient()

	// TODO : Move all error reqs to a retry pool to be executed after the initial batch with exponential+jitter
	req, err := http.NewRequest("HEAD", authenticatedDiscovery, nil)
	req.Header.Add("X-HTTP-Method-Override", "GET") // Documented in https://docs.cloud.google.com/apis/docs/system-parameters - otherwise, it 404s :)

	headRequest, err := sharedClient.Transport.RoundTrip(req)

	if err != nil {
		log.Printf("Failed to make GET request: %v", err)
		return false, err
	}

	headRequest.Body.Close()

	if headRequest.StatusCode == 404 {
		// Nothing is unusual with this - i think theres only one that returns 404 when there really isnt a discovery doc.
		// For the ones without a discovery doc, I'll work on contextless GETs. later.
		// TODO: Contextless GET edgecases for non-discoverable services
	}

	return headRequest.StatusCode == 200, nil
}
