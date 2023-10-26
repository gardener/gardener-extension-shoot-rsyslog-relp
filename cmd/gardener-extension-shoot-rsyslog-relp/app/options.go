// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"os"

	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	heartbeatcmd "github.com/gardener/gardener/extensions/pkg/controller/heartbeat/cmd"

	rsyslogrelpcmd "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/cmd/rsyslogrelp"
)

// ExtensionName is the name of the extension.
const ExtensionName = "shoot-rsyslog-relp"

// Options holds configuration passed to the rsyslog relp controller.
type Options struct {
	generalOptions     *controllercmd.GeneralOptions
	rsyslogRelpOptions *rsyslogrelpcmd.Options
	restOptions        *controllercmd.RESTOptions
	managerOptions     *controllercmd.ManagerOptions
	controllerOptions  *controllercmd.ControllerOptions
	lifecycleOptions   *controllercmd.ControllerOptions
	controllerSwitches *controllercmd.SwitchOptions
	reconcileOptions   *controllercmd.ReconcilerOptions
	heartbeatOptions   *heartbeatcmd.Options
	optionAggregator   controllercmd.OptionAggregator
}

// NewOptions creates a new Options instance.
func NewOptions() *Options {
	options := &Options{
		generalOptions:     &controllercmd.GeneralOptions{},
		rsyslogRelpOptions: &rsyslogrelpcmd.Options{},
		restOptions:        &controllercmd.RESTOptions{},
		managerOptions: &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(ExtensionName),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
			MetricsBindAddress:      ":8080",
			HealthBindAddress:       ":8081",
		},
		controllerOptions: &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		},
		lifecycleOptions: &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		},
		heartbeatOptions: &heartbeatcmd.Options{
			ExtensionName:        ExtensionName,
			RenewIntervalSeconds: 30,
			Namespace:            os.Getenv("LEADER_ELECTION_NAMESPACE"),
		},
		reconcileOptions:   &controllercmd.ReconcilerOptions{},
		controllerSwitches: rsyslogrelpcmd.ControllerSwitches(),
	}

	options.optionAggregator = controllercmd.NewOptionAggregator(
		options.generalOptions,
		options.rsyslogRelpOptions,
		options.restOptions,
		options.managerOptions,
		options.controllerOptions,
		controllercmd.PrefixOption("lifecycle-", options.lifecycleOptions),
		controllercmd.PrefixOption("heartbeat-", options.heartbeatOptions),
		options.controllerSwitches,
		options.reconcileOptions,
	)

	return options
}
