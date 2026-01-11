package main

import (
	"context"
	utils "fireblazer/utils"
	"flag"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/yarlson/pin"
)

var key = flag.String("apiKey", "", "API key to scan")
var dangerouslySkipVerification = flag.Bool("dangerouslySkipVerification", false, "Skip API key verification")
var outputFormat = flag.String("outputFormat", "interactive", "Output format (interactive|text|json|yaml)")
var outputDetails = flag.String("outputDetails", "full", "Comma delimited list of what to include in the details (description|title|name). Comma delimited.")
var isInteractive = *outputFormat == "interactive" || *outputFormat == ""

type APIDetails struct {
	Description string
	Title       string
}

type Service struct {
	CleanName    string
	DiscoveryUrl string
}

func main() {
	flag.Parse()

	falsePos := []string{
		"digitalassetlinks",
		"oauth2",
		"servicecontrol",
		"storage",
		"servicecontrol",
		"storage",
	}
	// begin authless discovery endpoint pickup and collect results afterwards, waitgroup
	var wg sync.WaitGroup
	gapiServices := make([]Service, 0)
	serviceDetailMap := make(map[string]APIDetails)

	wg.Add(1)

	discoveryFailed := 0
	go func() {
		var err error
		discoveryEndpoints, err := utils.GetDiscoveryEndpoints()

		if err != nil {
			if isInteractive || *outputFormat == "text" {
				log.Printf("Failed to get discovery endpoints: %v", err)
				discoveryFailed++
			}
		}

		for _, endpoint := range discoveryEndpoints {

			// utils.PreresolveHost(endpoint.DiscoveryRestUrl)

			gapiServices = append(gapiServices, Service{
				CleanName:    endpoint.Name,
				DiscoveryUrl: endpoint.DiscoveryRestUrl,
			})

			serviceDetailMap[endpoint.Name] = APIDetails{
				Description: endpoint.Description,
				Title:       endpoint.Title,
			}
		}

		gapiEndpoints, err := utils.GetEndpointsFromGapis()
		if err != nil {
			if isInteractive || *outputFormat == "text" {
				log.Printf("Failed to get supplementary endpoints from Github: %v", err)
				discoveryFailed++
			} else if discoveryFailed > 0 {
				// TODO: Local-first approach for endpoints
				log.Fatal("Both primary and supplementary endpoint retrieval failed, terminating.")
			}
		}

		for _, endpoint := range gapiEndpoints {
			cleanName := strings.Split(endpoint.Host, ".")[0]

			discoveryUrl := "https://" + endpoint.Host + "/$discovery/rest"
			// utils.PreresolveHost(discoveryUrl) // it gets stripped down again later - its fine. Rather not have 2 functions to resolve.

			if serviceDetailMap[cleanName].Title == "" {
				serviceDetailMap[cleanName] = APIDetails{
					Description: endpoint.Description,
					Title:       endpoint.Title,
				}

				gapiServices = append(gapiServices, Service{
					CleanName:    cleanName,
					DiscoveryUrl: discoveryUrl,
				})
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
	if *dangerouslySkipVerification {
		if isInteractive || *outputFormat == "text" {
			log.Println("Skipping API key verification.")
		}
	} else if valid, err := utils.TestKeyValidity(*key); !valid {
		log.Fatalf(`Invalid API key. 
If you're sure the key is valid, use the --dangerouslySkipVerification flag [fireblazer --dangerouslySkipVerification AIza-KeYHere]
And submit an issue at https://github.com/bedros-p/fireblazer - include this error message:

%v

----
`, err)
	} else {
		if isInteractive || *outputFormat == "text" {
			log.Println("Valid API key, proceeding.")
		}
	}

	// If valid go ahead and enumerate all enabled services

	// Collect results and process them
	wg.Wait()
	scanPin := pin.New("Scanning...")

	if isInteractive || *outputFormat == "text" {
		log.Printf("Successfully retrieved %d discovery endpoints - %d endpoint sources failed.", len(gapiServices), discoveryFailed)
		if discoveryFailed > 0 { // TODO: Local-first approach for endpoints
			log.Println("A number of endpoint sources failed.\n If there's a consistently failing source and the following domains aren't blocked, raise an issue at https://github.com/bedros-p/fireblazer :")
			log.Println("- www.googleapis.com \n- raw.githubusercontent.com")
		}
		if isInteractive {
			cancel := scanPin.Start(context.Background())
			defer cancel()
		}
	}

	failCount := 0

	discoveryWg := sync.WaitGroup{}
	foundServices := make([]string, 0)
	rem := len(gapiServices)
	foundCount := 0 // idw to repeatedly check the length of foundServices

	for _, item := range gapiServices {
		discoveryWg.Add(1)
		go func(item Service) {
			// 8 Services in scope. %d left...
			// baseMessage :=
			defer discoveryWg.Done()
			// log.Printf("Testing %v", item)
			if valid, err := utils.TestKeyServicePair(*key, item.DiscoveryUrl); valid {
				foundCount++
				foundServices = append(foundServices, item.CleanName)

				// log.Printf("Found discovery endpoint: %s", item)
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item, err)
				failCount++
			}

			rem--
			scanPin.UpdateMessage(fmt.Sprintf("%d services in scope. Scanning %d more...", foundCount, rem))

		}(item)
		// time.Sleep(1 * time.Millisecond) // slight delay to avoid overwhelming the client. QUIC seems to cope and burn with no delay.
	}
	discoveryWg.Wait()
	scanPin.Stop("Scan complete!")
	log.Println("APIs available to this API key:")

	for _, service := range foundServices {
		if slices.Contains(falsePos, service) {
			// Commented out - I only need to have them here as a reminder, dw, just so i know i should work on those.
			// log.Printf(" - %s (false positive)\n\t - %s - %s", service, serviceDetailMap[service].Description, serviceDetailMap[service].Title)
		} else {
			log.Printf(" - %s.googleapis.com / %s \n\t - %s", service, serviceDetailMap[service].Title, serviceDetailMap[service].Description)
		}
	}

	log.Printf("All discovery endpoint tests completed with %d failures.", failCount)

}
