# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

############# builder
FROM golang:1.20.6 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-rsyslog-relp
COPY . .

ARG EFFECTIVE_VERSION
ARG TARGETARCH

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION GOARCH=$TARGETARCH

############# base
FROM gcr.io/distroless/static-debian11:nonroot AS base

############# gardener-extension-shoot-rsyslog-relp
FROM base AS gardener-extension-shoot-rsyslog-relp
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-shoot-rsyslog-relp /gardener-extension-shoot-rsyslog-relp
ENTRYPOINT ["/gardener-extension-shoot-rsyslog-relp"]

############# gardener-extension-shoot-rsyslog-relp-admission
FROM base AS gardener-extension-shoot-rsyslog-relp-admission
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-shoot-rsyslog-relp-admission /gardener-extension-shoot-rsyslog-relp-admission
ENTRYPOINT ["/gardener-extension-shoot-rsyslog-relp-admission"]

############# rsyslog-relp-echo-server
FROM alpine:3.17.2 AS rsyslog-relp-echo-server
RUN apk update && apk add rsyslog-relp
ARG EFFECTIVE_VERSION
RUN echo "$EFFECTIVE_VERSION" > /etc/VERSION

ENTRYPOINT ["rsyslogd", "-n"]