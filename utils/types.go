package utils

import lib "github.com/bedros-p/fireblazer/lib"

type KeyResult struct {
	Key             string
	ProjectId       string
	Valid           bool
	InvalidReason   error
	FoundServices   []string
	FailCount       int
	MaxTime         *lib.ElapsedCombo
	Brand           map[string]interface{}
	P4SAServices    []string
	IsProjectNumber bool
}

type ServiceDetail struct {
	Name        string `json:"name" yaml:"name"`
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type StructuredOutput struct {
	Key                    string                 `json:"key" yaml:"key"`
	Valid                  bool                   `json:"valid" yaml:"valid"`
	InvalidReason          string                 `json:"invalid_reason,omitempty" yaml:"invalid_reason,omitempty"`
	ProjectId              string                 `json:"project_id,omitempty" yaml:"project_id,omitempty"`
	Brand                  map[string]interface{} `json:"brand,omitempty" yaml:"brand,omitempty"`
	Services               []string               `json:"services" yaml:"services"`
	ServiceDetails         []ServiceDetail        `json:"service_details,omitempty" yaml:"service_details,omitempty"`
	P4SAServices           []string               `json:"inferred_services,omitempty" yaml:"inferred_services,omitempty"`
	InferredServiceDetails []ServiceDetail        `json:"inferred_service_details,omitempty" yaml:"inferred_service_details,omitempty"`
	FailCount              int                    `json:"fail_count,omitempty" yaml:"fail_count,omitempty"`
}
