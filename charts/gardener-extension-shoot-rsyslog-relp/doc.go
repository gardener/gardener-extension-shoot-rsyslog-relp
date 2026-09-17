// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "bash $GARDENER_HACK_DIR/generate-controller-registration.sh extension-shoot-rsyslog-relp . $(cat ../../VERSION) ../../example/controller-registration.yaml Extension:shoot-rsyslog-relp"

// Package chart enables go:generate support for generating the correct controller registration.
package chart
