// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package trigger

import (
	"errors"
	"os"

	"github.com/goccy/go-yaml"
)

type MockAbstract struct {
	Filepath     string                      `yaml:"-"`
	Metadata     MockAbstractMetadata        `yaml:"metadata"`
	Subscription map[string]MockAbstractData `yaml:"subscription"`
	Event        map[string]MockAbstractData `yaml:"event"`
}

// `metdata` field in YAML
type MockAbstractMetadata struct {
	SupportedTransports []string `yaml:"supported_transports"`
	Type                string   `yaml:"type"`
	Version             string   `yaml:"version"`
}

// `subscription` or `event` field in YAML
type MockAbstractData struct {
	Type     string       `yaml:"type"`
	Ref      *string      `yaml:"ref,omitempty"`
	Default  *string      `yaml:"default,omitempty"`
	Data     *interface{} `yaml:"data,omitempty"` // Actually EventYamlEntry{} but we can't have that in Go!
	Optional *bool        `yaml:"optional,omitempty"`
}

// Basically the same as a MockAbstract, but contains a singular MockAbstractData
type ReferenceAbstract struct {
	Filepath  string                      `yaml:"-"`
	Name      string                      `yaml:"reference_name"`
	Reference map[string]MockAbstractData `yaml:"reference"`
}

func ParseEventYaml(path string) (MockAbstract, error) {
	// Read YAML file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return MockAbstract{}, err
	}

	// Parse into struct
	ey := MockAbstract{
		Filepath: path,
	}
	err = yaml.Unmarshal(data, &ey)
	if err != nil {
		return MockAbstract{}, err
	}

	// Run all the metadata error handling

	// `metadata.supported_transports` field is required to have values
	if len(ey.Metadata.SupportedTransports) == 0 {
		return MockAbstract{}, errors.New("YAML file '" + path + "' requires `metadata.supported_transports` field")
	}

	// `metadata.type` and `metadata.version` need to be set
	if ey.Metadata.Type == "" || ey.Metadata.Version == "" {
		return MockAbstract{}, errors.New("YAML file '" + path + "' requires `metadata.type` and `metadata.version` fields")
	}

	return ey, nil
}

func ParseReferenceYaml(path string) (ReferenceAbstract, error) {
	// Read YAML file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return ReferenceAbstract{}, err
	}

	// Parse into struct
	ey := ReferenceAbstract{
		Filepath: path,
	}
	err = yaml.Unmarshal(data, &ey)
	if err != nil {
		return ReferenceAbstract{}, err
	}

	return ey, nil
}
