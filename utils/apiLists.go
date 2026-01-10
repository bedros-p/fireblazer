package fireblazer

import (
	"encoding/json"
	"log"
	"net/http"
)

// Alternative Source for Discovery Doc
// Deduplicated, reaches around 240. Discovery doc, deduplicated, reaches around 300. And yet both have services the other doesnt.
// This would fill the gap to be as comprehensive as possible. Combined and deduplicated, you reach ~380 hostnames.

var apiListGithub = "https://raw.githubusercontent.com/googleapis/googleapis/refs/heads/master/api-index-v1.json"

// {
// 	"id": "google.actions.sdk.v2",
// 	"directory": "google/actions/sdk/v2",
// 	"version": "v2",
// 	"majorVersion": "v2",
// 	"hostName": "actions.googleapis.com",
// 	"title": "Actions API",
// 	"description": "A RESTful service for the Actions API.",
// 	"importDirectories": [
// 	"google/actions/sdk/v2",
// 	"google/actions/sdk/v2/conversation",
// 	"google/actions/sdk/v2/conversation/prompt",
// 	"google/actions/sdk/v2/conversation/prompt/content",
// 	"google/actions/sdk/v2/interactionmodel",
// 	"google/actions/sdk/v2/interactionmodel/prompt",
// 	"google/actions/sdk/v2/interactionmodel/prompt/content",
// 	"google/actions/sdk/v2/interactionmodel/type",
// 	"google/api",
// 	"google/protobuf",
// 	"google/rpc",
// 	"google/type"
// 	],
// 	"options": {
// 	"go_package": {
// 		---
// 	},
// 	"java_multiple_files": {
// 		"valueCounts": {
// 		"true": 60
// 		}
// 	},
// 	"java_package": {
// 		"valueCounts": {
// 		"com.google.actions.sdk.v2": 18,
// 		"com.google.actions.sdk.v2.conversation": 14,
// 		"com.google.actions.sdk.v2.interactionmodel": 8,
// 		"com.google.actions.sdk.v2.interactionmodel.prompt": 14,
// 		"com.google.actions.sdk.v2.interactionmodel.type": 6
// 		}
// 	}
// 	},
// 	"services": [
// 	{
// 		"shortName": "ActionsSdk",
// 		"fullName": "google.actions.sdk.v2.ActionsSdk",
// 		"methods": [
// 		{
// 			"shortName": "CreateVersion",
// 			"fullName": "google.actions.sdk.v2.ActionsSdk.CreateVersion",
// 			"mode": "CLIENT_STREAMING",
// 			"bindings": [
// 			{
// 				"httpMethod": "POST",
// 				"path": "/v2/{parent=projects/*}/versions:create"
// 			}
// 			]
// 		},
//      ...
// 		]
// 	}
// 	],
// 	"configFile": "actions_v2.yaml",
// 	"serviceConfigApiNames": [
// 	"google.actions.sdk.v2.ActionsSdk",
// 	"google.actions.sdk.v2.ActionsTesting"
// 	],
// 	"nameInServiceConfig": "actions.googleapis.com"
// },

// Trimmed for brevity

// im only interested in:
// description, name, host
type GapisApiItem struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Host        string `json:"hostname"`
}

type GapisContainer struct {
	Apis []GapisApiItem `json:"apis"`
}

func GetEndpointsFromGapis() ([]GapisApiItem, error) {
	client := http.DefaultClient // Github doesn't seem to like QUIC - using regular client

	body, err := client.Get(apiListGithub)
	if err != nil {
		// TODO: Local fallback
		log.Fatalf("Error fetching supplementary Gapis API list: %v", err)
		return nil, err
	}

	var apiList GapisContainer
	if err := json.NewDecoder(body.Body).Decode(&apiList); err != nil {
		log.Fatalf("Error decoding Gapis API list: %v", err)
		return nil, err
	}
	return apiList.Apis, nil
}
