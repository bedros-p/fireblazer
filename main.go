package main

import (
	utils "fireblazer/m/utils"
	"flag"
	"log"
	"slices"
	"sync"
	"time"
)

var key = flag.String("apiKey", "", "API key to scan")

type APIDetails struct {
	Description string
	Title       string
}

func main() {
	flag.Parse()

	falsePos := []string{
		"https://digitalassetlinks.googleapis.com/$discovery/rest?version=v1",
		"https://www.googleapis.com/discovery/v1/apis/oauth2/v2/rest",
		"https://servicecontrol.googleapis.com/$discovery/rest?version=v2",
		"https://storage.googleapis.com/$discovery/rest?version=v1",
		"https://servicecontrol.googleapis.com/$discovery/rest",
		"https://storage.googleapis.com/$discovery/rest",
	}
	// begin authless discovery endpoint pickup and collect results afterwards, waitgroup
	var wg sync.WaitGroup
	gapiServices := make([]string, 0)
	serviceDetailMap := make(map[string]APIDetails)

	wg.Add(1)

	go func() {
		var err error
		discoveryEndpoints, err := utils.GetDiscoveryEndpoints()

		if err != nil {
			log.Printf("Failed to get discovery endpoints: %v", err)
		}

		for _, endpoint := range discoveryEndpoints {
			gapiServices = append(gapiServices, endpoint.DiscoveryRestUrl)
			serviceDetailMap[endpoint.DiscoveryRestUrl] = APIDetails{
				Description: endpoint.Description,
				Title:       endpoint.Title,
			}
		}

		gapiEndpoints, err := utils.GetEndpointsFromGapis()
		if err != nil {
			log.Printf("Failed to get supplementary endpoints from Github: %v", err)
		}

		for _, endpoint := range gapiEndpoints {
			discoveryUrl := "https://" + endpoint.Host + "/$discovery/rest"
			if !slices.Contains(gapiServices, discoveryUrl) {
				gapiServices = append(gapiServices, discoveryUrl)
				serviceDetailMap[discoveryUrl] = APIDetails{
					Description: endpoint.Description,
					Title:       endpoint.Title,
				}
			}
		}
		wg.Done()
	}()

	if *key == "" {
		*key = flag.Arg(0)
		if *key == "" {
			log.Fatal("You must provide an API key. You can pass it as a named flag or as a positional flag. Usage samples: \n - \"fireblaze AIza-key\" \n - \"fireblaze --key=AIza-key\". \nTerminating.")
		}
	}

	if !utils.TestKeyValidity(*key) {
		log.Fatal("Invalid API key.")
	}

	// If valid go ahead and enumerate all enabled services

	// Collect results and process them
	wg.Wait()
	log.Println("Successfully retrieved discovery endpoints.")
	failCount := 0

	discoveryWg := sync.WaitGroup{}
	foundServices := make([]string, 0)

	for _, item := range gapiServices {
		discoveryWg.Add(1)
		go func(item string) {
			defer discoveryWg.Done()
			// log.Printf("Testing %v", item.DiscoveryRestUrl)
			if valid, err := utils.TestKeyServicePair(*key, item); valid {
				foundServices = append(foundServices, item)
				log.Printf("Found discovery endpoint: %s", item)
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item, err)
				failCount++
			}
		}(item)
		time.Sleep(10 * time.Millisecond) // slight delay to avoid overwhelming the client
	}
	discoveryWg.Wait()
	log.Println("APIs available to this API key:")

	for _, service := range foundServices {
		if slices.Contains(falsePos, service) {
			// Commented out - I only need to have them here as a reminder, dw, just so i know i should work on those.
			// log.Printf(" - %s (false positive)\n\t - %s - %s", service, serviceDetailMap[service].Description, serviceDetailMap[service].Title)
		} else {
			log.Printf(" - %s \n\t - %s - %s", service, serviceDetailMap[service].Description, serviceDetailMap[service].Title)
		}
	}

	log.Printf("All discovery endpoint tests completed with %d failures.", failCount)

}
