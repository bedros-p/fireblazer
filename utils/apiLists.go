package fireblazer

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
)

// Alternative Source for Discovery Doc
// Deduplicated, reaches around 240. Discovery doc, deduplicated, reaches around 300. And yet both have services the other doesnt.
// This would fill the gap to be as comprehensive as possible. Combined and deduplicated, you reach ~380 hostnames.

var apiListGithub = "https://raw.githubusercontent.com/googleapis/googleapis/refs/heads/master/api-index-v1.json"

type GapisApiItem struct {
	Description string `json:"description"`
	Title       string `json:"title"`
	Host        string `json:"hostname"`
	Version     string `json:"majorVersion"` // This would be the preferred version from what I can tell
}

type GapisContainer struct {
	Apis []GapisApiItem `json:"apis"`
}

func GetEndpointsFromGapis() ([]GapisApiItem, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// InsecureSkipVerify: true, // If you want to intercept the traffic with mitmproxy
			},
		},
	} // Github doesn't seem to like QUIC - using regular client

	body, err := client.Get(apiListGithub)
	if err != nil {
		// TODO: Local fallback
		log.Fatalf("Error fetching supplementary Gapis API list: %v", err)
		return nil, err
	}

	var apiList GapisContainer
	if err := json.NewDecoder(body.Body).Decode(&apiList); err != nil {
		log.Fatalf("Error decoding Gapis API list: %v", err)
		return nil, err
	}
	return apiList.Apis, nil
}
