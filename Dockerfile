# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

############# builder
FROM golang:1.25.5 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-shoot-rsyslog-relp

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

############# base
FROM gcr.io/distroless/static-debian13:nonroot AS base

############# gardener-extension-shoot-rsyslog-relp
FROM base AS shoot-rsyslog-relp
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-shoot-rsyslog-relp /gardener-extension-shoot-rsyslog-relp
ENTRYPOINT ["/gardener-extension-shoot-rsyslog-relp"]

############# gardener-extension-shoot-rsyslog-relp-admission
FROM base AS shoot-rsyslog-relp-admission
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-shoot-rsyslog-relp-admission /gardener-extension-shoot-rsyslog-relp-admission
ENTRYPOINT ["/gardener-extension-shoot-rsyslog-relp-admission"]

############# rsyslog-relp-echo-server
FROM alpine:3.23.2 AS shoot-rsyslog-relp-echo-server
RUN apk update && apk add rsyslog-relp
ARG EFFECTIVE_VERSION
RUN echo "$EFFECTIVE_VERSION" > /etc/VERSION

ENTRYPOINT ["rsyslogd", "-n"]
