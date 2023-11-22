// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"context"

	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(client client.Client, decoder runtime.Decoder, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		client:  client,
		decoder: decoder,
		logger:  logger.WithName("rsyslog-relp-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client  client.Client
	decoder runtime.Decoder
	logger  logr.Logger
}

// EnsureAdditionalFiles ensures that the rsyslog configuration files are added to the <new> files.
func (e *ensurer) EnsureAdditionalFiles(_ context.Context, _ gcontext.GardenContext, new, _ *[]extensionsv1alpha1.File) error {
	// TODO(plkokanov): retrieve cluster resource
	// TODO(plkokanov): retrieve extension resource if shoot is not hibernated and is not in deletion
	// TODO(plkokanov): retrieve referenced secret if tls is enabled in the extension resource
	// TODO(plkokanov): add secret certificate files to osc as secret ref if tls is enabled
	// TODO(plkokanov): fill in configure-rsyslog.sh template and add it to osc
	// TODO(plkokanov): fill in 60-audit.conf template and add it to here
	// TODO(plkokanov): once metrics are added, fill in process-stats.sh and add it to osc

	return nil
}

func (e *ensurer) EnsureAdditionalUnits(_ context.Context, _ gcontext.GardenContext, new, _ *[]extensionsv1alpha1.Unit) error {
	//TODO(plkokanov): add rsyslog-configurator.service to osc units

	return nil
}
