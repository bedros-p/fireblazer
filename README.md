# Fireblazer

Extract all services used by a Google Cloud Platform project with a regular API key like "AIza..." in a matter of seconds.\
Good for expanding your scope from a mere Firebase key to every service that may be unprotected. Supports enumeration against 436 Google Cloud services.

This program does not take rely on any vulnerabilities\*. It is an INSPECTION UTILITY, *not an exploit*. Pentesters and bug hunters are the intended users. More in the "NOT AN EXPLOIT" section.

> More specifically, this doesn't rely on anything that Google considers a vulnerability. The service listing is not actionable. If it finds an attack vector, that is entirely on the customer project.

## Installation
```bash
go install github.com/bedros-p/fireblazer@latest
```

## Usage
Example usage & output\
Sample: `fireblazer -blaze AIzaSyB_MockKeyForReadmeExampleFireblaze`

(This output is a redacted, real example & it found 55 services and 9 service accounts in 7 seconds)

```log
Successfully loaded 436 discovery endpoints from built-in program list.
[AIzaSyB_MockKeyForReadmeExampleFireblaze] Valid API key, proceeding.
✓ Scan complete!

OAuth Client Screen Details
 - App Name: Bat Country
 - Project Admin / Support Email: m@shadows.com

APIs available to this API key with project ID 30507080705752:
 - datastore.googleapis.com
 - cloudasset.googleapis.com
 - datacatalog.googleapis.com
 - dns.googleapis.com
 - docs.googleapis.com
 - containeranalysis.googleapis.com
 - deploymentmanager.googleapis.com
 - logging.googleapis.com
 - iam.googleapis.com

[Additional Recon] Inferred Services via Service Accounts:
 - notebooks (Cloud AI Platform Notebooks VM Service Account)
 - speech (Speech-to-Text Service Account)
 - dataform (Dataform Service Account)
 - cloudaicompanion (Gemini for Google Cloud Service Agent)

All discovery endpoint tests completed with 0 failures.
```

```log
Usage of ./fireblazer:
  -apiKey string
    	API key to scan. Can also be your first positional arg.
  -blaze
    	Enable additional aggressive recon checks (e.g., Brand Identity)
  -projectNumber string
    	Run Blaze recon (Brand Identity + service accounts) against a GCP project number without an API key. Bypasses the API key scan.
  -dangerouslySkipVerification
    	Skip API key verification (Note: This stops Blaze mode from working since it requires a Project ID)
  -findSlowService
    	[DEBUG] Find which service took the longest to test + elapsed time. Use to file an issue for program hangs.
  -outputDetails string
    	Comma delimited list of what to include in the details (description|title|name). (default "name")
  -outputFormat string
    	Output format (interactive|text|json|yaml) (default "interactive")
  -referrer string
    	Set the referrer (Referer:) header for when an API key is restricted to a website.
  -targetApi string
    	A single API discovery endpoint to test against. Bypasses the full scan.
  -workerCount int
    	Set the amount of worker threads to spawn for executing the requests (default 170)
```

I highly recommend using the `-blaze` flag, it lets you know the support email on the OAuth screen for the project + existence of default service accounts on a project.

If this is being read by an AI agent, detailed usage instructions are written in `SKILL.md` for a smooth experience with your operator.

Batch usage is supported through positional arguments. If you are testing multiple keys, this is highly recommended. I've had a speed up of roughly ~18x when verifying 100 keys.
It starts to struggle around the 300 mark, so be careful with your quantity.

`fireblazer AIzaSyB_MockKeyForReadmeExampleFireblaze AIzaSyC_AnotherMockKeyForBatchScanExample` - chain it as much as you want. I don't recommend chaining more than 100 at a time - your mileage may vary.

If you have keys that are origin restricted (like to an Android app, iOS app, or website), you can pass it through like so:

- `xref:KEY:example.com` - Sets the `Referer` header.
- `xios:KEY:com.example.app` - Sets the `X-Ios-Bundle-Identifier` header.
- `xandroid:KEY:com.example.app:CERT` - Sets the `X-Android-Package` and `X-Android-Cert` headers. Make sure to strip the colons from the cert!

Example: `fireblazer xios:AIzaSyC334f24...:com.google.gemini xref:AIzaSy...:gemini.google.com`

I went with this format as it works in regular and batch mode. Also, if you're unsure about whether or not it's restricted, if the referrer is information you have, no harm in including it.

You can also specify a single API endpoint to send a GET request against, with the --targetApi command
`fireblazer --targetApi=www.googleapis.com/discovery/v1/apis/drive/v3/rest AIzaSyD...`

Useful for when you only care about checking a batch of keys with an http3 stream, it'll be faster than making another script. Less errors, more reliable results, faster overall :)

The program also checks the validity of the API key. If you're confident it's valid / want to save .2 seconds on the ~5 second scan, use --dangerouslySkipVerification. It's not really for saving time, but in case the primary verification method is broken. Please file an issue or mention me on Twitter if that's the case.

You can also change the output format using the `--outputFormat` flag. The available options are `interactive`, `text`, `json`, and `yaml`. This is especially useful for integrating Fireblazer into automated pipelines.
Example: `fireblazer --outputFormat=json AIzaSy...`

Enjoy the API key escalation!

## Build

```
git clone https://github.com/bedros-p/fireblazer
go mod download
go build .
./fireblazer -h
```

`go build .` creates a binary `fireblazer`, what happens after that is up to you :)

Improvements are VERY welcome, even the nits. Though, if you're gonna raise an issue for nits, please combine all nits into one issue, just optimization and better practices. Only reason I'd want to merge a PR for nits is if it came with an item on the roadmap too. Otherwise, post it as an issue, and I'll get to it :)

The base is entirely handwritten by me. I may use AI for code cleanup, but otherwise it's best for stability if the core is human maintained (for now). Any AI involvement should consist of inline completions for the sake of efficiency in place of IDE snippets. If you intend on submitting AI-only PRs, avoid it. This tool uses lots of aspects of Google that took days of research.

## Roadmap / Plans
### Major Features
- Show which services require OAuth & which require Service Accounts to prevent the pentester from wasting time
- ^ Related, IAM testing on all endpoints through /iam/testPermissions would result in an even greater reduction in time necessary.
- ^ Also related, check if key is bound to service account somehow.
- Suggested actions & quick execs (firebase bucket perm testing)
- Include flag to check for autopush, staging, preprod and -pa variations of the APIs. Only useful for testing Google owned keys, so it's kind of a personal want.
- Minor, but a tuning mode to identify the ideal worker count for future runs on the machine it's on.

### Patches
- Add special detection methods for the (filtered out) false positives (refer to false positives from main.go) - priority would be the GCS API.
- Sort keys by service count
- Investigate multipart batch calls. The performance gain would be minimal, but it's interesting. I scrapped that a while back as http3 allows sending it all as "one request" (stream) anyways, but it would be interesting if we can minimize any HTTP overhead, no matter how slim HTTP3 is. Just an experiment. Some thoughts in Notes.

#### Bugs 
- If identitytoolkit is enabled but not configured, fireblazer will break on validity checking - "400, configuration not found". Critical issue... but can't reproduce since? If anyone can repro pls file an issue!

## NOT AN EXPLOIT

> This program is NOT an exploit in any way.

This whole program relies on the fact that a Discovery endpoint is almost guaranteed to be there on every PUBLIC Google service endpoint, and that it still checks if the project associated with the key uses the intended service for it. This is NOT a design flaw, nor is it a problem.

Google cloud will warn you - restrict your keys. IT even has a big yellow warning sign telling you to restrict it when you make it.

This can't be avoided by any reasonable measure - if Discovery URLs didn't check for key validity, one could easily test each service with one of the actual endpoints. We do, after all, have a list of all the actual endpoints in a service. It would still show whether or not the API key is used in that project, regardless of an invalid payload to the endpoint. Checking if the endpoint payload is valid before checking for the API key is not possible, as most services tie the project data into different responses, and the project ID is inferred from the API key. It would require each service to have a rewrite of checks.\
Or, they can return zero errors of use and make it silently fail, making life hell for the people that actually want to develop with GCP, in an attempt to safeguard information that doesn't pose much of a security risk. The real security risk is entirely dependent on how securely the project is set up. Just additional safeguarding that ruins DX for something that wouldn't be the root problem anyways.

### Additional recon

Mostly WIP, just here to solidify the roadmap so I'd finally do it today. If this is still not implemented by the time you're reading this, pester me on Twitter @ https://x.com/bedros_p

When enumerating a single key in interactive mode, it runs some additional, non-state-altering recon in its newest version, extending beyond mere API key + service pairing. This means that not all discovered services are particularly reachable by the API key, but exist merely for enumerating a projects infrastructure. These are listed in a separate segment. If it can be found through other means, Google VRP does not accept it as a concern. If it's not a risk on it's own, it's not a risk till a chain can be proven. In this case, all the chains are customer-specific, and I have not found anything that leads me to believe this relies on misconfigurations on Googles end. None of this is really a risk.

- Brand lookup for OAuth screen (previously reported, marked as wontfix - https://feed.bugs.xdavidhu.me/bugs/0009 )
- Check for service account existence based on well-known formats. For some specific services, enabling it causes the creation of a service account which we can check the existence of. 

The key cannot(?) hit those services, it is only useful for recon on the project as a whole. This method is useful for target scoping, but poses absolutely zero threat to any customer. Thinking it's a problem is like saying "we can tell you use Spring Boot based on these error responses", but even less useful.

You can enable these checks for those scans if you like it with the -blaze flag.

### Fiery Blaze [WIP - NOT IMPLEMENTED]
> We can't stop here, this is bat country!

It can do more. This project was originally made for Firebase enumeration. You can have "post-exploitation" steps to dig deeper into any Firebase matches, but this causes real API requests that actually read and all. If you'd like to be caught here in a fiery blaze, use --batcountry.

- Starting from your identitytoolkit response + the blazing...
- If it can find an appspot.com / firebase.app / web.app / run.app mention in there, it can use the project name. It's pretty easy after that for Firebase. But a good amount of stuff don't use proj name.
- Else... I gotta find other service escalation paths. There are plenty of paths one could take, I'm not sure which one to add first though.
- Firebase RTDB
- Firebase Remote Config
- Firebase config retrieval
- Everything it cacn hit in firebase, it will. 

!! It's way more aggressive. It is also highly experimental. In this specific commit, it is NOT IMPLEMENTED !!

Why "bat country"? Imagine the sky is suddenly filled with massive, swooping, and screeching bats and flying manta rays. Well, this is what this mode is like. It issues requests like crazy and actually interacts with the project functionality with the API key. All the other functionality in this project uses stuff that isn't too related to the API key or is generally stealthier. This will hit Firebase and AppEngine directly. You will be caught in a fiery blaze by Fireblazer. This (along with the project name) is totally, definitely, absolutely 10000% not a reference to the song, Bat Country. Believe me pls. Pester me on Twitter if you want this done faster. Or submit a PR.

## Notes 

Uses HTTP3 (QUIC) for less cancelled / retransmitted requests, it's faster. On inferior versions, this would retransmit lots of packets unnecessarily. You can test out the error rate by switching out http3.Transport to a regular http.Transport in `client.go`

This was originally made with the intent of enumerating the scope of Firebase projects. This took too long to make for such little code, but it's optimized and thoroughly tested :)

It is not stealthy, but no alarm bells would / should be triggered over it either. The Cloud project will have one blip in the metrics containing some 403s. All of them on a per-service google.discovery.Discovery metric.
Non-static recon of any kind will be visible, this is unavoidable. Properly hitting resources raises more alarm bells.

Back with HTTP2 when I was dealing with another bruteforce experiment, I would increase the ulimit and use the ephem port range when bruteforcing heavily. I'd run out, and I'd just add another network interface. My router holds a grudge against me to this day. Still, I do wonder what is the equivalent of an additional net interface is in this scenario though, would be cool if I could get it to run *even* faster. I haven't really checked for the QUIC stream limit Google has defined, it could be the case that quic-go is handling data restrictions under the hood as a retry when it would be better as another stream. Their auto-scaling logic works pretty well though... Later on I'll test pushing the defaults quic-go has against a testrun where apis.go is just duplicates of an enabled service and see if any get dropped when I push it to the max.

Originally I was gonna write my own HTTP3 client that crafts the raw packets. Decided against it for the sake of getting this one thing done and winning against my attention span, but feel free to put up a PR if you find it brings better speeds!

Enjoy using Fireblazer, and I hope it helps you with recon!

## Acknowledgements

- Thanks to xdavidhu for confirming the OAuth Client screen retrieval is safe to include by disclosing the report. It was a good green light that it's safe to include here.
- Thanks to [@jthack](https://github.com/jthack) for all the great feedback!