#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

operation="${1:-check}"

echo "> Check Skaffold Dependencies"

check_successful=true

function check() {
  if ! bash "$GARDENER_HACK_DIR"/check-skaffold-deps-for-binary.sh "$operation" --skaffold-file "$1" --binary "$2" --skaffold-config "$3"; then
    check_successful=false
  fi
}

check "skaffold.yaml" "gardener-extension-shoot-rsyslog-relp" "extension"
check "skaffold.yaml" "gardener-extension-shoot-rsyslog-relp-admission" "admission"

if [ "$check_successful" = false ] ; then
  exit 1
fi
