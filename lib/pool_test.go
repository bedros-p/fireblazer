package fireblazer

import (
	"net/url"
	"runtime"
	"strings"
	"testing"
)

func TestCalculatePoolTopology(t *testing.T) {
	tests := []struct {
		requests int
		conns    int
		ports    int
	}{
		{0, minPoolConns, 1},
		{1, minPoolConns, 1},
		{400, minPoolConns, 1},
		{401, 5, 1},
		{10_000, 100, 4},
		{43_600, maxPoolConns, 4},
		{1_000_000, maxPoolConns, 4},
	}
	if runtime.NumCPU() < maxPoolPorts {
		for i := range tests {
			if tests[i].ports > runtime.NumCPU() {
				tests[i].ports = runtime.NumCPU()
			}
		}
	}
	for _, tt := range tests {
		conns, ports := CalculatePoolTopology(tt.requests)
		if conns != tt.conns || ports != tt.ports {
			t.Errorf("CalculatePoolTopology(%d) = (%d,%d), want (%d,%d)", tt.requests, conns, ports, tt.conns, tt.ports)
		}
		if conns*streamsPerConn < min(tt.requests, maxPoolConns*streamsPerConn) {
			t.Errorf("CalculatePoolTopology(%d): %d streams can never cover the request set", tt.requests, conns*streamsPerConn)
		}
	}
}

func TestPrepareEndpoints(t *testing.T) {
	target := TargetKey{Raw: "k", Key: "AIza test/+", Referrer: "example.com"}
	services := []Service{
		{CleanName: "storage", DiscoveryUrl: "https://storage.googleapis.com/$discovery/rest"},
		{CleanName: "hangs", DiscoveryUrl: "https://visionai.googleapis.com/$discovery/rest"},
		{CleanName: "withquery", DiscoveryUrl: "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest?fields=auth"},
		{CleanName: "bare", DiscoveryUrl: "storage.googleapis.com/$discovery/rest"},
	}

	got := prepareEndpoints(target, services)
	if len(got) != 3 {
		t.Fatalf("got %d endpoints, want 3 (blacklisted host must be dropped)", len(got))
	}

	wantQuery := "fields=auth&key=" + url.QueryEscape("AIza test/+")
	for _, ep := range got {
		if ep.cleanName == "hangs" {
			t.Fatal("hanging host was not filtered")
		}
		if ep.scheme != "https" {
			t.Errorf("%s: scheme = %q", ep.cleanName, ep.scheme)
		}
		if ep.host == "" {
			t.Errorf("%s: empty host", ep.cleanName)
		}
		if ep.cleanName == "withquery" && !strings.HasSuffix(ep.uri, "?"+wantQuery) {
			t.Errorf("uri = %q, want suffix %q", ep.uri, wantQuery)
		}
		if ep.cleanName == "bare" && !strings.HasPrefix(ep.uri, "/$discovery/rest?") {
			t.Errorf("bare uri = %q", ep.uri)
		}
	}
}

func TestEndpointHost(t *testing.T) {
	cases := map[string]string{
		"https://storage.googleapis.com/$discovery/rest": "storage.googleapis.com",
		"storage.googleapis.com/$discovery/rest":         "storage.googleapis.com",
		"https://example.com:8443/x":                     "example.com",
	}
	for raw, want := range cases {
		if got := endpointHost(raw); got != want {
			t.Errorf("endpointHost(%q) = %q, want %q", raw, got, want)
		}
	}
}
