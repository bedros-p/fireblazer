package utils

import (
	"fmt"
	"log"
	"strings"

	lib "github.com/bedros-p/fireblazer/lib"
)

type ProcessConfig struct {
	DangerouslySkipVerification bool
	IsInteractive               bool
	OutputFormat                string
	WorkerCount                 int
	TimingEnabled               bool
	UseGet                      bool
}

func ParseTargetKey(raw string, globalRef string) lib.TargetKey {
	tk := lib.TargetKey{
		Raw:      raw,
		Key:      raw,
		Referrer: globalRef,
	}

	if strings.HasPrefix(raw, "xios:") {
		parts := strings.SplitN(raw, ":", 3)
		if len(parts) >= 3 {
			tk.Key = parts[1]
			tk.IosBundleId = parts[2]
			tk.Referrer = ""
		}
	} else if strings.HasPrefix(raw, "xandroid:") {
		parts := strings.SplitN(raw, ":", 4)
		if len(parts) == 4 {
			tk.Key = parts[1]
			tk.AndroidPkg = parts[2]
			tk.AndroidCert = parts[3]
			tk.Referrer = ""
		}
	} else if strings.HasPrefix(raw, "xref:") {
		parts := strings.SplitN(raw, ":", 3)
		if len(parts) >= 3 {
			tk.Key = parts[1]
			tk.Referrer = parts[2]
		}
	}
	return tk
}

func ProcessKey(target lib.TargetKey, gapiServices []lib.Service, updateCh chan lib.ScanUpdate, logCh chan string, cfg ProcessConfig) KeyResult {
	defer lib.TrackTime("ProcessKey for " + target.Raw)()
	res := KeyResult{Key: target.Raw}
	if cfg.DangerouslySkipVerification {
		if cfg.IsInteractive && logCh != nil {
			logCh <- fmt.Sprintf("[%s] Skipping API key verification.", target.Raw)
		}
		res.Valid = true
	} else if valid, projectDetails, err := lib.TestKeyValidity(target.Key); !valid {
		res.Valid = false
		res.InvalidReason = err
		if updateCh != nil {
			updateCh <- lib.ScanUpdate{Key: target.Raw, WasFound: false, ItemCleanName: "[INVALID]"}
		}
		return res
	} else {
		res.ProjectId = projectDetails.ProjectId
		if cfg.IsInteractive && logCh != nil {
			logCh <- fmt.Sprintf("[%s] Valid API key, proceeding.", target.Raw)
		} else if cfg.OutputFormat == "text" {
			log.Printf("[%s] is a valid API key.", target.Raw)
		}
		res.Valid = true
	}

	foundServices, failCount, maxTime := lib.ScanServices(target, gapiServices, cfg.WorkerCount, cfg.TimingEnabled, updateCh, cfg.UseGet)
	res.FoundServices = foundServices
	res.FailCount = failCount
	res.MaxTime = maxTime
	return res
}
