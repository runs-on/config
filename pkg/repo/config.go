package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	configvalidate "github.com/runs-on/config/pkg/validate"
	"gopkg.in/yaml.v3"
)

const DefaultEnvironment = "production"

type StringArray []string

func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*a = []string{single}
		return nil
	}

	var multi []string
	if err := unmarshal(&multi); err != nil {
		return err
	}

	*a = multi
	return nil
}

type IntArray []int

func (a *IntArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single int
	if err := unmarshal(&single); err == nil {
		*a = []int{single}
		return nil
	}

	var singleString string
	if err := unmarshal(&singleString); err == nil {
		single, err := strconv.Atoi(singleString)
		if err == nil {
			*a = []int{single}
			return nil
		}
	}

	var multi []int
	if err := unmarshal(&multi); err == nil {
		*a = multi
		return nil
	}

	var multiString []string
	if err := unmarshal(&multiString); err == nil {
		multi := make([]int, len(multiString))
		for i, s := range multiString {
			val, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", s)
			}
			multi[i] = val
		}
		*a = multi
		return nil
	}

	return fmt.Errorf("failed to unmarshal IntArray")
}

type BoolOrString string

func (b *BoolOrString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*b = BoolOrString("")
		return nil
	}

	var boolValue bool
	if err := json.Unmarshal(data, &boolValue); err == nil {
		*b = BoolOrString(strconv.FormatBool(boolValue))
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		lowerValue := strings.ToLower(stringValue)
		if lowerValue == "true" || lowerValue == "false" {
			*b = BoolOrString(lowerValue)
			return nil
		}
	}

	*b = BoolOrString("")
	return nil
}

func (b *BoolOrString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var boolValue bool
	if err := unmarshal(&boolValue); err == nil {
		*b = BoolOrString(strconv.FormatBool(boolValue))
		return nil
	}

	var stringValue string
	if err := unmarshal(&stringValue); err == nil {
		lowerValue := strings.ToLower(stringValue)
		if lowerValue == "true" || lowerValue == "false" {
			*b = BoolOrString(lowerValue)
			return nil
		}
	}

	*b = BoolOrString("")
	return nil
}

type RepoConfig struct {
	Extends string                `yaml:"_extends"`
	Runners map[string]RunnerSpec `yaml:"runners"`
	Images  map[string]ImageSpec  `yaml:"images"`
	Pools   map[string]PoolSpec   `yaml:"pools"`
	Admins  []string              `yaml:"admins"`
}

func (r *RepoConfig) ToYAML() (string, error) {
	yaml, err := yaml.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(yaml), nil
}

func (r *RepoConfig) String() string {
	yaml, _ := r.ToYAML()
	return yaml
}

func Parse(content string) (RepoConfig, error) {
	var repoConfig RepoConfig
	err := yaml.Unmarshal([]byte(content), &repoConfig)
	if err != nil {
		return repoConfig, err
	}

	if repoConfig.Runners == nil {
		repoConfig.Runners = map[string]RunnerSpec{}
	}
	if repoConfig.Images == nil {
		repoConfig.Images = map[string]ImageSpec{}
	}
	if repoConfig.Pools == nil {
		repoConfig.Pools = map[string]PoolSpec{}
	}
	return repoConfig, nil
}

type Diagnostic = configvalidate.Diagnostic

func Validate(ctx context.Context, content string, sourceName string) ([]Diagnostic, error) {
	return configvalidate.ValidateString(ctx, content, sourceName)
}

func DiagnosticsError(prefix string, diags []Diagnostic) error {
	return configvalidate.DiagnosticsError(prefix, diags)
}

type RunnerSpec struct {
	Id         string       `yaml:"id,omitempty"`
	Cpu        IntArray     `yaml:"cpu,omitempty"`
	Ram        IntArray     `yaml:"ram,omitempty"`
	Disk       string       `yaml:"disk,omitempty"`
	Volume     string       `yaml:"volume,omitempty"`
	Retry      StringArray  `yaml:"retry,omitempty"`
	Extras     StringArray  `yaml:"extras,omitempty"`
	Ssh        BoolOrString `yaml:"ssh,omitempty"`
	Private    BoolOrString `yaml:"private,omitempty"`
	NestedVirt BoolOrString `yaml:"nested-virt,omitempty"`
	Spot       string       `yaml:"spot,omitempty"`
	Family     StringArray  `yaml:"family,omitempty"`
	Image      string       `yaml:"image,omitempty"`
	Preinstall string       `yaml:"preinstall,omitempty"`
	Prerun     string       `yaml:"prerun,omitempty"`
	Tags       StringArray  `yaml:"tags,omitempty"`
}

func (r *RunnerSpec) GetId() string {
	if r.Id != "" {
		return r.Id
	}
	cpuStrings := make([]string, len(r.Cpu))
	for i, v := range r.Cpu {
		cpuStrings[i] = strconv.Itoa(v)
	}

	ramStrings := make([]string, len(r.Ram))
	for i, v := range r.Ram {
		ramStrings[i] = strconv.Itoa(v)
	}

	result := []string{
		"family",
		strings.Join(r.Family, "+"),
		"cpu",
		strings.Join(cpuStrings, "+"),
		"ram",
		strings.Join(ramStrings, "+"),
	}
	return strings.Join(result, "-")
}

type ImageSpec struct {
	Id             string            `yaml:"id,omitempty"`
	Platform       string            `yaml:"platform,omitempty"`
	Arch           string            `yaml:"arch,omitempty"`
	Name           string            `yaml:"name,omitempty"`
	Owner          string            `yaml:"owner,omitempty"`
	Preinstall     string            `yaml:"preinstall,omitempty"`
	Prerun         string            `yaml:"prerun,omitempty"`
	Ami            string            `yaml:"ami,omitempty"`
	MainDiskSize   int               `yaml:"main_disk_size,omitempty"`
	RootDeviceName string            `yaml:"root_device_name,omitempty"`
	Tags           map[string]string `yaml:"tags,omitempty"`
}

func (i *ImageSpec) GetId() string {
	if i.Id != "" {
		return i.Id
	}
	return i.Ami
}

type ScheduleMatch struct {
	Day  []string `yaml:"day,omitempty" json:"day,omitempty"`
	Time []string `yaml:"time,omitempty" json:"time,omitempty"`
}

type PoolSchedule struct {
	Name    string         `yaml:"name" json:"name"`
	Stopped int            `yaml:"stopped" json:"stopped"`
	Hot     int            `yaml:"hot" json:"hot"`
	Match   *ScheduleMatch `yaml:"match,omitempty" json:"match,omitempty"`
}

type PoolSpec struct {
	Version  string
	Env      string         `yaml:"env"`
	Timezone string         `yaml:"timezone"`
	Schedule []PoolSchedule `yaml:"schedule"`
	Runner   string         `yaml:"runner"`
	MaxSurge int            `yaml:"max_surge"`
}

func (p *PoolSpec) UnmarshalYAML(node *yaml.Node) error {
	type poolSpecAlias PoolSpec
	var aux struct {
		*poolSpecAlias `yaml:",inline"`
		Environment    string `yaml:"environment"`
	}
	aux.poolSpecAlias = (*poolSpecAlias)(p)

	if err := node.Decode(&aux); err != nil {
		return err
	}
	if p.Env == "" && aux.Environment != "" {
		p.Env = aux.Environment
	}
	return nil
}
