package utils

import (
	"bytes"
	"fmt"

	lib "github.com/bedros-p/fireblazer/lib"
)

func MarshalStructured(results []KeyResult, showTitle bool, showDesc bool) []StructuredOutput {
	var structuredResults []StructuredOutput

	for _, res := range results {
		out := StructuredOutput{
			Key:          res.Key,
			Valid:        res.Valid,
			ProjectId:    res.ProjectId,
			Brand:        res.Brand,
			FailCount:    res.FailCount,
			Services:     []string{},
			P4SAServices: res.P4SAServices,
		}

		if !res.Valid && res.InvalidReason != nil {
			out.InvalidReason = res.InvalidReason.Error()
		}

		var sDetails []ServiceDetail
		for _, service := range res.FoundServices {
			serviceName := service + ".googleapis.com"
			out.Services = append(out.Services, serviceName)

			if showTitle || showDesc {
				meta, hasMeta := lib.ApiMetadata[service]
				detail := ServiceDetail{Name: serviceName}
				if hasMeta {
					if showTitle {
						detail.Title = meta.Title
					}
					if showDesc {
						detail.Description = meta.Summary
					}
				}
				sDetails = append(sDetails, detail)
			}
		}

		if len(sDetails) > 0 {
			out.ServiceDetails = sDetails
		}

		var inferredDetails []ServiceDetail
		for _, saSvc := range res.P4SAServices {
			detail := ServiceDetail{Name: saSvc}
			if showTitle || showDesc {
				if name, exists := lib.SANames[saSvc]; exists {
					detail.Title = name
				}
			}
			inferredDetails = append(inferredDetails, detail)
		}

		if len(inferredDetails) > 0 {
			out.InferredServiceDetails = inferredDetails
		}

		structuredResults = append(structuredResults, out)
	}
	return structuredResults
}

func MarshalText(results []KeyResult, keys []string, targetApi string, timingEnabled bool, showTitle bool, showDesc bool) []byte {
	var buf bytes.Buffer

	if targetApi != "" {
		buf.WriteString(fmt.Sprintf("\nTARGET: %s\n", targetApi))
	}

	for _, res := range results {
		if targetApi != "" {
			status := "❌"
			if len(res.FoundServices) > 0 {
				status = "✅"
			}
			buf.WriteString(fmt.Sprintf("%s : %s\n", res.Key, status))
			continue
		}

		if len(keys) > 1 {
			buf.WriteString(fmt.Sprintf("\n---%s---\n", res.Key))
		}
		if !res.Valid {
			buf.WriteString(fmt.Sprintf("Invalid API key: %s\nError testing validity: %v\nIf you're sure the key is valid, use the --dangerouslySkipVerification flag.\n", res.Key, res.InvalidReason))
			continue
		}

		if res.Brand != nil {
			buf.WriteString("\nOAuth Client Screen Details\n")
			if displayName, ok := res.Brand["displayName"]; ok && displayName != "" {
				buf.WriteString(fmt.Sprintf(" - App Name: %v\n", displayName))
			}
			if supportEmail, ok := res.Brand["supportEmail"]; ok && supportEmail != "" {
				buf.WriteString(fmt.Sprintf(" - Project Admin / Support Email: %v\n", supportEmail))
			}
			if homePageUrl, ok := res.Brand["homePageUrl"]; ok && homePageUrl != "" {
				buf.WriteString(fmt.Sprintf(" - Homepage: %v\n", homePageUrl))
			}
		}

		if res.IsProjectNumber {
			buf.WriteString(fmt.Sprintf("\n[Blaze] Recon for project number %s (no API key):\n", res.ProjectId))
		} else {
			buf.WriteString(fmt.Sprintf("\nAPIs available to this API key with project ID %s:\n", res.ProjectId))
		}

		for _, service := range res.FoundServices {
			baseMsg := fmt.Sprintf(" - %s.googleapis.com", service)
			meta, hasMeta := lib.ApiMetadata[service]

			details := ""
			if hasMeta {
				if showTitle && meta.Title != "" {
					details += meta.Title
				}
				if showDesc && meta.Summary != "" {
					if details != "" {
						details += " - "
					}
					details += meta.Summary
				}
			}
			if details != "" {
				buf.WriteString(fmt.Sprintf("%s\n   ^-- %s\n", baseMsg, details))
			} else {
				buf.WriteString(fmt.Sprintf("%s\n", baseMsg))
			}
		}

		if len(res.P4SAServices) > 0 {
			buf.WriteString("\n[Additional Recon] Inferred Services via Service Accounts:\n")
			for _, saSvc := range res.P4SAServices {
				if name, exists := lib.SANames[saSvc]; exists {
					buf.WriteString(fmt.Sprintf(" - %s (%s)\n", saSvc, name))
				} else {
					buf.WriteString(fmt.Sprintf(" - %s\n", saSvc))
				}
			}
			buf.WriteString("\n")
		}

		if !res.IsProjectNumber {
			buf.WriteString(fmt.Sprintf("All discovery endpoint tests completed with %d failures.\n", res.FailCount))
		}

		if timingEnabled {
			buf.WriteString(fmt.Sprintf("Longest running service - %v\n\n\n", res.MaxTime))
		}
	}

	return buf.Bytes()
}
