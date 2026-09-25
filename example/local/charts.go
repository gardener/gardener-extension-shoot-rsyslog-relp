// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"embed"
	"path/filepath"
)

var (
	// Charts embeds the local charts in embed.FS.
	//go:embed charts
	Charts embed.FS

	// ChartPath is the path to the example local charts.
	ChartsPath = filepath.Join("charts")
	// RsyslogRelpEchoServerChartPath is the path to the rsyslog-relp-echo-server local chart.
	RsyslogRelpEchoServerChartPath = filepath.Join(ChartsPath, "rsyslog-relp-echo-server")
)
