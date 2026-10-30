package fireblazer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/sync/errgroup"
)

// Discovery endpoints whose backend never answers. They are skipped so one
// dead service cannot add its deadline to every key in a batch. Verified
// against live GFE on 2026-10-30: each of these hangs for >6 s over both
// HTTP/2 and HTTP/3, while datalabeling/dlp (previously listed) respond.
var HangingDiscoveryHosts = map[string]struct{}{
	"apigeeregistry.googleapis.com":           {},
	"cloudvideosearch.googleapis.com":         {},
	"contentwarehouse.googleapis.com":         {},
	"rapidmigrationassessment.googleapis.com": {},
	"visionai.googleapis.com":                 {},
	"workloadcertificate.googleapis.com":      {},
}

// Endpoints that timed out twice in this process are skipped for the rest of
// the run, so a host that starts hanging mid-batch costs one key, not all.
var autoHangingHosts sync.Map

func IsHangingHost(host string) bool {
	if _, ok := HangingDiscoveryHosts[host]; ok {
		return true
	}
	_, ok := autoHangingHosts.Load(host)
	return ok
}

var timeoutStrikes sync.Map // host -> *atomic.Int32

func noteEndpointTimeout(host string) bool {
	v, _ := timeoutStrikes.LoadOrStore(host, new(atomic.Int32))
	if v.(*atomic.Int32).Add(1) >= 2 {
		if _, loaded := autoHangingHosts.LoadOrStore(host, struct{}{}); !loaded {
			return true
		}
	}
	return false
}

const streamDeadline = 3 * time.Second

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

	log.Printf("%s", string(bodyContent))

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

type preparedEndpoint struct {
	cleanName string
	host      string
	url       url.URL
}

func endpointHost(raw string) string {
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	if h, _, err := net.SplitHostPort(raw); err == nil {
		return h
	}
	return raw
}

// prepareEndpoints parses and filters the catalogue once per key. It also
// appends the API key, so the dispatcher never touches url.Parse or
// url.Values per request.
func prepareEndpoints(target TargetKey, services []Service) []preparedEndpoint {
	key := url.QueryEscape(target.Key)
	out := make([]preparedEndpoint, 0, len(services))
	for _, s := range services {
		raw := s.DiscoveryUrl
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			raw = "https://" + raw
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			continue
		}
		host := u.Hostname()
		if IsHangingHost(host) {
			continue
		}
		query := u.RawQuery
		if query != "" {
			query += "&"
		}
		query += "key=" + key
		path := u.Path
		if path == "" {
			path = "/"
		}
		scheme := u.Scheme
		if scheme == "" {
			scheme = "https"
		}
		out = append(out, preparedEndpoint{
			cleanName: s.CleanName,
			host:      u.Host,
			url:       url.URL{Scheme: scheme, Path: path, RawQuery: query},
		})
	}
	return out
}

type pendingStream struct {
	cleanName string
	host      string
	stream    *http3.RequestStream
	start     time.Time
}

// ScanServices keeps the old signature; it creates the process-wide pool on
// first use.
func ScanServices(target TargetKey, gapiServices []Service, workerCount int, timingEnabled bool, updateCh chan<- ScanUpdate, useGet bool) ([]string, int, *ElapsedCombo) {
	return ScanServicesWithPool(target, gapiServices, workerCount, timingEnabled, updateCh, useGet, nil)
}

// ScanServicesWithPool fires every discovery request without waiting for a
// response, then reads statuses on a separate consumer pool. This keeps the
// wire busy instead of running one RTT per worker round, and it only reads
// response headers, never the discovery JSON body.
func ScanServicesWithPool(target TargetKey, gapiServices []Service, workerCount int, timingEnabled bool, updateCh chan<- ScanUpdate, useGet bool, pool *H3ConnectionPool) ([]string, int, *ElapsedCombo) {
	endpoints := prepareEndpoints(target, gapiServices)
	if len(endpoints) == 0 {
		return nil, 0, &ElapsedCombo{}
	}

	if pool == nil {
		var err error
		pool, err = GetH3ConnectionPool(len(endpoints))
		if err != nil || pool == nil {
			if err != nil {
				log.Printf("HTTP/3 pool unavailable (%v); falling back to per-key connections", err)
			}
			return scanServicesFallback(target, gapiServices, workerCount, timingEnabled, updateCh, useGet)
		}
	}

	method := http.MethodHead
	if useGet {
		method = http.MethodGet
	}
	headers := make(http.Header, 5)
	if !useGet {
		headers.Set("X-HTTP-Method-Override", "GET")
	}
	if target.Referrer != "" {
		headers.Set("Referer", target.Referrer)
	}
	if target.IosBundleId != "" {
		headers.Set("X-Ios-Bundle-Identifier", target.IosBundleId)
	}
	if target.AndroidPkg != "" {
		headers.Set("X-Android-Package", target.AndroidPkg)
	}
	if target.AndroidCert != "" {
		headers.Set("X-Android-Cert", target.AndroidCert)
	}
	req := &http.Request{Method: method, Header: headers}

	pending := make(chan pendingStream, len(endpoints))

	var maxTimeMutex sync.Mutex
	maxTime := &ElapsedCombo{}
	var foundMutex sync.Mutex
	foundServices := make([]string, 0)
	var failCount atomic.Int64

	emit := func(name string, found bool) {
		if updateCh == nil {
			return
		}
		select {
		case updateCh <- ScanUpdate{Key: target.Raw, WasFound: found, ItemCleanName: name}:
		default:
		}
	}

	readers := min(workerCount, 64, len(endpoints))
	if readers < 1 {
		readers = 1
	}

	var readWg sync.WaitGroup
	readWg.Add(readers)
	for range readers {
		go func() {
			defer readWg.Done()
			for p := range pending {
				_ = p.stream.SetDeadline(time.Now().Add(streamDeadline))
				resp, err := p.stream.ReadResponse()

				if timingEnabled {
					elapsed := time.Since(p.start).Milliseconds()
					maxTimeMutex.Lock()
					if elapsed > maxTime.TimeElapsed {
						maxTime = &ElapsedCombo{ServiceClean: p.cleanName, TimeElapsed: elapsed}
					}
					maxTimeMutex.Unlock()
				}

				if err != nil {
					failCount.Add(1)
					if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
						if noteEndpointTimeout(endpointHost(p.host)) {
							log.Printf("Discovery endpoint %s timed out twice; skipping it for the rest of this run", p.host)
						}
					}
					p.stream.Close()
					emit(p.cleanName, false)
					continue
				}

				found := resp.StatusCode == http.StatusOK
				// The status is in memory now; stop the body before it is read.
				p.stream.CancelRead(quic.StreamErrorCode(http3.ErrCodeNoError))
				if found {
					foundMutex.Lock()
					foundServices = append(foundServices, p.cleanName)
					foundMutex.Unlock()
				}
				emit(p.cleanName, found)
			}
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := range endpoints {
		ep := &endpoints[i]
		req.URL = &ep.url
		req.Host = ep.host

		conn := pool.GetNextConn()
		if conn == nil {
			failCount.Add(1)
			emit(ep.cleanName, false)
			continue
		}

		start := time.Time{}
		if timingEnabled {
			start = time.Now()
		}

		stream, err := conn.OpenRequestStream(ctx)
		if err != nil {
			failCount.Add(1)
			emit(ep.cleanName, false)
			continue
		}
		if err := stream.SendRequestHeader(req); err != nil {
			failCount.Add(1)
			stream.Close()
			emit(ep.cleanName, false)
			continue
		}
		// No request body: close the send half now. Without this the stream
		// (and its MAX_STREAMS credit) stays open forever.
		stream.Close()

		pending <- pendingStream{
			cleanName: ep.cleanName,
			host:      ep.host,
			stream:    stream,
			start:     start,
		}
	}

	close(pending)
	readWg.Wait()

	return foundServices, int(failCount.Load()), maxTime
}

func scanServicesFallback(target TargetKey, gapiServices []Service, workerCount int, timingEnabled bool, updateCh chan<- ScanUpdate, useGet bool) ([]string, int, *ElapsedCombo) {
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
		if IsHangingHost(endpointHost(item.DiscoveryUrl)) {
			continue
		}
		scanGroup.Go(func() error {
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
