package fireblazer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type TargetKey struct {
	Raw         string
	Key         string
	Referrer    string
	IosBundleId string
	AndroidPkg  string
	AndroidCert string
}

func TestKeyServicePair(target TargetKey, service string, useGet bool) (bool, error) {

	host, _ := url.Parse(service)
	hostname := host.Hostname()

	authenticatedDiscovery := AppendAPIKeyToURL(service, target.Key)
	// sharedClient := GetClient()

	// TODO : Move all error reqs to a retry pool to be executed after the initial batch with exponential+jitter
	method := "HEAD"
	if useGet {
		method = "GET"
	}

	req, _ := http.NewRequest(method, authenticatedDiscovery, nil)
	req.Header.Add("Host", hostname)
	if !useGet {
		req.Header.Add("X-HTTP-Method-Override", "GET") // Documented in https://docs.cloud.google.com/apis/docs/system-parameters - otherwise, it 404s :)
	}

	if target.Referrer != "" {
		req.Header.Add("Referer", target.Referrer)
	}
	if target.IosBundleId != "" {
		req.Header.Add("X-Ios-Bundle-Identifier", target.IosBundleId)
	}
	if target.AndroidPkg != "" {
		req.Header.Add("X-Android-Package", target.AndroidPkg)
	}
	if target.AndroidCert != "" {
		req.Header.Add("X-Android-Cert", target.AndroidCert)
	}

	headRequest, err := ReqHeaderOnly(*req, target.Raw, false)

	if err != nil {
		log.Printf("Failed to make request to %s: %v", service, err)
		return false, err
	}

	headRequest.Body.Close()

	return headRequest.StatusCode == 200, nil
}

// WIP - just need to figure out how to use this damn thing
// I was hoping to be able to multipart it and send multiple services with one big payload. But I might just remove this. Http3 lets me send the data in one big stream, there's no need for this anymore.
// Still, it has potential for other uses if I keep digging.
func MultipartAllDiscoveries(apiKey string, cleannames []string) (map[string]bool, error) {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)

	for i, service := range cleannames {
		// host, _ := url.Parse(service)

		field := make(textproto.MIMEHeader)
		field.Add("Host", service)

		field.Add("Content-ID", fmt.Sprintf("%d:23923944", i))
		field.Add("Content-Type", "application/http")
		part, _ := w.CreatePart(field)
		// part.Write([]byte("GET /$discovery/rest" + apiKey))

		// part.Write([]byte("GET /discovery/" + strings.Split(host.Hostname(), ".")[0] + "/apis?key="+ apiKey))
		// part.Write([]byte("GET /discovery/v1/apis/" + service + "/v1/rest?key=" + apiKey))
		part.Write([]byte("GET /apis/" + service + "/v1/rest?key=" + apiKey))
	}
	// https://discovery.googleapis.com/discovery/v1/apis/abusiveexperiencereport/v1/rest
	w.Close()
	body, err := io.ReadAll(&buf)
	log.Println(string(body))
	// return nil, nil
	req, _ := http.NewRequest("GET", "https://discovery.googleapis.com/batch/discovery/v1", &buf)
	req.Header.Set("Content-Type", "multipart/mixed; boundary="+w.Boundary())

	client := GetClient()

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Something broke %v", err)
	}
	defer resp.Body.Close()
	bodyContent, _ := io.ReadAll(resp.Body)

	log.Printf(string(bodyContent))

	return nil, nil
}

type Service struct {
	CleanName    string
	DiscoveryUrl string
}

type ElapsedCombo struct {
	ServiceClean string
	TimeElapsed  int64
}

type ScanUpdate struct {
	Key           string
	WasFound      bool
	ItemCleanName string
}

// This function is maybe my finest work, but the decrement thing might be better off for interactive mode handled in main.go?
// Either way, all functioons should strive to be as clear-cut like this. Does its thing and thats it.
func ScanServices(target TargetKey, gapiServices []Service, workerCount int, timingEnabled bool, updateCh chan<- ScanUpdate, useGet bool) ([]string, int, *ElapsedCombo) {
	defer TrackTime("ScanServices total")()
	var maxTimeMutex sync.Mutex
	maxTime := &ElapsedCombo{
		ServiceClean: "",
		TimeElapsed:  0,
	}

	var scanGroup errgroup.Group
	scanGroup.SetLimit(workerCount)

	var foundMutex sync.Mutex
	foundServices := make([]string, 0)

	var failMutex sync.Mutex
	failCount := 0

	for _, item := range gapiServices {
		scanGroup.Go(func() error {
			defer TrackWorkerSampleTime("Worker " + item.CleanName)()
			var start time.Time
			if timingEnabled {
				start = time.Now()
			}

			wasFound := false
			if valid, err := TestKeyServicePair(target, item.DiscoveryUrl, useGet); valid {
				foundMutex.Lock()
				foundServices = append(foundServices, item.CleanName)
				foundMutex.Unlock()
				wasFound = true
			} else if err != nil {
				log.Printf("Error testing discovery endpoint %s: %v", item.CleanName, err)
				failMutex.Lock()
				failCount++
				failMutex.Unlock()
			}

			if timingEnabled {
				elapsed := time.Since(start).Milliseconds()
				maxTimeMutex.Lock()
				if elapsed > maxTime.TimeElapsed {
					maxTime = &ElapsedCombo{
						ServiceClean: item.CleanName,
						TimeElapsed:  elapsed,
					}
				}
				maxTimeMutex.Unlock()
			}

			if updateCh != nil {
				select {
				case updateCh <- ScanUpdate{
					Key:           target.Raw,
					WasFound:      wasFound,
					ItemCleanName: item.CleanName,
				}:
				default:
				}
			}
			return nil
		})
	}

	scanGroup.Wait()

	return foundServices, failCount, maxTime
}
