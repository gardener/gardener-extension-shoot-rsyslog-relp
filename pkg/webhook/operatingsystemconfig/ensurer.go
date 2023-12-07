// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"context"
	"errors"
	"fmt"

	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
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
func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx gcontext.GardenContext, new, _ *[]extensionsv1alpha1.File) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	if cluster.Shoot == nil {
		return errors.New("cluster.shoot is not yet populated")
	}

	if cluster.Shoot.DeletionTimestamp != nil {
		e.logger.Info("Shoot has a deletion timestamp set, skipping the OperatingSystemConfig mutation", "shoot", client.ObjectKeyFromObject(cluster.Shoot))
		return nil
	}

	extension := &extensionsv1alpha1.Extension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shoot-rsyslog-relp",
			Namespace: cluster.ObjectMeta.Name,
		},
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(extension), extension); err != nil {
		return fmt.Errorf("failed to get extension '%s': %w", client.ObjectKeyFromObject(extension), err)
	}

	shootRsyslogRelpConfig := &rsyslog.RsyslogRelpConfig{}
	if extension.Spec.ProviderConfig != nil {
		if _, _, err := e.decoder.Decode(extension.Spec.ProviderConfig.Raw, nil, shootRsyslogRelpConfig); err != nil {
			return fmt.Errorf("failed to decode provider config: %w", err)
		}
	}

	var additionalFiles []extensionsv1alpha1.File

	rsyslogFiles, err := getRsyslogFiles(ctx, e.client, extension.Namespace, shootRsyslogRelpConfig, cluster)
	if err != nil {
		return err
	}
	additionalFiles = append(additionalFiles, rsyslogFiles...)
	additionalFiles = append(additionalFiles, getAuditdFiles()...)

	mergeFiles(new, additionalFiles...)
	return nil
}

func (e *ensurer) EnsureAdditionalUnits(_ context.Context, _ gcontext.GardenContext, new, _ *[]extensionsv1alpha1.Unit) error {
	unit := getRsyslogConfiguratorUnit()
	mergeUnits(new, unit)

	return nil
}

func mergeFiles(files *[]extensionsv1alpha1.File, newFiles ...extensionsv1alpha1.File) {
	var (
		newFilesSet = make(map[string]struct{}, len(newFiles))
		resFiles    = make([]extensionsv1alpha1.File, 0, len(*files))
	)

	for _, f := range newFiles {
		newFilesSet[f.Path] = struct{}{}
	}

	for _, f := range *files {
		if _, ok := newFilesSet[f.Path]; !ok {
			resFiles = append(resFiles, f)
		}
	}

	*files = append(resFiles, newFiles...)
}

func mergeUnits(units *[]extensionsv1alpha1.Unit, newUnits ...extensionsv1alpha1.Unit) {
	var (
		newUnitsSet = make(map[string]struct{}, len(newUnits))
		resUnits    = make([]extensionsv1alpha1.Unit, 0, len(*units))
	)

	for _, u := range newUnits {
		newUnitsSet[u.Name] = struct{}{}
	}

	for _, u := range *units {
		if _, ok := newUnitsSet[u.Name]; !ok {
			resUnits = append(resUnits, u)
		}
	}

	*units = append(resUnits, newUnits...)
}
