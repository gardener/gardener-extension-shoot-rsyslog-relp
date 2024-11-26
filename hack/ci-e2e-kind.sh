#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o nounset
set -o pipefail
set -o errexit

# If running in prow, we need to ensure that garden.local.gardener.cloud resolves to localhost
ensure_glgc_resolves_to_localhost() {
  if [ -n "${CI:-}" ]; then
    printf "\n127.0.0.1 garden.local.gardener.cloud\n" >> /etc/hosts
    printf "\n::1 garden.local.gardener.cloud\n" >> /etc/hosts
  fi
}

REPO_ROOT="$(readlink -f $(dirname ${0})/..)"
GARDENER_VERSION=$(go list -m -f '{{.Version}}' github.com/gardener/gardener)

if [[ ! -d "$REPO_ROOT/gardener" ]]; then
  git clone --branch $GARDENER_VERSION https://github.com/gardener/gardener.git
else
  (cd "$REPO_ROOT/gardener" && git checkout $GARDENER_VERSION)
fi

source "${REPO_ROOT}"/gardener/hack/ci-common.sh

clamp_mss_to_pmtu

ensure_glgc_resolves_to_localhost

# test setup
make -C "${REPO_ROOT}"/gardener kind-up
export KUBECONFIG=$REPO_ROOT/gardener/example/gardener-local/kind/local/kubeconfig

# export all container logs and events after test execution
trap '{
  export_artifacts "gardener-local"
  make -C "${REPO_ROOT}"/gardener kind-down
}' EXIT

make -C "${REPO_ROOT}"/gardener gardener-up
make extension-up
make test-e2e-local
make extension-down
make -C "${REPO_ROOT}"/gardener  gardener-down
