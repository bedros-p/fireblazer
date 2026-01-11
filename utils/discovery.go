package fireblazer

import (
	"encoding/json"
	"log"
)

// Setting preferred=true makes it so that every service has one entry instead of multiple for the different supported versions
const discoveryEndpoint = "https://discovery.googleapis.com/discovery/v1/apis?preferred=true"

type DiscoveryItem struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	DiscoveryRestUrl string `json:"discoveryRestUrl"`
	Version          string `json:"version"`
	Name             string `json:"name"`
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

	var discoveryListResponse discoveryListing
	if err := json.NewDecoder(resp.Body).Decode(&discoveryListResponse); err != nil {
		log.Fatalf("Failed to decode JSON %v", err)
		return nil, err
	}

	return discoveryListResponse.Items, nil
}
