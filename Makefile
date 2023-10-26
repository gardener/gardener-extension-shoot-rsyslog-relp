# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

EXTENSION_PREFIX            := gardener-extension
NAME_ADMISSION              := shoot-rsyslog-relp-admission
NAME                        := shoot-rsyslog-relp
REGISTRY                    := eu.gcr.io/gardener-project/gardener
IMAGE_PREFIX                := $(REGISTRY)/extensions
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
VERSION                     := $(shell cat "$(REPO_ROOT)/VERSION")
EFFECTIVE_VERSION           := $(VERSION)-$(shell git rev-parse HEAD)
GOARCH                      ?= $(shell go env GOARCH)
PARALLEL_E2E_TESTS          := 2

ifndef ARTIFACTS
	export ARTIFACTS=/tmp/artifacts
endif

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif

#########################################
# Tools                                 #
#########################################

TOOLS_DIR := $(REPO_ROOT)/hack/tools
include $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install
install:
	@LD_FLAGS="-w $(shell EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION $(EXTENSION_PREFIX)-$(NAME))" \
	$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install.sh ./...

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-images
docker-images:
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --build-arg TARGETARCH=$(GOARCH) -t $(IMAGE_PREFIX)/$(NAME):$(EFFECTIVE_VERSION) -t $(IMAGE_PREFIX)/$(NAME):latest -f Dockerfile -m 6g --target $(EXTENSION_PREFIX)-$(NAME) .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) --build-arg TARGETARCH=$(GOARCH) -t $(IMAGE_PREFIX)/$(NAME_ADMISSION):$(EFFECTIVE_VERSION) -t $(IMAGE_PREFIX)/$(NAME_ADMISSION):latest -f Dockerfile -m 6g --target $(EXTENSION_PREFIX)-$(NAME_ADMISSION) .

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/.ci/*
	@$(REPO_ROOT)/hack/update-github-templates.sh
	@ln -sf ../vendor/github.com/gardener/gardener/hack/cherry-pick-pull.sh $(HACK_DIR)/cherry-pick-pull.sh

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/clean.sh ./cmd/... ./pkg/... ./test/...

.PHONY: check-generate
check-generate:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM) $(YQ)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-charts.sh ./charts
	@hack/check-skaffold-deps.sh

.PHONY: generate
generate: $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(YQ)
	@GO111MODULE=off hack/update-codegen.sh
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/generate-sequential.sh ./charts/... ./cmd/... ./pkg/... ./test/...

.PHONY: generate-controller-registration
generate-controller-registration:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/generate-sequential.sh ./charts/...

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg ./test

.PHONY: test
test: $(REPORT_COLLECTOR)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh ./cmd/... ./pkg/...

.PHONY: test-integration
test-integration: $(REPORT_COLLECTOR) $(SETUP_ENVTEST)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-integration.sh ./test/integration/...

.PHONY: test-cov
test-cov:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test test-cov test-clean

test-e2e-local: $(GINKGO)
	./hack/test-e2e-local.sh --procs=$(PARALLEL_E2E_TESTS) ./test/e2e/...

ci-e2e-kind: $(KIND)
	./hack/ci-e2e-kind.sh

# use static label for skaffold to prevent rolling all gardener components on every `skaffold` invocation
extension-up extension-down: export SKAFFOLD_LABEL = skaffold.dev/run-id=extension-local

extension-up: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) run

extension-dev: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) dev --cleanup=false --trigger=manual

extension-down: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) delete