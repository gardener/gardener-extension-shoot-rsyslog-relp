// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog
// +k8s:openapi-gen=true
// +k8s:defaulter-gen=TypeMeta

//go:generate gen-crd-api-reference-docs -api-dir github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1 -config ../../../../hack/api-reference/rsyslog.json -template-dir "$GARDENER_HACK_DIR/api-reference/template" -out-file ../../../../hack/api-reference/rsyslog.md

// Package v1alpha1 contains the Rsyslog Relp Shoot extension.
// +groupName=rsyslog-relp.extensions.gardener.cloud
package v1alpha1 // import "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
