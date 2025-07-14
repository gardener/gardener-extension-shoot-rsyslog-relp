#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0


set -o errexit
set -o nounset
set -o pipefail

echo "> E2E Tests"

ginkgo_flags=

local_address="172.18.255.1"

seed_name="local"

shoot_names=(
  e2e-rslog-relp.local
  e2e-rslog-tls.local
  e2e-rslog-hib.local
  e2e-rslog-fd.local
  e2e-rslog-filter.local
)

if [ -n "${CI:-}" -a -n "${ARTIFACTS:-}" ]; then
  for shoot in "${shoot_names[@]}" ; do
    printf "\n$local_address api.%s.external.$seed_name.gardener.cloud\n$local_address api.%s.internal.local.gardener.cloud\n" $shoot $shoot >>/etc/hosts
  done
else
  missing_entries=()

  for shoot in "${shoot_names[@]}"; do
    for ip in internal external; do
      if ! grep -q -x "$local_address api.$shoot.$ip.local.gardener.cloud" /etc/hosts; then
        missing_entries+=("$local_address api.$shoot.$ip.local.gardener.cloud")
      fi
    done
  done

  if [ ${#missing_entries[@]} -gt 0 ]; then
    printf "Hostnames for the following Shoots are missing in /etc/hosts:\n"
    for entry in "${missing_entries[@]}"; do
      printf " - %s\n" "$entry"
    done
    printf "To access shoot clusters and run e2e tests, you have to extend your /etc/hosts file.\nPlease refer to https://github.com/gardener/gardener/blob/master/docs/deployment/getting_started_locally.md#accessing-the-shoot-cluster\n\n"
    exit 1
  fi
fi

# reduce flakiness in contended pipelines
export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=5s
export GOMEGA_DEFAULT_EVENTUALLY_POLLING_INTERVAL=200ms
# if we're running low on resources, it might take longer for tested code to do something "wrong"
# poll for 5s to make sure, we're not missing any wrong action
export GOMEGA_DEFAULT_CONSISTENTLY_DURATION=5s
export GOMEGA_DEFAULT_CONSISTENTLY_POLLING_INTERVAL=200ms

GO111MODULE=on ginkgo run --timeout=1h $ginkgo_flags --v --show-node-events "$@"
