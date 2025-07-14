// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
)

// ProjectName calculates the name of the project given the shoot namespace and shoot name.
func ProjectName(namespace, shootName string) string {
	var projectName = strings.TrimPrefix(namespace, "shoot-")
	projectName = strings.TrimSuffix(projectName, "-"+shootName)
	projectName = strings.Trim(projectName, "-")
	return projectName
}
