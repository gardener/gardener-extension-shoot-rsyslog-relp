# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

ENSURE_GARDENER_MOD         := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR           := $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
EXTENSION_PREFIX            := gardener-extension
NAME                        := shoot-rsyslog-relp
NAME_ADMISSION              := $(NAME)-admission
NAME_ECHO_SERVER            := $(NAME)-echo-server
IMAGE                       := europe-docker.pkg.dev/gardener-project/releases/gardener/extensions/shoot-rsyslog-relp
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
VERSION                     := $(shell cat "$(REPO_ROOT)/VERSION")
EFFECTIVE_VERSION           := $(VERSION)-$(shell git rev-parse HEAD)
ECHO_SERVER_VERSION         := v0.1.0
IMAGE_TAG                   := $(EFFECTIVE_VERSION)
LD_FLAGS                    := "-w $(shell EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) bash $(GARDENER_HACK_DIR)/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION $(EXTENSION_PREFIX)-$(NAME))"
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

TOOLS_DIR := $(HACK_DIR)/tools
include $(GARDENER_HACK_DIR)/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install
install:
	@LD_FLAGS=$(LD_FLAGS) \
	bash $(GARDENER_HACK_DIR)/install.sh ./cmd/...

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-images
docker-images:
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(IMAGE):$(IMAGE_TAG) -f Dockerfile -m 6g --target $(NAME) .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(IMAGE)-admission:$(IMAGE_TAG) -f Dockerfile -m 6g --target $(NAME_ADMISSION) .

###################################################################
# Rules related to the shoot-rsysog-relp-echo-server docker image #
###################################################################

.PHONY: echo-server-docker-image
echo-server-docker-image:
	@docker build --platform linux/amd64,linux/arm64 --build-arg EFFECTIVE_VERSION=$(ECHO_SERVER_VERSION) -t $(IMAGE)-echo-server:$(ECHO_SERVER_VERSION) -t $(IMAGE)-echo-server:latest -f Dockerfile -m 6g --target $(NAME_ECHO_SERVER) .

.PHONY: push-echo-server-image
push-echo-server-image:
	@docker push $(IMAGE)-echo-server:$(ECHO_SERVER_VERSION)
	@docker push $(IMAGE)-echo-server:latest

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: tidy
tidy:
	@GO111MODULE=on go mod tidy
	@mkdir -p $(REPO_ROOT)/.ci/hack && cp $(GARDENER_HACK_DIR)/.ci/* $(REPO_ROOT)/.ci/hack/ && chmod +xw $(REPO_ROOT)/.ci/hack/*
	@GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(HACK_DIR)/update-github-templates.sh
	@cp $(GARDENER_HACK_DIR)/cherry-pick-pull.sh $(HACK_DIR)/cherry-pick-pull.sh && chmod +xw $(HACK_DIR)/cherry-pick-pull.sh

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@bash $(GARDENER_HACK_DIR)/clean.sh ./cmd/... ./pkg/... ./test/...

.PHONY: check-generate
check-generate:
	@bash $(GARDENER_HACK_DIR)/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM) $(YQ)
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check-charts.sh ./charts
	@GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) $(HACK_DIR)/check-skaffold-deps.sh

.PHONY: generate
generate: $(VGOPATH) $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(YQ)
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./charts/... ./cmd/... ./pkg/... ./test/...

.PHONY: generate-controller-registration
generate-controller-registration:
	@bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./charts/...

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg ./test

.PHONY: test
test: $(REPORT_COLLECTOR)
	@bash $(GARDENER_HACK_DIR)/test.sh ./cmd/... ./pkg/...

.PHONY: test-integration
test-integration: $(REPORT_COLLECTOR) $(SETUP_ENVTEST)
	@bash $(GARDENER_HACK_DIR)/test-integration.sh ./test/integration/...

.PHONY: test-cov
test-cov:
	@bash $(GARDENER_HACK_DIR)/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@bash $(GARDENER_HACK_DIR)/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test test-cov test-clean

test-e2e-local: $(GINKGO)
	./hack/test-e2e-local.sh --procs=$(PARALLEL_E2E_TESTS) ./test/e2e/...

ci-e2e-kind: $(KIND) $(YQ)
	./hack/ci-e2e-kind.sh

# use static label for skaffold to prevent rolling all gardener components on every `skaffold` invocation
extension-up extension-down: export SKAFFOLD_LABEL = skaffold.dev/run-id=extension-local

extension-up: $(SKAFFOLD) $(HELM) $(KUBECTL) $(KIND)
	@LD_FLAGS=$(LD_FLAGS) $(SKAFFOLD) run

extension-dev: $(SKAFFOLD) $(HELM) $(KUBECTL) $(KIND)
	$(SKAFFOLD) dev --cleanup=false --trigger=manual

extension-down: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) delete
