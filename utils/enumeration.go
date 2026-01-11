package fireblazer

import (
	"log"
	"net/http"
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
	req, err := http.NewRequest("HEAD", authenticatedDiscovery, nil)
	req.Header.Add("X-HTTP-Method-Override", "GET") // Documented in https://docs.cloud.google.com/apis/docs/system-parameters - otherwise, it 404s :)

	// log.Println("Created request, starting round trip")
	headRequest, err := sharedClient.Transport.RoundTrip(req)

	// log.Printf("Req finished, err handling %v", service)
	if err != nil {
		log.Printf("Failed to make GET request: %v", err)
		return false, err
	}
	// log.Printf("Req finished, closing body %v", service)
	headRequest.Body.Close()
	// log.Printf("closed body %v", service)

	if headRequest.StatusCode == 404 {
		// Nothing is unusual with this - i think theres only one that returns 404 when there really isnt a discovery doc.
		// For the ones without a discovery doc, I'll work on contextless GETs. later.
		// TODO: Contextless GET edgecases for non-discoverable services
		// body, _ := io.ReadAll(headRequest.Body)
		// log.Printf("Response body for %s: %s", authenticatedDiscovery, string(body))
	}

	// log.Printf("GET request to %s returned status code %d", authenticatedDiscovery, headRequest.StatusCode)
	return headRequest.StatusCode == 200, nil
}
