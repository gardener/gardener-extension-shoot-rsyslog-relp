// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate sh -c "bash $GARDENER_HACK_DIR/generate-crds.sh -p crd- extensions.gardener.cloud resources.gardener.cloud monitoring.coreos.com_v1"
// Only leave resource kinds which are relevant for the tests: managedresources, clusters, extensions
// and the doc.go file used for generation
//go:generate find . -not ( -name *managedresources.yaml -or -name *clusters.yaml -or -name *extensions.yaml -or -name *prometheusrules.yaml -or -name *operatingsystemconfigs.yaml -or -name *scrapeconfigs.yaml -or -name doc.go ) -delete

// Package resources contains generated manifests for CRDs used by the
// lifecycle controller integration tests.
package resources
