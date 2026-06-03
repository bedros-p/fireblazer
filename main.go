package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	lib "github.com/bedros-p/fireblazer/lib"

	"github.com/yarlson/pin"

	"github.com/bedros-p/fireblazer/utils"
	"gopkg.in/yaml.v3"
)

// TODO: IAM stuff
// TODO: Print out stuff on the Pin thing and quickly wipe it like "If you can see this, your terminal size is too small or you're viewing it on a terminal that doesn't like interactive modes. Use -outputFormat=text for a better experience :)"
// TODO: Optimize further
// TODO: See if i can pre-process the JSON and bake the struct into the binary. Seems excess, but might work out
// TODO: Firebase full enumeration

var key = flag.String("apiKey", "", "API key to scan. Can also be your first positional arg.")
var referrer = flag.String("referrer", "", "Set the referrer (Referer:) header for when an API key is restricted to a website.")
var dangerouslySkipVerification = flag.Bool("dangerouslySkipVerification", false, "Skip API key verification (Note: This stops Blaze mode from working since it requires a Project ID)")
var workerCount = flag.Int("workerCount", 170, "Set the amount of worker threads to spawn for executing the requests")
var targetApi = flag.String("targetApi", "", "A single API discovery endpoint to test against. Bypasses the full scan.")

// interactive|text|json|yaml
var outputFormat = flag.String("outputFormat", "interactive", "Output format (interactive|text|json|yaml)")
var outputDetails = flag.String("outputDetails", "name", "Comma delimited list of what to include in the details (description|title|name).")
var timingEnabled = flag.Bool("findSlowService", false, "[DEBUG] Find which service took the longest to test + elapsed time. Use to file an issue for program hangs.")
var blaze = flag.Bool("blaze", false, "Enable additional aggressive recon checks (e.g., Brand Identity)")
var maxConnections = flag.Int("maxConnections", 5000, "Maximum number of QUIC connections to pool")
var isInteractive = false

var scanPin = pin.New("Initializing...")
var cancel context.CancelFunc

type APIDetails struct {
	Description string
	Title       string
}

// not too sure how to handle this in the schema without bloating it up, but here's what i think
// might just have some utility `fireblazer describe`. I lowk want to make a sep tool for quick single service surface mapping, it might work better there.

func main() {
	defer lib.TrackTime("Total Execution")()
	flag.Parse()

	detailsMode := *outputDetails
	showTitle := false
	showDesc := false

	for _, p := range strings.Split(detailsMode, ",") {
		p = strings.TrimSpace(p)
		if p == "full" {
			showTitle = true
			showDesc = true
		} else if p == "title" {
			showTitle = true
		} else if p == "description" {
			showDesc = true
		}
	}

	isInteractive = *outputFormat == "interactive" || *outputFormat == ""
	if !*timingEnabled {
		log.SetFlags(0)
	}
	// utils.MultipartAllDiscoveries(*key, []string{"generativelanguage.googleapis.com", "discovery.googleapis.com"})
	// return

	if isInteractive {
		cancel = scanPin.Start(context.Background())
	}
	tLoad := lib.TrackTime("Load Services")
	gapiServices := loadServices(*targetApi)
	tLoad()

	rawKeys := []string{}
	if *key != "" {
		rawKeys = append(rawKeys, *key)
	}
	rawKeys = append(rawKeys, flag.Args()...)

	keys := []string{}
	seenKeys := make(map[string]bool)
	for _, k := range rawKeys {
		if !seenKeys[k] {
			seenKeys[k] = true
			keys = append(keys, k)
		}
	}

	if len(keys) == 0 {
		log.Fatal("You must provide at least one API key. You can pass it as a named flag or as positional arguments. Usage samples: \n - \"fireblazer AIza-key1 AIza-key2\" \n - \"fireblazer --apiKey=AIza-key\". \nTerminating.")
	}

	if isInteractive || *outputFormat == "text" {
		if *targetApi != "" {
			log.Printf("Using single target API: %s", *targetApi)
		} else {
			log.Printf("Successfully loaded %d discovery endpoints from built-in program list.", len(gapiServices))
		}
	}

	var updateCh chan lib.ScanUpdate
	var logCh chan string
	var updateDone chan struct{}

	if isInteractive {
		defer cancel()
		totalServicesCount := len(gapiServices)
		if *blaze {
			totalServicesCount += lib.GetP4SACount()
		}
		updateCh, logCh, updateDone = startInteractiveDisplay(keys, totalServicesCount, *referrer)
	}

	// Calculate optimal connection pool size
	endpointsPerKey := len(gapiServices)
	// Add 1 to ensure integer division rounds up naturally
	connectionsPerKey := (endpointsPerKey + *workerCount - 1) / *workerCount
	numConns := connectionsPerKey * len(keys)

	if numConns > *maxConnections {
		numConns = *maxConnections
	}
	if numConns < 1 {
		numConns = 1
	}

	// Warmup Phase
	if isInteractive || *outputFormat == "text" {
		log.Printf("Calculated optimal QUIC connection pool size: %d", numConns)
	}
	ctxWarmup, cancelWarmup := context.WithTimeout(context.Background(), 15*time.Second)
	lib.WarmupConnections(ctxWarmup, numConns)
	cancelWarmup()

	var wg sync.WaitGroup
	results := make([]utils.KeyResult, len(keys))

	for i, k := range keys {
		wg.Add(1)
		go func(i int, rawKey string) {
			defer wg.Done()
			target := utils.ParseTargetKey(rawKey, *referrer)
			cfg := utils.ProcessConfig{
				DangerouslySkipVerification: *dangerouslySkipVerification,
				IsInteractive:               isInteractive,
				OutputFormat:                *outputFormat,
				WorkerCount:                 *workerCount,
				TimingEnabled:               *timingEnabled,
				UseGet:                      *targetApi != "",
				MaxConns:                    numConns,
			}
			res := utils.ProcessKey(target, gapiServices, updateCh, logCh, cfg)

			if res.Valid && res.ProjectId != "" && *blaze {
				brand, err := lib.GetBrandIdentity(res.ProjectId)
				if err == nil && brand != nil {
					res.Brand = brand
				}

				saServices, err := lib.EnumerateServiceAccounts(res.ProjectId, *workerCount, updateCh, target.Raw)
				if err == nil && len(saServices) > 0 {
					res.P4SAServices = saServices
				}
			}
			results[i] = res
		}(i, k)
	}

	wg.Wait()

	if isInteractive {
		if updateCh != nil {
			close(updateCh)
		}
		if logCh != nil {
			close(logCh)
		}
		<-updateDone
		scanPin.Stop("Scan complete!")
	} else if *outputFormat == "text" {
		log.Println("Scan complete!")
	}

	var outputData []byte
	var err error

	if *outputFormat == "json" {
		outputData, err = json.MarshalIndent(utils.MarshalStructured(results, showTitle, showDesc), "", "  ")
	} else if *outputFormat == "yaml" {
		outputData, err = yaml.Marshal(utils.MarshalStructured(results, showTitle, showDesc))
	} else {
		outputData = utils.MarshalText(results, keys, *targetApi, *timingEnabled, showTitle, showDesc)
	}

	if err != nil {
		log.Fatalf("Error marshaling output: %v", err)
	}

	fmt.Println(string(outputData))

	lib.KeyLogFile.Close()
}

func loadServices(targetApi string) []lib.Service {
	var services []lib.Service

	if targetApi != "" {
		hostname := strings.Split(targetApi, "/")[0]
		cleanName := strings.Split(hostname, ".")[0]
		discoveryUrl := targetApi
		if !strings.HasPrefix(discoveryUrl, "http") {
			discoveryUrl = "https://" + discoveryUrl
		}
		services = append(services, lib.Service{
			CleanName:    cleanName,
			DiscoveryUrl: discoveryUrl,
		})
	} else {
		for _, raw := range lib.GoogleApiList {
			hostname := strings.Split(raw, "/")[0]
			cleanName := strings.Split(hostname, ".")[0]
			services = append(services, lib.Service{
				CleanName:    cleanName,
				DiscoveryUrl: "https://" + raw,
			})
		}
	}
	return services
}

func startInteractiveDisplay(keys []string, totalServices int, globalReferrer string) (chan lib.ScanUpdate, chan string, chan struct{}) {
	updateCh := make(chan lib.ScanUpdate, *workerCount*len(keys))
	logCh := make(chan string, len(keys)*3)
	updateDone := make(chan struct{})

	go func() {
		totalRem := totalServices * len(keys)
		totalFound := 0
		lastScan := ""

		for updateCh != nil || logCh != nil {
			select {
			case update, ok := <-updateCh:
				if !ok {
					updateCh = nil
					continue
				}

				if update.ItemCleanName == "[INVALID]" {
					totalRem -= totalServices
					lastScan = "Invalid Key Skipped"
				} else if update.ItemCleanName == "[SKIP_BLAZE]" {
					totalRem -= lib.GetP4SACount()
					lastScan = "Blaze Skipped (No Project ID)"
				} else {
					totalRem--
					if update.WasFound {
						totalFound++
					}
					lastScan = update.ItemCleanName
				}

				scanPin.UpdateMessage(fmt.Sprintf("Keys %d | Found %d | Rem %d | Scanning %s", len(keys), totalFound, totalRem, lastScan))
			case msg, ok := <-logCh:
				if !ok {
					logCh = nil
					continue
				}
				fmt.Printf("\x1b[2K\r%s\n", msg)
			}
		}
		close(updateDone)
	}()

	return updateCh, logCh, updateDone
}
