package fireblazer

import (
	"encoding/json"
	"log"
	"net/url"
)

type GoogleErrorResponse struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const keyCheckEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"

// Contains all general google-specific shenanigans (behavior that has lore to it kinda)

func TestKeyValidity(apiKey string) bool {
	sharedClient := GetClient()
	// It's a non-billed service that's generally quick to respond, i might change it to use a custom transport with a much shorter timeout since i implemented backoff
	resp, err := ReqWithBackoff(AppendAPIKeyToURL(keyCheckEndpoint, apiKey), sharedClient)
	if err != nil {
		resp.Body.Close()
		// TODO: Skip verification flag, id rather have it on for now.
		log.Fatalf("Error testing API key validity: %v\n. Ensure that you can connect to https://generativelanguage.googleapis.com as it's used for checking key validity. To skip primary validation (at risk of invalid results), use the --dangerouslySkipVerification flag.", err)
		return false
	}
	defer resp.Body.Close()

	var result GoogleErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON -> .error.message == "API key not valid. Please pass a valid API key." - I dunno if status 400 is reused, safest bet is the error message
	// 400 shouldn't be reused with listmodels, but I'll be checking edgecases later
	if result.Error != nil {
		if result.Error.Message == "API key not valid. Please pass a valid API key." {
			return false
		}
	}

	return true
}

// URL - parse query parameters and add api key to it
func AppendAPIKeyToURL(apiUrl string, apiKey string) string {
	httpURL, _ := url.Parse(apiUrl)
	values := httpURL.Query()
	values.Add("key", apiKey)
	return httpURL.Scheme + "://" + httpURL.Host + httpURL.Path + "?" + values.Encode()
}
