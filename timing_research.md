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


----

This commit will not merge. Ever. This branch exists solely for research.

The additions to main.go is a basic implementation of my thoughts and the basic distribution logic, so I have no problem putting that in. I will take extensive time to ensure the pooling in client.go is as described. So far, the results line up perfectly with what I expected and thats without AI putting in some "This is a placeholder - in a real environment..." - it works really well and doesn't error out... at all? Previously I mentioned in the README that it would drop a bunch of things and lead to a network blackhole. 

Initially I suspected that if we take too long on a stream, the load balancer would kick in and Google would send a PREFERRED_ADDRESS frame to one that isn't busy. But since I was handling Dials and connection streams manually, I figured maybe I'd have to handle it. I tried to get AI to replicate the scenario where I got hit by so many network blackholes. I tried it on a slower, public wifi connection, and most of it got dropped, and this was something i've been aware of for a long time but couldn't identify the root cause, Wireshark was of no help because filtering to the streams I would never actually see that it was the router dropping it all along. But it was the only explanation that made sense. Fed everything I knew to Gemini 3.1 Pro and it worked right then and there, fixing all the black hole issues and convinced it it's not because of ratelimiting or something that silly.

I'm pretty happy with how it turned out. But I am interested, one roundtrip takes 220 ms, this takes 3 seconds per key, which is still pretty quick, but if it really can scale as I outlined, I should be able to push it way down per-key in general.

Still, batching overhead is mostly gone! All of what remains is just how this sorta stuff works. It wouldn't be possible to get a perfect o(1) here. I'll implement the warmup methods first, then work on the balancing.