// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package charts

import (
	"embed"
	"path/filepath"
)

var (
	// InternalChart embeds the internal charts in embed.FS
	//
	//go:embed internal
	InternalChart embed.FS

	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join("internal")
	// RsyslogConfiguratorChartPath is the path for internal Rsyslog Relp Chart.
	RsyslogConfiguratorChartPath = filepath.Join(InternalChartsPath, "rsyslog-relp-configurator")
	// RsyslogConfigurationCleanerChartPath is the path for the internal Rsyslog Relp Chart that will clean up the configuration.
	RsyslogConfigurationCleanerChartPath = filepath.Join(InternalChartsPath, "rsyslog-relp-configuration-cleaner")
)
