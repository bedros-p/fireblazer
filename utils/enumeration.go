package fireblazer

import (
	"io"
	"log"
)

func TestKeyServicePair(apiKey string, service string) (bool, error) {
	// this whole program relies on the fact that discovery rest urls are guaranteed to be there on every PUBLIC Google service endpoint.
	// this is NOT a design flaw
	// it checks if the key has access to that specific service / if its enabled in the gcp project associated with the key
	// this cant be avoided by any reasonable measure - if this isn't done, one could easily test it with each service with service-specific requests.
	// especially with AI tooling!!! it would be trivial to go through all endpoint Discovery docs and have AI pick out the requests that dont require project specific info.
	// if a service doesnt have a generic endpoint (one that doesnt require project specific info), nothing can be done anyways.
	// all it changes is it hides the services you cant do anything with ANYWAYS, so whatever
	// i should probably write that in the readme

	authenticatedDiscovery := AppendAPIKeyToURL(service, apiKey)
	sharedClient := GetClient()

	// TODO : Move all error reqs to a retry pool to be executed after the initial batch with exponential+jitter
	headRequest, err := sharedClient.Get(authenticatedDiscovery)

	if err != nil {
		log.Printf("Failed to make GET request: %v", err)
		return false, err
	}

	defer headRequest.Body.Close()

	if headRequest.StatusCode == 404 {
		body, _ := io.ReadAll(headRequest.Body)
		log.Printf("Response body for %s: %s", authenticatedDiscovery, string(body))
	}

	log.Printf("GET request to %s returned status code %d", authenticatedDiscovery, headRequest.StatusCode)
	return headRequest.StatusCode == 200, nil
}
