package main

import (
	utils "fireblazer/m/utils"
	"flag"
	"log"
	"sync"
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
	for _, item := range discoveryEndpoints {
		log.Printf("Found discovery endpoint: %s", item.DiscoveryRestUrl)
	}

}
