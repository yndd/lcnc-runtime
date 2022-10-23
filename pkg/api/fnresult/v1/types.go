package v1

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Result contains the structured result from an individual function
type Result struct {
	// Image is the full name of the image that generates this result
	// Image and Exec are mutually exclusive
	Image string `yaml:"image,omitempty"`
	// ExecPath is the the absolute os-specific path to the executable file
	// If user provides an executable file with commands, ExecPath should
	// contain the entire input string.
	ExecPath string `yaml:"exec,omitempty"`
	// TODO(droot): This is required for making structured results subpackage aware.
	// Enable this once test harness supports filepath based assertions.
	// Pkg is OS specific Absolute path to the package.
	// Pkg string `yaml:"pkg,omitempty"`
	// Stderr is the content in function stderr
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the exit code from running the function
	ExitCode int `yaml:"exitCode"`
	// Results is the list of results for the function
	Results framework.Results `yaml:"results,omitempty"`
}

const (
	ResultListKind       = "FunctionResultList"
	ResultListGroup      = "lcnc.yndd.io"
	ResultListVersion    = "v1"
	ResultListAPIVersion = ResultListGroup + "/" + ResultListVersion
)

// ResultList contains aggregated results from multiple functions
type ResultList struct {
	yaml.ResourceMeta `yaml:",inline"`
	// ExitCode is the exit code of kpt command
	ExitCode int `yaml:"exitCode"`
	// Items contain a list of function result
	Items []Result `yaml:"items,omitempty"`
}

// NewResultList returns an instance of ResultList with metadata
// field populated.
func NewResultList() *ResultList {
	return &ResultList{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: ResultListAPIVersion,
				Kind:       ResultListKind,
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: "fnresults",
				},
			},
		},
		Items: []Result{},
	}
}
