// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package trigger

import (
	"errors"
	"os"

	"github.com/goccy/go-yaml"
)

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
