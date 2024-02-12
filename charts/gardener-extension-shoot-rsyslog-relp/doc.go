// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "../../vendor/github.com/gardener/gardener/hack/generate-controller-registration.sh shoot-rsyslog-relp . $(cat ../../VERSION) ../../example/controller-registration.yaml Extension:shoot-rsyslog-relp"

// Package chart enables go:generate support for generating the correct controller registration.
package chart
