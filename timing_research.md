# Timing research to make this faster

Rn I got Gemini inserting perf markers and stuff. I have *some* ideas but I need to verify them. First one being the handshake blocks wayyy too much at the start. It wouldnt be a prooblem for large batch scans, but I wanna get the speed as much as possible.

Rn, we're sending all of it on a single http3 stream like this:
- resolve DNS once
- send everything on that same stream, modifying the host headers to route it to the right Google API thanks to their load balancer, but form a new stream per key. I suspect if there's a bottleneck, it's in my key spawning.
- worker count is a hard ceil, but its rolling, so whenever a req is complete, it goes onto the next one

I think back when I was using HTTP2 with no batch option, the worker count was a useful setting. But I don't see why with H3 I hold a per-key worker count. Workers were just there to avoid overwhelming net and cpu.\
But I think it can be pushed harder?

I want a better way to do this. 

Look, if a roundtrip takes 220ms, this program should scale as much as possible to make sure there's no overhead. With no ratelimiting (they're all diff services & different api keys), we should still be close to sub second scans even for 30 keys. I'm considering splitting the conns for batch ids for max efficiency. If each service takes 220 ms to respond, I can stay as close as I can to that for as much CPU & RAM & net I have. Anything else is just overhead on my part.

Rn the key conn mapping is:

```go
	if useActualResolvedName {
		destAddr = hostname + ":443"
	} else {
		destAddr = "googleapis.com:443"
	}

	connKey := apiKey + "|" + destAddr

	if conn, ok := h3ConnMap[connKey]
```

I might make it so that:
```go
	if useActualResolvedName {
		destAddr = hostname + ":443"
	} else {
		destAddr = "googleapis.com:443"
	}

	connKey := apiKey + "|" + destAddr + batchNum

	if conn, ok := h3ConnMap[connKey]
```

Where batchnum is the current batch id of the key wave.

My thoughts for the new program flow:
- Warm up the DNS resolution - i used to do this in the older versions of the program iirc
- Collect all the deduplicated keys
- Async #1 -> Spawn off the key verification work 
- Loop through all the keys and create the HTTP request representation for every service and pre-batch the workers to support that iodea
- On key verification, launch the batches, each of which sending off 170 / "worker count" requests to their own unique connection. 

I should instead rename it to batchSize if making this, and worker count as a concept should instead be parallelkeycount.

