// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/original/components/kubelet"
	oscutils "github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/utils"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var logger = log.Log.WithName("operating-system-config-webhook")

// New returns a new mutating webhook that adds the required rsyslog configuration files to the OperatingSystemConfig resource.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Adding webhook to manager")

	fciCodec := oscutils.NewFileContentInlineCodec()

	decoder := serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder()

	mutator := genericmutator.NewMutator(
		mgr,
		NewEnsurer(mgr.GetClient(), decoder, logger),
		oscutils.NewUnitSerializer(),
		kubelet.NewConfigCodec(fciCodec),
		fciCodec,
		logger,
	)
	types := []extensionswebhook.Type{
		{Obj: &extensionsv1alpha1.OperatingSystemConfig{}},
	}
	handler, err := extensionswebhook.NewBuilder(mgr, logger).WithMutator(mutator, types...).Build()
	if err != nil {
		return nil, err
	}

	namespaceSelector := extensionswebhook.BuildExtensionTypeNamespaceSelector(constants.ExtensionType, []extensionsv1alpha1.ExtensionClass{
		extensionsv1alpha1.ExtensionClassShoot,
	})

	webhook := &extensionswebhook.Webhook{
		Name:              "operating-system-config",
		Types:             types,
		Target:            extensionswebhook.TargetSeed,
		Path:              "/operating-system-config",
		Webhook:           &admission.Webhook{Handler: handler},
		NamespaceSelector: namespaceSelector,
	}

	return webhook, nil
}
