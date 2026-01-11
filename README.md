# Fireblazer

Extract all services used by a Google Cloud Platform project with a regular API key like "AIza...".\
Good for expanding your scope from a mere Firebase key to every service that may be unprotected.

## Installation
```bash
go install github.com/bedros-p/fireblazer@latest
```

### From source

```
git clone https://github.com/bedros-p/fireblazer
go mod download
go build .
./fireblazer -h
```

`go build .` creates a binary `fireblazer`, what happens after that is up to you :)

## Usage
Example usage & output\
`fireblazer AIzaSyC334f24LundukeS8uSkjWoke18`

Output:
```log
2026/01/11 21:18:35 Valid API key, proceeding.
2026/01/11 21:18:36 Successfully retrieved 376 discovery endpoints - 0 endpoint sources failed.
✓ Scan complete! Identified 5 services available in the project.
2026/01/11 21:18:41 APIs available to this API key:
2026/01/11 21:18:41  - cloudprofiler.googleapis.com / Cloud Profiler API 
	- Manages continuous profiling information.
2026/01/11 21:18:41  - bigtableadmin.googleapis.com / Cloud Bigtable Admin API 
	- Administer your Cloud Bigtable tables and instances.
2026/01/11 21:18:41  - container.googleapis.com / Kubernetes Engine API 
	- Builds and manages container-based applications, powered by the open source Kubernetes technology.
2026/01/11 21:18:41  - cloudfunctions.googleapis.com / Cloud Functions API 
	- Manages lightweight user-provided functions executed in response to events.
2026/01/11 21:18:41  - cloudscheduler.googleapis.com / Cloud Scheduler API 
	- Creates and manages jobs run on a regular recurring schedule.
2026/01/11 21:18:41 All discovery endpoint tests completed with 0 failures.
```

The program also checks the validity of the API key. If you're confident it's valid / want to save .2 seconds on the ~5 second scan, use --dangerouslySkipVerification. It's not really for saving time, but in case the primary verification method is broken.

Enjoy the API key escalation!

## Roadmap / Plans
### Major Features
- Support multiple output formats (YAML, JSON, Plain text & fancy cli outputs \[spinners\]) (Partial implementation)
- Show which services require OAuth & which require Service Accounts to prevent the pentester from wasting time
- Suggested actions & quick execs (firebase bucket perm testing)
- Include project ID in the output. Can be useful for some services.

### Patches
- Use a file containing the endpoints while waiting for the up-to-date stuff to load in from the two sources (GoogleAPIs Github & GoogleAPIs Discovery service). Compare content-length with a HEAD and if there's a change get the new one. Or if you want to contribute do it your way idk just make it good 
- Add special detection methods for the (filtered) false positives (refer to false positives from main.go) - priority would be the GCS API.
- First request should be used for validating the key instead of having an alternate request for it (can possibly be bundled with the first discovery request, or launched during the scan and cancelling when invalid)

#### Bugs 
- Every now and then, a network black hole would occur. A retry pool is still necessary for unstable connections, even with all the help this program already gets from using HTTP3.

## Notes
Uses HTTP3 (QUIC) for less cancelled / retransmitted requests, it's faster. On inferior versions, this would retransmit lots of packets unnecessarily. You can test out the error rate by switching out http3.Transport to a regular http.Transport in `client.go`

Apologies for the sloppy code. Been a while since I've written Go. Or worked on something for more than an hour. Couple of months? Improvements are VERY welcome, even the nits. Though... not the really annoying nits. I mean like, just optimization and better practices.

This took too long to make for such little code.\
I should probably clean up the comments.