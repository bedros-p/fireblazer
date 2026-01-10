# Fireblazer

Extract all services used by a Google Cloud Platform project with a regular API key like "AIza...".\
Good for expanding your scope from a mere Firebase key to every service that may be unprotected.

It is buggy right now (duplicated entries), but no false positives. 

## Building
Install all tha dependencies first obv\
`go build .` creates a binary `fireblazer`

## Usage
Example usage & output\
`./fireblazer AIzaSyC334f24LundukeS8uSkjWoke18`

Output:
```log
2026/01/10 06:34:22 Successfully retrieved discovery endpoints.
2026/01/10 06:34:43 APIs available to this API key:
2026/01/10 06:34:43  - https://generativelanguage.googleapis.com/$discovery/rest 
	- The Gemini API allows developers to build generative AI applications using Gemini models. Gemini is our most capable model, built from the ground up to be multimodal. It can generalize and seamlessly understand, operate across, and combine different types of information including language, images, audio, video, and code. You can use the Gemini API for use cases like reasoning across text and images, content generation, dialogue agents, summarization and classification systems, and more. - Generative Language API
2026/01/10 06:34:43 All discovery endpoint tests completed with 0 failures.
```

The program also checks the validity of the API key. If you're confident it's valid / want to save half a second on the 30 second scan, use --dangerouslySkipVerification. It's not really for saving time, but in case the primary verification method is broken.

Yeah that's all the program does, it just scoops up all the services in use by a cloud project. Enjoy the escalation!

## Roadmap / Plans
### Major Features
- Show which services require OAuth & which require Service Accounts to prevent the pentester from wasting time
- Suggested actions & quick execs (firebase bucket perm testing)

### Patches
- Use a file containing the endpoints while waiting for the up-to-date stuff to load in from the two sources (GoogleAPIs Github & GoogleAPIs Discovery service). Compare content-length with a HEAD and if there's a change get the new one. Or if you want to contribute do it your way idk just make it good 
- Add special detection methods for the (filtered) false positives (refer to false positives from main.go) - priority would be the GCS API.
- Need it to go FASTER!!!! I picked QUIC / HTTP3 because I had lots of out-of-order requests messing up, and lots of EOF & conn resets, none of which happened on QUIC. Though I guess I can attribute that to the fact that it's slower now. I had a HEAD-only HTTP2 impl, refer to the commit history if you'd like to make that work I guess?
- Include project ID in the output. Can be useful for some services.

#### Bugs 
- Deduplicate properly based on service ID & pick the non-empty description - service ID `logging.googleapis.com` -> `logging`. It's an easy task, I just broke the impl is all with my silly experiment of same-domain stuff and I don't want to revert the commit because I think it has potential (it probably doesn't)
- Goes too SLOW!!!!!!!!!!!!!!!!!!!!!!!!!

## Notes
Apologies for the sloppy code. Been a while since I've written Go. Or worked on something for more than an hour. Couple of months? Improvements are VERY welcome, even the nits. Though... not the really annoying nits. I mean like, just optimization and better practices.

I'm only publishing this in tha current buggy state because it's 6:52AM and I've been up trying to make this as good as possible since 1AM.

This took too long to make for such little code.