# Architecture Evolution: Fireblazer QUIC Optimization

This document outlines the architectural shift in Fireblazer's network and concurrency models to achieve near-constant-time scaling for massive API key batch scans.

## The Baseline Architecture (Before)

Initially, Fireblazer used a per-key concurrency model that mapped exactly one QUIC connection per API key.

**Data Flow:**
1. `ProcessKey` launched concurrently for every API key provided.
2. Inside each `ProcessKey`, 170 workers (`workerCount`) were launched simultaneously.
3. Every worker called `ReqHeaderOnly`, requesting a connection from the `h3ConnMap` keyed by `apiKey + destAddr`.
4. If the connection didn't exist, the worker hit `h3Mutex.Lock()` and initiated the `net.ResolveUDPAddr` and `tr.DialEarly` sequence.

**The Physical Bottlenecks (Why 50 keys took 8.4+ seconds):**

1. **The Mutex Collision:** When 50 keys were scanned, 8,500 goroutines instantly fired. They all slammed into the global `h3Mutex`. The first 50 goroutines (one for each key) held the lock while establishing the QUIC connection (taking ~130ms each). The remaining 8,450 workers were completely frozen in context-switching purgatory waiting for the lock.
2. **The QUIC Stream Limit:** A single QUIC connection usually restricts multiplexing to 100 or 256 concurrent HTTP/3 streams. Because Fireblazer spawned 170 workers per key, it frequently bumped against the `MAX_STREAMS` ceiling for that single connection. This forced `quic-go` to queue requests locally until stream credits were replenished, introducing artificial latency.
3. **The DNS Flood:** When fallback routing was triggered (or when 50 connections initialized simultaneously), it triggered dozens of concurrent `net.ResolveUDPAddr` calls to the system DNS resolver, serializing the lookup requests at the OS level.

---

## The Optimized Architecture (Now)

The new architecture shifts from a "1 Connection per Key" model to a "Warmed-up Global Multiplexed Pool" model, decoupling the cryptographic connections from the target payload.

**Data Flow:**

1. **Calculate Optimal Pool Size:** 
   In `main.go`, we calculate how many connections are mathematically required to ensure we never hit the QUIC stream limit:
   `connectionsPerKey = ceil(endpoints / workerCount)` -> (436/170 = 3).
   `numConns = 3 * len(keys)`. (e.g., 50 keys -> 150 connections).

2. **The Concurrent Warmup Phase:**
   Before any workers are launched, `WarmupConnections` uses an `errgroup` to rapidly establish all 150 QUIC connections. 
   - **Crucial Scope Fix:** The `h3Mutex` inside `getSharedH3Conn` was narrowed to only protect map reads/writes. This allows the 150 `tr.DialEarly` network IO calls to happen truly concurrently, without locking out the rest of the program.

3. **Global DNS Cache:**
   The very first connection to acquire the lock during warmup executes `net.ResolveUDPAddr("udp", "googleapis.com:443")`. It saves this Anycast IP to a global variable (`lib.CachedGoogleApiIp`). The remaining 149 connections completely bypass the system DNS resolver and dial the IP directly.

4. **Per-Worker Stream Distribution:**
   When the concurrent scan begins, the connection mapping is no longer tied to the API key. Instead, the `poolIndex` is determined mathematically per request: `poolIndex := i % numConns`.
   - If 50 keys are scanned, all 8,500 active streams are perfectly sprayed across the 150 connections.
   - Every single QUIC connection handles a constant, mathematically optimal ~56 streams, well below the `MAX_STREAMS` ceiling. Zero queuing delays.

5. **Transport Multiplexing (The Router Fix):**
   All pooled QUIC connections to `googleapis.com` intentionally share a single `quic.Transport` (and thus, a single local ephemeral UDP port). 
   - *Why?* If we forced 150 independent local UDP sockets, the OS and local router NAT tables would interpret the burst of 8,500 packets as a flood and drop them, causing 17-second retransmission timeouts.
   - By multiplexing all 150 QUIC cryptographic contexts over a single local UDP socket, the router sees one smooth, high-throughput data stream. `quic-go` efficiently demultiplexes the returning packets using QUIC Connection IDs.

## Final Result

By eliminating mutex contention, bypassing DNS, dodging stream limits, and perfectly multiplexing transport sockets, Fireblazer now processes 50 keys in virtually the exact same time it takes to process 1 key (approx. 3.2 seconds). It scales with absolute near-constant time.