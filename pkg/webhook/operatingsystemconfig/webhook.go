// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/original/components/kubelet"
	oscutils "github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	namespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: v1beta1constants.LabelExtensionPrefix + constants.ExtensionType, Operator: metav1.LabelSelectorOpIn, Values: []string{"true"}},
		},
	}

	webhook := &extensionswebhook.Webhook{
		Name:     "operating-system-config",
		Provider: "",
		Types:    types,
		Target:   extensionswebhook.TargetSeed,
		Path:     "/operating-system-config",
		Webhook:  &admission.Webhook{Handler: handler},
		Selector: namespaceSelector,
	}

	return webhook, nil
}
