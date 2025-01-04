// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package trigger

type MockEventBase struct {
	Subscription map[string]any `json:"subscription"`
	Event        map[string]any `json:"event"`
}
