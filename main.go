package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	utils "github.com/bedros-p/fireblazer/utils"

	"github.com/yarlson/pin"
	"golang.org/x/sync/errgroup"
)

var key = flag.String("apiKey", "", "API key to scan")
var dangerouslySkipVerification = flag.Bool("dangerouslySkipVerification", false, "Skip API key verification")
var outputFormat = flag.String("outputFormat [WIP]", "interactive", "Output format (interactive|text|json|yaml)")
var outputDetails = flag.String("outputDetails [WIP]", "full", "Comma delimited list of what to include in the details (description|title|name). Comma delimited.")
var isInteractive = *outputFormat == "interactive" || *outputFormat == ""

type APIDetails struct {
	Description string
	Title       string
}

type Service struct {
	CleanName    string
	DiscoveryUrl string
}

// Experimenting with building the stream myself to close as soon as response headers are received. Now I've got a workaround for forcing HEAD :)
// I was 100% going to make my own http3 client since I already have a QUIC transport. But a short workaround was found :)

// func main() {
// 	req, err := http.NewRequest("GET", "https://generativelanguage.googleapis.com/$discovery/rest", nil)
// 	if err != nil {
// 		log.Println("Failed to create new request")
// 	}
// 	resp, err := utils.GetClient().Do(req)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	// resp := utils.ReqHeaderOnly(*req)
// 	log.Printf("Resp status round trip : %s \n", resp.Status)
// }

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

	//  Those don't work / hang the program
	blacklisted := []string{
		"poly",
		"lifesciences",
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

	type ElapsedCombo struct {
		serviceClean string
		timeElapsed  int64
	}

	// maxTime := &ElapsedCombo{
	// 	serviceClean: "",
	// 	timeElapsed:  0,
	// }
	rem := len(gapiServices)
	foundCount := 0 // idw to repeatedly check the length of foundServices
	var tasks errgroup.Group
	tasks.SetLimit(100)
	// 100 - 20 seconds
	for _, item := range gapiServices {
		if slices.Contains(slices.Concat(blacklisted, falsePos), item.CleanName) {
			continue
		}
		discoveryWg.Add(1)
		// itemCopy := item
		tasks.Go(func() error {
			// 8 Services in scope. %d left...
			// baseMessage :=
			defer discoveryWg.Done()
			// log.Printf("Testing %v", item)
			// start := time.Now()
			if valid, err := utils.TestKeyServicePair(*key, item.DiscoveryUrl); valid {
				foundCount++
				foundServices = append(foundServices, item.CleanName)
				// log.Printf("Found discovery endpoint: %s", item)
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item, err)
				failCount++
			}

			// elapsed := time.Since(start).Milliseconds()
			// if elapsed > maxTime.timeElapsed {
			// 	maxTime = &ElapsedCombo{
			// 		serviceClean: item.CleanName,
			// 		timeElapsed:  elapsed,
			// 	}
			// }

			rem--
			go scanPin.UpdateMessage(fmt.Sprintf("Service count - %d in scope. Scanning %d more... %v", foundCount, rem, item.CleanName))
			return nil
		})
		// time.Sleep(1 * time.Millisecond) // slight delay to avoid overwhelming the client. QUIC seems to cope and burn with no delay.
	}
	discoveryWg.Wait()

	scanPin.Stop(fmt.Sprintf("Scan complete! Identified %d services available in the project.", foundCount))
	// log.Printf("Elapsed combo - %v\n\n\n", maxTime) // Code for measuring the longest running one
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
