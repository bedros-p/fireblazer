# Slow-WiFi Research: What Actually Limits Fireblazer

**Date:** 2026-10-30
**Host:** AMD Ryzen 5 5500U (6 cores / 12 threads), 7.3 GB RAM, Linux, Go 1.26.3
**Stack:** `quic-go v0.59.0`, HTTP/3, ALPN `h3`
**Status:** historical research. The implementation now lives on
`optimization-impl` (2026-10-30); the throwaway worktree is gone. Section 9
adds live re-test numbers from a new network.

---

## TL;DR

- The "near-constant-time" claim is really `T(N) = max(L, N · c / P)`, where
  `L` is the slowest endpoint's latency, `c` is client CPU per request, and
  `P` is cores. It is flat only until `N ≈ L · P / c`, then it is linear.
- `quic-go`'s `http3` layer costs ~130 µs/request before the network does
  anything. Bypassing it with a raw-QUIC HTTP/3 client drops that to ~39 µs
  on loopback.
- Against real GFE the same raw client costs **~128 µs/request**. On 12
  threads that is **~94k req/s**, which is the number that matters. The
  documented 121k is the same CPU wall on slightly faster silicon.
- 305k on GFE needs **~40 threads** (`305k × 128 µs`) and **~610 connections**
  at a 200 ms RTT (`305k × 0.2 s / 100`). 0-RTT makes those 610 connections
  cheap; it does not change either ceiling.
- The documented hang list is **stale**. Live GFE testing found five new
  hanging endpoints and showed that two of the original three now respond.
  This, not packet loss, was the cause of the 20–40 s scan tails.
- A missing `stream.Close()` in the stashed dispatcher leaked every request
  stream. Fixing it turns a 4 GB heap / 19 % GC / thousands of errors into a
  flat heap and zero errors.

---

## 1. The model

For `N` independent requests over `P` cores with per-request client cost `c`
and slowest-service latency `L`:

```
T(N) = max( L , N · c / P )
```

The first term is the "scales to the slowest service" intuition. The second is
the cost of actually producing and consuming `N` requests. The flat region ends
at:

```
N* = L · P / c
```

| Scenario | `L` | `c` | `P` | Flat up to |
|---|---:|---:|---:|---:|
| Stash `http3`, GFE-sized responses | 1.4 s | 129.9 µs | 12 | ~129k requests |
| Tuned `http3`, loopback | ~20 ms | 69.3 µs | 12 | ~3.5k requests |
| Raw HTTP/3, loopback | ~20 ms | 39.4 µs | 12 | ~6k requests |
| Raw HTTP/3, GFE | 1.4 s | 128.4 µs | 12 | ~131k requests |

This is why the published table looks constant from 1 key to 50 keys (both are
pinned to the ~1.4 s `documentai` tail) and linear at 1M (`1M / 121k ≈ 8.3 s`,
which is exactly what was measured).

---

## 2. Where the CPU goes

### 2.1 The `http3` path

Allocation profile at 217k requests (`alloc_space`):

| Share | Site |
|---:|---|
| 49 % | `http3.(*ClientConn).openRequestStream` |
| 12 % | `http3.parseHeaders` → `textproto.MIMEHeader.Add` |
| 7 % | `quic-go.(*frameSorter).push` |
| 2 % | application code |

CPU profile at 434k requests:

| Share | Site |
|---:|---|
| 29 % | `quic-go.(*Conn).run` (packet crypto, ACK, timers) |
| 22 % | `runtime.nanotime` (clock reads / timers) |
| 22 % | `frameParser.ParseNext` / `quicvarint.Read` (byte-wise stream reads) |
| 20 % | application dispatch (request build, stream open, send) |
| 17 % | GC / `mallocgc` |

Per-request allocations were ~2.6 KB. This is why `GOGC` matters so much:

| `GOGC` | Client µs/req | Loopback req/s |
|---:|---:|---:|
| 100 | 93.1 | 58.8k |
| 200 | 73.5 | 71.7k |
| 400 | 69.3 | 73.2k |

### 2.2 The raw path

`scratch/rawclient/main.go` speaks HTTP/3 directly over `*quic.Conn`:

- one control stream with SETTINGS, empty QPACK encoder/decoder streams
- HEADERS frames encoded with `qpack.Encoder` into reused buffers
- manual frame parsing, decode only `:status`
- no `http3.ClientConn`, `RequestStream`, `stateTrackingStream`,
  `http.Response`, `MIMEHeader`, or per-stream `context.WithCancel`

| Path | Client µs/req | Loopback req/s | 1M wall | Failures |
|---|---:|---:|---:|---:|
| Stash `http3` | 129.9 | 43.2k | 21.5–26.6 s | 8k–50k |
| Tuned `http3` | 69–93 | 62–73k | 13.9–17.5 s | 0–1.7k |
| **Raw HTTP/3** | **39.4–41.3** | **96–100k** | **10.4 s** | **0** |

Raw path tuning that mattered:

| Knob | Effect |
|---|---|
| 4 UDP sockets instead of 1 | 41.3 µs vs 42.8 µs |
| `GOGC=800` + `GOMEMLIMIT` | 39.9 µs |
| 64 consumers/key vs 150 | small, within noise |
| `bufio` around stream reads | **worse** (47 µs); extra copy |
| per-stream `SetDeadline` | removed; costs a timer + clock per request |
| `CancelRead` on HEAD | removed; no body to cancel |
| `GetNextConn` `RWMutex` | removed; `conns` is immutable |

---

## 3. The stream leak

The stashed `ScanServicesFast` hand-writes the request
(`OpenRequestStream` + `SendRequestHeader`) instead of going through
`ClientConn.doRequest`, which is what normally calls `str.Close()` when
`req.Body == nil`. The original code only did `CancelRead` + `Body.Close()`,
so the send half of every stream stayed open.

At 1M requests:

| | Before `Close()` | After `Close()` |
|---|---:|---:|
| Wall clock | 21.5–26.6 s | 19.8 s |
| Throughput | 37.8–46.9k req/s | 50.9k req/s |
| Failures | 8,060–50,344 | 0 |
| Heap | 844 MB → 4 GB | flat, 60–130 MB |
| GC CPU | 18–19 % | ~15 % |

One line:

```go
p.stream.Close()
```

This is a correctness/memory bug, not the throughput ceiling — but it explains
every "1M has errors" result and the superlinear wall time.

---

## 4. Slow WiFi / real GFE

All GFE tests use the real `lib.GoogleApiList` discovery endpoints, HEAD with
`x-http-method-override: GET`, and throwaway API keys. Statuses are counted,
not just 200s.

### 4.1 Path characteristics from this sandbox

| Metric | Value |
|---|---|
| QUIC handshake, single conn | ~0.8–0.9 s |
| 50 conns | 17.6–20.6 s |
| 100 conns | 38.4 s |
| 250 conns | 84.6 s |
| p50 request latency | 273–380 ms |
| p90 | 812–895 ms |
| p99 | 1.15–1.62 s |
| Max (transient) | 5.6–7.9 s |

A handful of requests per run hit QUIC PTO backoff
(`1 + 2 + 4 + 8 + 16 ≈ 31 s`). With no per-stream deadline this dominates wall
time even though p99 is ~1.3 s.

### 4.2 GFE throughput

| Run | Requests | Conns | Wall | req/s | Client µs/req | Failures |
|---|---:|---:|---:|---:|---:|---:|
| 100 keys | 42,600 | 100 | 4.70 s | 9,059 | 157.7 | 1 |
| 100 keys, no deadline | 42,600 | 100 | 10.4 s | 4,075 | 134.1 | 0 |
| 300 keys, no deadline | 127,800 | 100 | 11.56 s | 11,053 | **128.4** | 0 |

GFE client CPU profile (100 keys): `nanotime`/timers 29.5 %, response parsing
23.8 %, `Conn.run` 21.3 %, packet writes 16 %.

The per-request cost against GFE is ~3× the loopback cost because GFE's
responses are larger (more QPACK fields, more packets) and the path is lossy.
This is the number that caps a real deployment:

```
12 threads / 128.4 µs = 93.4k req/s
```

Which is essentially the published 121k (`12 / 121k ≈ 99 µs`) — same wall,
slightly different silicon.

### 4.3 Handshakes and 0-RTT are not the bottleneck

Handshakes are a warmup cost, and the pool is reused across every key in a run
(100 handshakes carried 127,800 requests). 0-RTT reduces pool construction to
tens of milliseconds and lets the first request ride in the first flight, but:

- it does not raise `initial_max_streams_bidi` (100 per conn);
- it does not reduce per-request CPU.

Both ceilings still apply:

```
throughput = min( P / c ,  conns × 100 / RTT )
```

| RTT | Conns for 305k |
|---|---:|
| 200 ms | ~610 |
| 800 ms (this sandbox) | ~2,440 |

### 4.4 What 305k actually requires

- **CPU:** `305k × 128.4 µs ≈ 39.2 cores`. A 40-thread host at this per-request
  cost can do it; a 12-thread laptop cannot.
- **Conns:** ~610 at 200 ms RTT, ~2,440 at 800 ms.
- **Hang blacklist:** mandatory (see section 5).
- **Per-stream deadline:** ~3 s, to bound PTO tails.
- **1M in 1.4 s** would need `1M / 1.4 s = 714k req/s` → ~92 cores at GFE cost,
  or ~28 cores at the loopback raw cost.

---

## 5. The hang list is stale

Documented in the branches:

- `implementation.md` §Step 1 and `research-process.md` §9:
  `datalabeling`, `apigeeregistry`, `dlp`
- commit `7700a98`: `translationhub` (unresponsive), `analytics` (no discovery
  doc)
- commit `dd2d3c6`: "I will sunset datalabeling and apigeeregistry too as those
  hang"

Live test, all 434 endpoints, 1 key, 3 s deadline, blacklist disabled:

| Host | Result |
|---|---|
| `apigeeregistry.googleapis.com` | hangs |
| `cloudvideosearch.googleapis.com` | hangs |
| `contentwarehouse.googleapis.com` | hangs |
| `rapidmigrationassessment.googleapis.com` | hangs |
| `visionai.googleapis.com` | hangs |
| `workloadcertificate.googleapis.com` | hangs |
| `cloudtasks.googleapis.com` | hung once, then clean ×3 — transient |
| `datalabeling.googleapis.com` | responds now |
| `dlp.googleapis.com` | responds now |

So: one of the old three still hangs, two recovered, and five new hosts hang.
With the five new hosts added to `HangingDiscoveryBlacklist`, three consecutive
diagnostic runs had zero timeouts and p99 ~1.15 s.

Recommended current set:

```go
var HangingDiscoveryBlacklist = map[string]bool{
    "datalabeling.googleapis.com/$discovery/rest":             true,
    "apigeeregistry.googleapis.com/$discovery/rest":           true,
    "dlp.googleapis.com/$discovery/rest":                      true,
    "cloudvideosearch.googleapis.com/$discovery/rest":         true,
    "contentwarehouse.googleapis.com/$discovery/rest":         true,
    "rapidmigrationassessment.googleapis.com/$discovery/rest": true,
    "visionai.googleapis.com/$discovery/rest":                 true,
    "workloadcertificate.googleapis.com/$discovery/rest":      true,
}
```

Keeping `datalabeling`/`dlp` costs two endpoints of coverage but protects
against them flapping. Longer term this should be discovered automatically:
log every endpoint that hits the deadline and persist the set, rather than
hardcoding a list that decays.

Observation: with a throwaway key, 426 endpoints produced
`399 × 400`, `26 × 404` per key; across 300 keys, 14 returned `200` (open
discovery docs) and 2 returned `502`. `502` appears transiently and is worth
counting separately from hangs.

---

## 6. Code changes worth keeping

Applied on `optimization-impl` (2026-10-30); the scratch worktree is gone:

1. `p.stream.Close()` after reading the status — leak fix.
2. Prepared endpoint cache: no `url.Parse` per request.
3. Per-key header map built once; one `http.Request`/`url.URL` reused for all
   endpoints of a key.
4. No per-stream `SetDeadline`; no `CancelRead` for HEAD.
5. Lock-free `GetNextConn`.
6. Five new hanging hosts in `HangingDiscoveryBlacklist`.
7. `GOGC=200`/`GOMEMLIMIT` (measured; not yet in code).

Prototypes (throwaway, `//go:build ignore`):

- `scratch/localbench/main.go` — loopback HTTP/3 server + client harness
- `scratch/rawclient/main.go` — raw HTTP/3 client vs loopback
- `scratch/gfeclient/main.go` — raw HTTP/3 client vs GFE, with latency
  percentiles and per-endpoint diagnostics

Environment knobs: `FB_KEYS`, `FB_CONNS`, `FB_PORTS`, `FB_CONSUMERS`,
`FB_DEADLINE_MS` (`-1` disables), `FB_DIAG`, `FB_NO_BLACKLIST`,
`FB_CPUPROFILE`.

---

## 7. Caveats

- Loopback numbers share the CPU with a `quic-go` test server (~72–82 µs/req).
  They measure combined capacity, not client-only; the client-only ceiling is
  derived from `P / c`.
- The sandbox path to GFE is ~800 ms RTT and lossy; 100 handshakes taking 38 s
  is not representative of a normal network. Absolute GFE throughput here is
  not meaningful; the per-request CPU is.
- Wall-clock on a 15 W laptop moves ±20 % with thermals/clocks. `c` is the
  stable metric.
- The raw client is a measurement vehicle: no GOAWAY, no SETTINGS validation,
  no trailers, no GET-body drain, no retry/fallback, and it relies on the
  local-only `FB_BENCH_ADDR` patch.

---

## 8. Next steps

1. Port the blacklist update and the `Close()` fix to `optimization-round`.
2. Attack GFE's 128 µs: the 23.8 % parse share is the most actionable
   (frame-aware buffering without the copy penalty that `bufio` introduced).
3. Add 0-RTT warmup to the pool so 610+ connections cost ~50 ms, not seconds.
4. Auto-discover hanging endpoints instead of maintaining a static list.
5. Re-run the 305k target on a ~40-thread host with a clean ~200 ms path and
   ~610 warm connections.

---

## 9. Live re-test: new network, 2026-10-30

Connection: 12-thread Ryzen 5 5500U, campus Wi-Fi, ~16 ms TCP connect,
~3.5 MB/s path, no IPv6 route. All tests use 100 throwaway `AIza...` keys
(43,500 discovery requests) against live GFE, `-dangerouslySkipVerification`,
JSON output, 0-RTT disabled by a fresh process.

### 9.1 The 24-second key is two 10-second hangs

A single key took **23.4–25.2 s** (3 runs, 6 failures each). Probe over plain
TCP (HTTP/2, not the QUIC client) reproduced the same split:

| Host | HTTP/2 probe |
|---|---|
| `apigeeregistry.googleapis.com` | no response in 6 s |
| `cloudvideosearch.googleapis.com` | no response in 6 s |
| `contentwarehouse.googleapis.com` | no response in 6 s |
| `rapidmigrationassessment.googleapis.com` | no response in 6 s |
| `visionai.googleapis.com` | no response in 6 s |
| `workloadcertificate.googleapis.com` | no response in 6 s |
| `datalabeling.googleapis.com` | 404 in 0.49 s |
| `dlp.googleapis.com` | 400 in 1.10 s |

The old path sets a 10 s stream deadline and then retries once with a
per-host DNS lookup; six hosts therefore cost **2 × 10 s**. That is 20 of the
23.7 s. `datalabeling` and `dlp` have recovered and should not be blacklisted.
The static set in `lib/enumeration.go` is the six confirmed hosts above;
any host timing out twice in one process is also skipped for the rest of the
run.

### 9.2 Base vs optimized (same network, same corpus)

| Case | Base (`optimization-impl` HEAD) | Optimized | Old claim |
|---|---:|---:|---:|
| 1 key / 430 requests | 23.4–25.2 s | 2.0–2.2 s | 1.52–1.72 s |
| 5 keys / 2,150 requests | 26.3–26.8 s | 3.0–4.3 s | — |
| 100 keys / 43,500 requests | aborted after ~6 min | **3.4–4.6 s** | ~8.5 s |
| Throughput, best run | — | **12.6k req/s** | 4,660 req/s (50k) |
| Failures | 6 per key | **0** | 0 |

The base 100-key run never produced output: after its serialized handshakes
failed, it fell back to resolving every service host for every key and wrote
45,606 error lines. That is the old code's behavior on this network, not a
corpus problem.

The optimized single-key run is bounded by one real slow endpoint
(`areainsights.googleapis.com`, 1.5 s), not a hang. The 100-key run is bounded
by GFE latency and response tails, not client CPU: ~6.6 s CPU / 43.5k requests
≈ 152 µs per request including 250 handshakes, or ~79k req/s of client
capacity on 12 threads. The documented 121k needs the raw-client path from
§2.2; it is not reachable through `http3.ClientConn`.

### 9.3 What was applied

- `lib/pool.go`: process-wide pool of up to 250 pre-warmed connections over
  4 UDP sockets; round-robin, lock-free selection; handshakes run
  concurrently instead of under the old global mutex.
- Decoupled dispatch/read pipeline in `lib/enumeration.go`: send all 430
  requests first, read statuses on up to 64 consumers per key, cancel the
  body after the HEADERS frame, and `Close()` the send half immediately.
- 3 s per-stream deadline; no retry storm. 6-host static blacklist plus
  in-process auto-blacklist after two timeouts.
- Endpoint parsing and key appending happen once per key; headers and the
  request object are reused across all endpoints of that key.
- Batch keys still run concurrently (up to 128), so the pool is shared
  instead of one connection per key.

Caveats: campus Wi-Fi latency moved ±30 % between runs; `areainsights` added
~1.5 s to some single-key runs. Absolute wall times are not stable, but the
failure count and the 6-host split were reproducible across every run.
