// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package trigger

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

type MockEventBase struct {
	Subscription map[string]any `json:"subscription"`
	Event        map[string]any `json:"event"`
}
