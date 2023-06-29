// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config"
)

// Config contains configuration for the shoot rsyslog relp extension.
type Config struct {
	config.Configuration
}
