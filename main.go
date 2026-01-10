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

func main() {
	flag.Parse()
	// begin authless discovery endpoint pickup and collect results afterwards, waitgroup
	var wg sync.WaitGroup
	gapiServices := make([]string, 0)

	wg.Add(1)

	go func() {
		var err error
		discoveryEndpoints, err := utils.GetDiscoveryEndpoints()

		if err != nil {
			log.Printf("Failed to get discovery endpoints: %v", err)
		}

		for _, endpoint := range discoveryEndpoints {
			gapiServices = append(gapiServices, endpoint.DiscoveryRestUrl)
		}

		gapiEndpoints, err := utils.GetEndpointsFromGapis()
		if err != nil {
			log.Printf("Failed to get supplementary endpoints from Github: %v", err)
		}

		for _, endpoint := range gapiEndpoints {
			if !slices.Contains(gapiServices, "https://"+endpoint.Host+"/$discovery/rest") {
				gapiServices = append(gapiServices, "https://"+endpoint.Host+"/$discovery/rest")
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
	for _, item := range gapiServices {
		discoveryWg.Add(1)
		go func(item string) {
			defer discoveryWg.Done()
			// log.Printf("Testing %v", item.DiscoveryRestUrl)
			if valid, err := utils.TestKeyServicePair(*key, item); valid {
				log.Printf("Found discovery endpoint: %s", item)
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item, err)
				failCount++
			}
		}(item)
		time.Sleep(30 * time.Millisecond) // slight delay to avoid overwhelming the client
	}
	discoveryWg.Wait()
	log.Printf("All discovery endpoint tests completed with %d failures.", failCount)

}
