package fireblazer

import (
	"encoding/json"
	"log"
)

// Some APIs are missing (e.g generativelanguage). No idea how to counter it. Maybe the googleapis github / their protos would have more services named.
const discoveryEndpoint = "https://discovery.googleapis.com/discovery/v1/apis"

// {
//     "kind": "discovery#directoryItem",
//     "id": "abusiveexperiencereport:v1",
//     "name": "abusiveexperiencereport",
//     "version": "v1",
//------     "title": "Abusive Experience Report API",
//------     "description": "Views Abusive Experience Report data, and gets a list of sites that have a significant number of abusive experiences.",
//------     "discoveryRestUrl": "https://abusiveexperiencereport.googleapis.com/$discovery/rest?version=v1",
//     "icons": {
//         "x16": "https://www.gstatic.com/images/branding/product/1x/googleg_16dp.png",
//         "x32": "https://www.gstatic.com/images/branding/product/1x/googleg_32dp.png"
//     },
//-----     "documentationLink": "https://developers.google.com/abusive-experience-report/",
//     "preferred": true
// }

// Querying a discovery doc with an API key that doesnt use that service returns an error code, with 7 services being exempt

// We don't have to include every field if unused, we'll just use the stuff we want
type DiscoveryItem struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	DiscoveryRestUrl  string `json:"discoveryRestUrl"`
	DocumentationLink string `json:"documentationLink"`
	Name              string `json:"name"`
}

type discoveryListing struct {
	Items []DiscoveryItem `json:"items"`
}

// This returns a list of discovery endpoints
func GetDiscoveryEndpoints() ([]DiscoveryItem, error) {
	resp, err := GetClient().Get(discoveryEndpoint)
	if err != nil {
		// TODO: Create a persistent back up
		log.Fatal(err)
		return nil, err
	}

	// use decode instead of unmarshaling since we have a stream
	var discoveryListResponse discoveryListing
	if err := json.NewDecoder(resp.Body).Decode(&discoveryListResponse); err != nil {
		log.Fatalf("Failed to decode JSON %v", err)
		return nil, err
	}

	seen := make(map[string]struct{})
	var deduplicated []DiscoveryItem
	for _, item := range discoveryListResponse.Items {
		if _, ok := seen[item.Name]; !ok {
			seen[item.Name] = struct{}{}
			deduplicated = append(deduplicated, item)
		}
	}
	return deduplicated, nil
}
