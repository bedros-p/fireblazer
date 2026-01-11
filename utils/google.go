package fireblazer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type GoogleErrorResponse struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const keyCheckEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"

// Contains all general google-specific shenanigans that don't belong elsewhere (behavior that has lore to it kinda)

func TestKeyValidity(apiKey string) (bool, error) {
	sharedClient := GetClient()
	req, _ := http.NewRequest("GET", AppendAPIKeyToURL(keyCheckEndpoint, apiKey), nil)
	resp, err := ReqWithBackoff(req, sharedClient)

	if err != nil {
		resp.Body.Close()
		return false, err
	}

	defer resp.Body.Close()

	var result GoogleErrorResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON -> .error.message == "API key not valid. Please pass a valid API key." - I dunno if status 400 is reused, safest bet is the error message
	// 400 shouldn't be reused with listmodels, but I'll be checking edgecases later
	if result.Error != nil {
		errorMarshal, err := json.Marshal(result)
		errorString := string(errorMarshal) + " ---- HTTP STATUS " + resp.Status
		if result.Error.Message == "API key not valid. Please pass a valid API key." || (resp.StatusCode != 200 && resp.StatusCode != 403) {
			if err != nil {
				return false, fmt.Errorf("DOUBLE WHAMMY : API Key not valid, JSON marshal for error message failed too. Error->Message: %s", result.Error.Message)
			}
			return false, fmt.Errorf("%v", errorString)
		}
	}

	return true, nil
}

// Parse query params and append the API key to it.
func AppendAPIKeyToURL(apiUrl string, apiKey string) string {
	httpURL, _ := url.Parse(apiUrl)
	values := httpURL.Query()
	values.Add("key", apiKey)
	return httpURL.Scheme + "://" + httpURL.Host + httpURL.Path + "?" + values.Encode()
}
