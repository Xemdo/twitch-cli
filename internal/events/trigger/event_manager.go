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

	"github.com/goccy/go-yaml"
)

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
	var subObj map[string]any

	data, err := handleRoot(mockAbstract.Subscription, mockAbstract, p)
	if err != nil {
		return nil, err
	}

	subObj = data

	return subObj, nil
}

func handleRoot(data map[string]MockAbstractData, root MockAbstract, p TriggerParameters) (map[string]any, error) {
	workingField := make(map[string]any)

	for identifier, innards := range data {
		if innards.Ref != nil && *innards.Ref != "--" {
			// This is a reference field, so handle it as such

			resolvedRef, err := resolveReference(identifier, innards, root, p)
			if err != nil {
				return nil, err
			}
			workingField[identifier] = resolvedRef
		} else {
			// Not a reference field, handle normally

			if innards.Type == "object" {
				// Handle recursively
				if innards.Data == nil {
					return nil, fmt.Errorf("Parsing error: in identifier `%v`, `data` field must exist when `type` is set to `object` (in file %v)", identifier, root.Filepath)
				}

				newData, err := convertToMockAbstractDataMap((*innards.Data).(map[string]interface{}))
				if err != nil {
					return nil, fmt.Errorf("Parsing error: Unexpected error when converting `data` object for identifier `%v`: %v", identifier, err)
				}

				childData, err := handleRoot(newData, root, p)
				if err != nil {
					return nil, err
				}

				workingField[identifier] = childData
			} else {
				// Use the default, or use the backup default

				if innards.Default != nil {
					workingField[identifier] = innards.Default
				} else {
					backupDefault, err := getBackupDefault(innards.Type, identifier, root)
					if err != nil {
						return nil, err
					}
					workingField[identifier] = backupDefault
				}
			}
		}
	}

	return workingField, nil
}

func convertToMockAbstractDataMap(interfaceMap map[string]interface{}) (map[string]MockAbstractData, error) {
	d, err := yaml.Marshal(interfaceMap)
	if err != nil {
		return nil, err
	}

	newMap := map[string]MockAbstractData{}
	err = yaml.Unmarshal(d, &newMap)
	if err != nil {
		return nil, err
	}

	return newMap, nil
}

func resolveReference(identifier string, innards MockAbstractData, root MockAbstract, p TriggerParameters) (any, error) {
	// Check if this should be handled by a processed reference file
	for _, refAbstract := range referenceAbstracts {
		if refAbstract.Name == *innards.Ref {
			// Generate object based on the reference file
			// TODO: Write this recursively later on, when there's reusable code
			// Current limitations: Only goes 1 deep; no recursive references because of this

			fmt.Printf("TODO: Handle ref: %v\n", refAbstract.Name)

			return "TODO", nil
		}
	}

	// Resolve the reference using "built-ins"
	resolvedRef, err := resolveBuiltInReference(identifier, innards, root, p)
	if err != nil {
		return nil, err
	}

	return resolvedRef, nil
}

func resolveBuiltInReference(identifier string, innards MockAbstractData, root MockAbstract, p TriggerParameters) (any, error) {
	switch *innards.Ref {
	case "event_id":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `event_id` must be matched with type `string`")
		}
		return p.EventID, nil

	case "subscription_type":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `subscription_type` must be matched with type `string`")
		}
		return root.Metadata.Type, nil

	case "subscription_version":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `subscription_version` must be matched with type `string`")
		}
		return root.Metadata.Version, nil

	case "status":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `status` must be matched with type `string`")
		}
		return p.SubscriptionStatus, nil

	case "timestamp":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `timestamp` must be matched with type `string`")
		}
		return p.Timestamp, nil

	case "cost":
		if innards.Type != "int" {
			return nil, fmt.Errorf("Parsing error: ref `cost` must be matched with type `int`")
		}
		return p.Cost, nil

	case "target_id":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `target_id` must be matched with type `string`")
		}
		return p.ToUser, nil

	case "transport_method":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `transport_method` must be matched with type `string`")
		}
		return p.Transport, nil

	case "transport_callback":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `transport_callback` must be matched with type `string`")
		}
		if p.Transport == "webhook" {
			return "null", nil // TODO: Get this from somewhere else in case there's actually a callback
		} else {
			return nil, nil
		}

	case "transport_session_id":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `transport_method` must be matched with type `string`")
		}
		if p.Transport == "websocket" {
			return "null", nil // TODO: Get this from somewhere else in case there's actually a session ID
		} else {
			return nil, nil
		}

	case "transport_connected_at":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `transport_method` must be matched with type `string`")
		}
		if p.Transport == "websocket" {
			return "null", nil // TODO: Get this from somewhere else in case there's actually a timestamp
		} else {
			return nil, nil
		}

	case "transport_disconnected_at":
		if innards.Type != "string" {
			return nil, fmt.Errorf("Parsing error: ref `transport_method` must be matched with type `string`")
		}
		if p.Transport == "websocket" {
			return "null", nil // TODO: Get this from somewhere else in case there's actually a timestamp
		} else {
			return nil, nil
		}
	}

	// TODO: Replace with error
	fmt.Printf("Unhandled ref '%v' on identifier '%v'. Using string default instead.\n", *innards.Ref, identifier)
	return "", nil
}

func getBackupDefault(dataType string, identifier string, root MockAbstract) (any, error) {
	switch dataType {
	case "string":
		return "", nil
	case "string[]":
		return []string{}, nil
	case "int":
		return 0, nil
	case "int[]":
		return []int{}, nil
	case "object":
		return make(map[string]any), nil
	case "object[]":
		return []map[string]any{}, nil
	default:
		return nil, fmt.Errorf("Unexpected type `%v` for identifier `%v` (in file %v)", dataType, identifier, root.Filepath)
	}
}

func GenerateEventObject() (map[string]MockAbstractData, error) {
	return nil, nil
}
