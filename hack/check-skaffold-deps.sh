#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

echo "> Check Skaffold Dependencies"

check_successful=true

out_dir=$(mktemp -d)
function cleanup_output {
  rm -rf "$out_dir"
}
trap cleanup_output EXIT

function check() {
  skaffold_file="$1"
  binary_name="$2"
  skaffold_config_name="$3"

  skaffold_yaml="$(cat "$(dirname "$0")/../$skaffold_file")"

  path_current_skaffold_dependencies="${out_dir}/current-$skaffold_file-deps-$binary_name.txt"
  path_actual_dependencies="${out_dir}/actual-$skaffold_file-deps-$binary_name.txt"

  echo "$skaffold_yaml" |\
    yq eval "select(.metadata.name == \"$skaffold_config_name\") | .build.artifacts[] | select(.ko.main == \"./cmd/$binary_name\") | .ko.dependencies.paths[]?" - |\
    sort |\
    uniq > "$path_current_skaffold_dependencies"

  go list -f '{{ join .Deps "\n" }}' "./cmd/$binary_name" |\
    grep "github.com/gardener/gardener-extension-shoot-rsyslog-relp/" |\
    sed 's/github\.com\/gardener\/gardener-extension-shoot-rsyslog-relp\///g' |\
    sort |\
    uniq > "$path_actual_dependencies"

  # always add vendor directory and VERSION file
  echo "vendor" >> "$path_actual_dependencies"
  echo "VERSION" >> "$path_actual_dependencies"

  # sort dependencies
  sort -o $path_current_skaffold_dependencies{,}
  sort -o $path_actual_dependencies{,}

  echo -n ">> Checking defined dependencies in Skaffold config '$skaffold_config_name' for '$binary_name' in '$skaffold_file'..."
  if ! diff="$(diff "$path_current_skaffold_dependencies" "$path_actual_dependencies")"; then
    check_successful=false

    echo
    echo ">>> The following actual dependencies are missing in $skaffold_file (need to be added):"
    echo "$diff" | grep '>' | awk '{print $2}'
    echo
    echo ">>> The following dependencies defined in $skaffold_file are not needed actually (need to be removed):"
    echo "$diff" | grep '<' | awk '{print $2}'
    echo
  else
    echo " success."
  fi
}

check "skaffold.yaml" "gardener-extension-shoot-rsyslog-relp" "extension"
check "skaffold.yaml" "gardener-extension-shoot-rsyslog-relp-admission" "admission"

if [ "$check_successful" = false ] ; then
  exit 1
fi