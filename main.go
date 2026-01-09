package main

import (
	utils "fireblazer/m/utils"
	"flag"
	"log"
	"sync"
	"time"
)

var key = flag.String("apiKey", "", "API key to scan")

func main() {
	flag.Parse()
	// begin authless discovery endpoint pickup and collect results afterwards, waitgroup
	var wg sync.WaitGroup
	var discoveryEndpoints []utils.DiscoveryItem

	wg.Add(1)

	go func() {
		defer wg.Done()
		var err error
		discoveryEndpoints, err = utils.GetDiscoveryEndpoints()
		if err != nil {
			log.Printf("Failed to get discovery endpoints: %v", err)
		}
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

	discoveryWg := sync.WaitGroup{}
	for _, item := range discoveryEndpoints {
		discoveryWg.Add(1)
		go func(item utils.DiscoveryItem) {
			defer discoveryWg.Done()
			// log.Printf("Testing %v", item.DiscoveryRestUrl)
			if valid, err := utils.TestKeyServicePair(*key, item); valid {
				log.Printf("Found discovery endpoint: %s", item.DiscoveryRestUrl)
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item.DiscoveryRestUrl, err)
			}
		}(item)
		time.Sleep(30 * time.Millisecond) // slight delay to avoid overwhelming the client
	}
	discoveryWg.Wait()
	log.Println("All discovery endpoint tests completed.")

}
