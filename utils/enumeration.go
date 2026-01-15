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
)

func TestKeyServicePair(apiKey string, service string, referrer string) (bool, error) {

	host, _ := url.Parse(service)
	hostname := host.Hostname()

	authenticatedDiscovery := AppendAPIKeyToURL(service, apiKey)
	// sharedClient := GetClient()

	// TODO : Move all error reqs to a retry pool to be executed after the initial batch with exponential+jitter
	req, _ := http.NewRequest("HEAD", authenticatedDiscovery, nil)
	req.Header.Add("Host", hostname)
	req.Header.Add("X-HTTP-Method-Override", "GET") // Documented in https://docs.cloud.google.com/apis/docs/system-parameters - otherwise, it 404s :)
	if referrer != "" {
		req.Header.Add("Referer", referrer)
	}

	headRequest, err := ReqHeaderOnly(*req, false)

	if err != nil {
		log.Printf("Failed to make HEAD request (with X-HTTP-Method-Override: GET): %v", err)
		return false, err
	}

	headRequest.Body.Close()

	if headRequest.StatusCode == 404 {
		// Nothing is unusual with this - i think theres only one that returns 404 when there really isnt a discovery doc.
		// For the ones without a discovery doc, I'll work on contextless GETs. later.
		// TODO: Contextless GET edgecases for non-discoverable services
	}

	return headRequest.StatusCode == 200, nil
}

// WIP - just need to figure out how to use this damn thing
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
