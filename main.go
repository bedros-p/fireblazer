package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	utils "github.com/bedros-p/fireblazer/utils"

	"github.com/yarlson/pin"
	"golang.org/x/sync/errgroup"
)

var key = flag.String("apiKey", "", "API key to scan. Can also be your first positional arg.")
var dangerouslySkipVerification = flag.Bool("dangerouslySkipVerification", false, "Skip API key verification")
var workerCount = flag.Int("workerCount", 170, "Set the amount of worker threads to spawn for executing the requests")

// interactive|text|json|yaml
var outputFormat = flag.String("outputFormat", "interactive", "[WIP] Output format (interactive|text|json|yaml)")
var outputDetails = flag.String("outputDetails", "full", "[WIP] Comma delimited list of what to include in the details (description|title|name). Comma delimited.")
var isInteractive = *outputFormat == "interactive" || *outputFormat == ""
var timingEnabled = flag.Bool("findSlowService", false, "[DEBUG] Find which service took the longest to test + elapsed time. Use to file an issue for program hangs.")

type APIDetails struct {
	Description string
	Title       string
}

type Service struct {
	CleanName    string
	DiscoveryUrl string
}

var discoverySourcesFailedMsg string = `
A number of endpoint sources failed.
If there's a consistently failing source and the following domains aren't blocked, raise an issue at https://github.com/bedros-p/fireblazer

- www.googleapis.com
- raw.githubusercontent.com
`

func main() {
	flag.Parse()

	// utils.MultipartAllDiscoveries(*key, []string{"generativelanguage.googleapis.com", "discovery.googleapis.com"})
	// return

	falsePos := []string{
		"digitalassetlinks",
		"oauth2",
		"servicecontrol",
		"storage",
	}

	//  Those don't work / hang the program
	blacklisted := []string{
		"poly",
		"lifesciences",
	}

	discoverySourcesLoaded := make(chan struct{})

	gapiServices := make([]Service, 0)
	serviceDetailMap := make(map[string]APIDetails)

	discoveryFailed := 0
	go func() {
		var err error
		discoveryEndpoints, err := utils.GetDiscoveryEndpoints()

		if err != nil {
			if isInteractive || *outputFormat == "text" {
				log.Printf("Failed to get discovery endpoints: %v", err)
				discoveryFailed++
			}
			if *outputFormat == "json" || *outputFormat == "yaml" {
				// TODO: JSON and YAML format error handling
			}
		}

		for _, endpoint := range discoveryEndpoints {

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
		close(discoverySourcesLoaded)
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
		if err != nil {
			log.Fatalf("Error testing API key validity: %v\n. Ensure that you can connect to https://generativelanguage.googleapis.com as it's used for checking key validity. To skip primary validation (at risk of invalid results), use the --dangerouslySkipVerification flag.", err)
		}

		log.Println("Invalid API key.")
		log.Println("If you're sure the key is valid, use the --dangerouslySkipVerification flag [fireblazer --dangerouslySkipVerification AIza-KeYHere]")
		log.Println("And submit an issue at https://github.com/bedros-p/fireblazer - include this error message:\n%v\n----", err)
		os.Exit(-1)
	} else {
		if isInteractive || *outputFormat == "text" {
			log.Println("Valid API key, proceeding.")
		}
	}

	// Need to wait for the discovery services to load before proceeding w the scan
	<-discoverySourcesLoaded

	scanPin := pin.New("Scanning...")

	if isInteractive || *outputFormat == "text" {
		log.Printf("Successfully retrieved %d discovery endpoints - %d endpoint sources failed.", len(gapiServices), discoveryFailed)
		if discoveryFailed > 0 { // TODO: Local-first approach for endpoints
			log.Println(discoverySourcesFailedMsg)
		}
	}

	if isInteractive {
		cancel := scanPin.Start(context.Background())
		defer cancel()
	}

	type ElapsedCombo struct {
		serviceClean string
		timeElapsed  int64
	}

	maxTime := &ElapsedCombo{
		serviceClean: "",
		timeElapsed:  0,
	}

	var scanGroup errgroup.Group
	scanGroup.SetLimit(*workerCount)

	rem := len(gapiServices)

	foundServices := make([]string, 0)
	foundCount := 0 // idw to repeatedly check the length of foundServices

	failCount := 0

	for _, item := range gapiServices {
		if slices.Contains(slices.Concat(blacklisted, falsePos), item.CleanName) {
			continue
		}

		scanGroup.Go(func() error {
			var start time.Time
			if *timingEnabled {
				start = time.Now() // i doubt that this is an expensive operation in ANY way. Still.
			}

			if valid, err := utils.TestKeyServicePair(*key, item.DiscoveryUrl); valid {
				foundCount++
				foundServices = append(foundServices, item.CleanName)
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item, err)
				failCount++
			}

			if *timingEnabled {
				start = time.Now()
				elapsed := time.Since(start).Milliseconds()
				if elapsed > maxTime.timeElapsed {
					maxTime = &ElapsedCombo{
						serviceClean: item.CleanName,
						timeElapsed:  elapsed,
					}
				}
			}

			rem--
			go scanPin.UpdateMessage(fmt.Sprintf("Service count - %d in scope. Scanning %d more... %v", foundCount, rem, item.CleanName))
			return nil
		})
	}

	scanGroup.Wait()

	scanPin.Stop(fmt.Sprintf("Scan complete! Identified %d services available in the project.", foundCount))
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

	if *timingEnabled {
		log.Printf("Longest running service - %v\n\n\n", maxTime)
	}

}
