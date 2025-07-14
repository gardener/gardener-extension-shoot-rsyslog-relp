#!/usr/bin/env bash
# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
REPO_ROOT_DIR="$(realpath "${SCRIPT_DIR}"/..)"

SHOOT_NAME=""
SHOOT_NAMESPACE=""
ECHO_SERVER_IMAGE=""

parse_flags() {
  while test $# -gt 0; do
    case "$1" in
    --shoot-name)
      shift; SHOOT_NAME="$1"
      ;;
    --shoot-namespace)
      shift; SHOOT_NAMESPACE="$1"
      ;;
    --echo-server-image)
      shift; ECHO_SERVER_IMAGE="$1"
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
    esac
    shift
  done
}

ensure_file_from_template() {
  local file=$1
  local tmpl="${file}".tmpl
  if [[ -n "$2" ]]; then
    tmpl=$2
  fi
  if [[ ! -f "${file}" ]]; then
    echo "Creating \"${file}\" from template."
    cp "${tmpl}" "${file}"
  fi
}

parse_flags "$@"

tmp_shoot_kubeconfig=$(mktemp)
cleanup_shoot_kubeconfig() {
  rm -f "${tmp_shoot_kubeconfig}"
}
trap cleanup_shoot_kubeconfig EXIT

echo "Generating temporary kubeconfig for '${SHOOT_NAMESPACE}/${SHOOT_NAME}'."
cat << EOF | kubectl create --raw /apis/core.gardener.cloud/v1beta1/namespaces/"${SHOOT_NAMESPACE}"/shoots/"${SHOOT_NAME}"/adminkubeconfig -f - | jq -r '.status.kubeconfig' | base64 -d > "${tmp_shoot_kubeconfig}"
{
  "apiVersion": "authentication.gardener.cloud/v1alpha1",
  "kind": "AdminKubeconfigRequest",
  "spec": {
    "expirationSeconds": 600
  }
}
EOF

echo "Installing rsyslog relp echo server into shoot cluster."
echo_server_service="rsyslog-relp-echo-server"
echo_server_namespace="rsyslog-relp-echo-server"
helm upgrade --install \
  --wait \
  --history-max=4 \
  --namespace "${echo_server_namespace}" \
  --create-namespace \
  --kubeconfig "${tmp_shoot_kubeconfig}" \
  --set images.rsyslog="${ECHO_SERVER_IMAGE}" \
  rsyslog-relp-echo-server \
  "${REPO_ROOT_DIR}/example/local/charts/rsyslog-relp-echo-server"

echo "Retrieving ClusterIP of the ${echo_server_namespace}/${echo_server_service} service."
service_ip=$(kubectl --kubeconfig "${tmp_shoot_kubeconfig}" -n "${echo_server_namespace}" get service "${echo_server_service}" -o yaml | yq '.spec.clusterIPs[0]')
if [[ -z "${service_ip}" || "${service_ip}" == "null" ]]; then
  echo "ClusterIP of ${echo_server_namespace}/${echo_server_service} service not assigned."
  exit 1
fi

echo "Deploying rsyslog-relp-tls secret in Garden cluster."
kubectl apply -f <(yq -e ".metadata.namespace = \"${SHOOT_NAMESPACE}\"" "${REPO_ROOT_DIR}/example/secret-rsyslog-tls-certs.yaml")

echo "Deploying audit-config configMap in Garden cluster."
kubectl apply -f <(yq -e ".metadata.namespace = \"${SHOOT_NAMESPACE}\"" "${REPO_ROOT_DIR}/example/configmap-rsyslog-audit.yaml")

extension_config_patch_file="${SHOOT_NAMESPACE}--${SHOOT_NAME}--extension-config-patch.yaml"

echo "Enabling shoot-rsyslog-relp extension by patching shoot with ${extension_config_patch_file}."
ensure_file_from_template "${REPO_ROOT_DIR}/example/extension/${extension_config_patch_file}" "${REPO_ROOT_DIR}/example/extension/extension-config-patch.yaml.tmpl"
yq -ie ".spec.extensions[0].providerConfig.target = \"${service_ip}\"" "${REPO_ROOT_DIR}/example/extension/${extension_config_patch_file}"
kubectl -n "${SHOOT_NAMESPACE}" patch shoot "${SHOOT_NAME}" --patch-file "${REPO_ROOT_DIR}/example/extension/${extension_config_patch_file}"