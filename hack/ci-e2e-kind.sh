#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o nounset
set -o pipefail
set -o errexit


REPO_ROOT="$(readlink -f $(dirname ${0})/..)"
GARDENER_VERSION=$(go list -m -f '{{.Version}}' github.com/gardener/gardener)

if [[ ! -d "$REPO_ROOT/gardener" ]]; then
  git clone --branch $GARDENER_VERSION https://github.com/gardener/gardener.git
else
  (cd "$REPO_ROOT/gardener" && git checkout $GARDENER_VERSION)
fi

source "${REPO_ROOT}"/gardener/hack/ci-common.sh

clamp_mss_to_pmtu

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
