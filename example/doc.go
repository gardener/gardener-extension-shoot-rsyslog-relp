// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "extension-generator --name=extension-shoot-rsyslog-relp --provider-type=shoot-rsyslog-relp --component-category=extension --extension-oci-repository=europe-docker.pkg.dev/gardener-project/public/charts/gardener/extensions/shoot-rsyslog-relp:$(cat ../VERSION) --admission-runtime-oci-repository=europe-docker.pkg.dev/gardener-project/public/charts/gardener/extensions/shoot-rsyslog-relp-admission-runtime:$(cat ../VERSION) --admission-application-oci-repository=europe-docker.pkg.dev/gardener-project/public/charts/gardener/extensions/shoot-rsyslog-relp-admission-application:$(cat ../VERSION) --destination=./extension/registration/base/extension.yaml"
//go:generate sh -c "$TOOLS_BIN_DIR/kustomize build ./extension/registration -o ./extension.yaml"

package example
