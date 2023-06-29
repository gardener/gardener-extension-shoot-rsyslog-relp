// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

func decodeRsyslogRelpConfig(decoder runtime.Decoder, config *runtime.RawExtension, fldPath *field.Path) (*rsyslog.RsyslogRelpConfig, error) {
	if config == nil {
		return nil, field.Required(fldPath, "Rsyslog relp configuration is required when using gardener-extension-shoot-rsyslog-relp")
	}

	rsyslogRelpConfig := &rsyslog.RsyslogRelpConfig{}
	if err := runtime.DecodeInto(decoder, config.Raw, rsyslogRelpConfig); err != nil {
		return nil, err
	}

	return rsyslogRelpConfig, nil
}
