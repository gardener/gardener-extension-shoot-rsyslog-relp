#!/bin/bash

# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o nounset
set -o pipefail
set -o errexit

repo_root="$(readlink -f $(dirname ${0})/..)"

latest_commit=$(git rev-parse HEAD)
version="${EFFECTIVE_VERSION:-$latest_commit}"-$RANDOM

docker build --build-arg EFFECTIVE_VERSION=$version --tag shoot-rsyslog-relp-local:$version --target gardener-extension-shoot-rsyslog-relp $repo_root
docker build --build-arg EFFECTIVE_VERSION=$version --tag shoot-rsyslog-relp-admission-local:$version --target gardener-extension-shoot-rsyslog-relp-admission $repo_root

kind load docker-image shoot-rsyslog-relp-local:$version --name gardener-local
kind load docker-image shoot-rsyslog-relp-admission-local:$version --name gardener-local

mkdir -p $repo_root/tmp
cp -f $repo_root/example/controller-registration.yaml $repo_root/tmp/controller-registration.yaml
yq -i e "(select (.providerConfig.values.image) | .providerConfig.values.image.tag) |= \"$version\"" $repo_root/tmp/controller-registration.yaml
yq -i e '(select (.providerConfig.values.image) | .providerConfig.values.image.repository) |= "docker.io/library/shoot-rsyslog-relp-local"' $repo_root/tmp/controller-registration.yaml
yq -i e '(select (.providerConfig.values.image) | .providerConfig.values.image.pullPolicy) |= "IfNotPresent"' $repo_root/tmp/controller-registration.yaml

kubectl apply -f "$repo_root/tmp/controller-registration.yaml"

# install admission controller
path_tls="$repo_root/hack/admission-tls"
admission_tls="$repo_root/tmp/admission-tls"
mkdir -p "$admission_tls"

# generate webhook TLS certificate if not yet done
cert_name="shoot-rsyslog-relp-admission"
ca_name="${cert_name}-ca"
if [[ ! -f "$admission_tls/${ca_name}.pem" ]]; then
  cfssl gencert \
    -initca "$path_tls/${ca_name}-csr.json" | cfssljson -bare "$admission_tls/$ca_name" -
fi

if [[ ! -f "$admission_tls/${cert_name}-tls.pem" ]]; then
  cfssl gencert \
    -profile=server \
    -ca="$admission_tls/${ca_name}.pem" \
    -ca-key="$admission_tls/${ca_name}-key.pem" \
    -config="$path_tls/ca-config.json" \
    "$path_tls/${cert_name}-config.json" | cfssljson -bare "$admission_tls/${cert_name}-tls"
fi

admission_chart_path="$repo_root/charts/gardener-extension-shoot-rsyslog-relp-admission"

helm upgrade \
    --install \
    --wait \
    --values "$admission_chart_path/values.yaml" \
    --set application.enabled="true" \
    --set runtime.enabled="true" \
    --set runtime.image.repository="docker.io/library/shoot-rsyslog-relp-admission-local" \
    --set runtime.image.tag="$version" \
    --set runtime.webhookConfig.tls.crt="$(cat $repo_root/tmp/admission-tls/shoot-rsyslog-relp-admission-tls.pem)" \
    --set runtime.webhookConfig.tls.key="$(cat $repo_root/tmp/admission-tls/shoot-rsyslog-relp-admission-tls-key.pem)" \
    --set application.webhookConfig.caBundle="$(cat $repo_root/tmp/admission-tls/shoot-rsyslog-relp-admission-ca.pem)" \
    --namespace garden \
    "shoot-rsylog-relp-admission" \
    "$admission_chart_path"
