// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package trigger

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type MockEventBase struct {
	Subscription map[string]any `json:"subscription"`
	Event        map[string]any `json:"event"`
}

var mockEvents []MockAbstract
var referenceAbstracts []ReferenceAbstract

func RegisterAllEvents() error {
	// Find directory holding YAML EventSub templates
	exeDir, err := os.Executable()
	if err != nil {
		return err
	}
	templatesBaseDir := path.Join(filepath.Dir(exeDir), "templates", "events")
	refBaseDir := path.Join(templatesBaseDir, "_ref")
	hiddenDirectoriesPath := path.Join(templatesBaseDir, "_")

	// Go through template file directory to find all event files
	files := []string{}
	err = filepath.Walk(templatesBaseDir, func(fpath string, info fs.FileInfo, err error) error {
		if !info.IsDir() && // Must be a file, not a directory
			!strings.HasPrefix(fpath, hiddenDirectoriesPath) && // Can't be in any base-level folder starting with _ (e.g. /_ref)
			strings.HasSuffix(strings.ToLower(fpath), ".yaml") { // Must be a .yaml file
			files = append(files, fpath)
		}
		return nil
	})
	if err != nil {
		return errors.New("Could not read EventSub yaml files: " + err.Error())
	}

	// Read and store all the event files
	for _, f := range files {
		abstract, err := ParseEventYaml(f)
		if err != nil {
			return err
		}

		// Check for duplicates and register
		ok, duplicateFilepath := RegisterSubscriptionType(abstract)
		if !ok {
			return errors.New("Duplicate subscription type/version pair:\n - " + abstract.Filepath + "\n - " + duplicateFilepath)
		}
	}

	// Go through /_ref directory to find all the reference files
	files = []string{}
	err = filepath.Walk(refBaseDir, func(fpath string, info fs.FileInfo, err error) error {
		if !info.IsDir() && // Must be a file, not a directory
			strings.HasSuffix(strings.ToLower(fpath), ".yaml") { // Must be a .yaml file
			files = append(files, fpath)
		}
		return nil
	})
	if err != nil {
		return errors.New("Could not read EventSub yaml files: " + err.Error())
	}

	// Read and store all the reference files
	for _, f := range files {
		abstract, err := ParseReferenceYaml(f)
		if err != nil {
			return err
		}

		// Check for duplicates and register
		ok, duplicateFilepath := RegisterReference(abstract)
		if !ok {
			return errors.New("Duplicate reference file:\n - " + abstract.Filepath + "\n - " + duplicateFilepath)
		}
	}

	return nil
}

func RegisterSubscriptionType(eventAbstract MockAbstract) (bool, string) {
	// Look for duplicates
	for _, sub := range mockEvents {
		if sub.Metadata.Type == eventAbstract.Metadata.Type && sub.Metadata.Version == eventAbstract.Metadata.Version {
			return false, sub.Filepath
		}
	}

	mockEvents = append(mockEvents, eventAbstract)

	return true, ""
}

func RegisterReference(refAbstract ReferenceAbstract) (bool, string) {
	// Look for duplicates
	for _, ref := range referenceAbstracts {
		if ref.Name == refAbstract.Name {
			return false, ref.Filepath
		}
	}

	referenceAbstracts = append(referenceAbstracts, refAbstract)

	return true, ""
}

func NEW_GetByTriggerAndTransportAndVersion(trigger string, transport string, version string) (*MockAbstract, error) {
	validEventBadVersions := []string{}
	var latestEventSeen *MockAbstract

	for _, sub := range mockEvents {
		if trigger == sub.Metadata.Type {
			// Found an event type that match's user input

			// Check if transport is valid
			validTransport := false
			for _, t := range sub.Metadata.SupportedTransports {
				if transport == t {
					validTransport = true
					break
				}
			}
			if !validTransport {
				if strings.EqualFold(transport, "websocket") {
					return nil, errors.New("Invalid transport. This event is not available via WebSockets.")
				}
				return nil, fmt.Errorf("Invalid transport. This event supports the following transport types: %v", strings.Join(sub.Metadata.SupportedTransports, ", "))
			}

			// Check for matching verison; Assumes version is not empty but doesn't matter performance-wise
			if version == sub.Metadata.Version {
				return &sub, nil
			} else {
				validEventBadVersions = append(validEventBadVersions, sub.Metadata.Version)
				latestEventSeen = &sub
			}
		}
	}

	// When no version is given, and there's only one version available, use the default version.
	if version == "" && len(validEventBadVersions) == 1 {
		return latestEventSeen, nil
	}

	// Error for events with non-existent version used
	if len(validEventBadVersions) != 0 {
		sort.Strings(validEventBadVersions)
		errStr := fmt.Sprintf("Invalid version given. Valid version(s): %v", strings.Join(validEventBadVersions, ", "))
		if version == "" {
			errStr += "\nUse --version to specify"
		}
		return nil, errors.New(errStr)
	}

	// Default error
	return nil, errors.New("Invalid event") // TODO
}

func GenerateSubscriptionObject(mockAbstract MockAbstract, p TriggerParameters) (map[string]any, error) {
	subObj := make(map[string]any)

	for identifier, innards := range mockAbstract.Subscription {
		/*if innards.Type == "string" {
			subObj[identifier] = "Test"
		} else if innards.Type == "string[]" {
			subObj[identifier] = []string{}
		} else if innards.Type == "int" {
			subObj[identifier] = 0
		} else if innards.Type == "int[]" {
			subObj[identifier] = []int{}
		} else if innards.Type == "object" {
			// TODO
			subObj[identifier] = make(map[string]any)
		} else if innards.Type == "object[]" {
			subObj[identifier] = []map[string]any{}
		}*/

		if innards.Ref != nil {
			handled := false

			// Check if this should be handled by a processed reference file
			for _, refAbstract := range referenceAbstracts {
				if refAbstract.Name == *innards.Ref {
					// Generate object based on the reference file
					// TODO: Write this recursively later on, when there's reusable code
					// Current limitations: Only goes 1 deep; no recursive references because of this

					fmt.Printf("TODO: Handle ref %v\n", refAbstract.Name)

					handled = true
				}
			}

			if handled {
				continue
			}

			switch *innards.Ref {
			case "event_id":
				if innards.Type != "string" {
					return nil, fmt.Errorf("Parsing error: ref `event_id` must be matched with type `string`")
				}
				subObj[identifier] = p.EventID

			case "subscription_type":
				if innards.Type != "string" {
					return nil, fmt.Errorf("Parsing error: ref `subscription_type` must be matched with type `string`")
				}
				subObj[identifier] = mockAbstract.Metadata.Type

			case "version":
				if innards.Type != "string" {
					return nil, fmt.Errorf("Parsing error: ref `version` must be matched with type `string`")
				}
				subObj[identifier] = mockAbstract.Metadata.Version

			case "status":
				if innards.Type != "string" {
					return nil, fmt.Errorf("Parsing error: ref `status` must be matched with type `string`")
				}
				subObj[identifier] = p.SubscriptionStatus

			case "timestamp":
				if innards.Type != "string" {
					return nil, fmt.Errorf("Parsing error: ref `timestamp` must be matched with type `string`")
				}
				subObj[identifier] = p.Timestamp

			case "cost":
				if innards.Type != "int" {
					return nil, fmt.Errorf("Parsing error: ref `cost` must be matched with type `int`")
				}
				subObj[identifier] = p.Cost
			}
		}

		// TODO: Handle non-reference fields

	}

	return subObj, nil
}

func GenerateEventObject() (map[string]MockAbstractData, error) {
	return nil, nil
}
