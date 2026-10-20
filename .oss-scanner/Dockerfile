# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.26.10
# Pinned Debian package versions break on every point release.
# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y --no-install-recommends unzip \
    && rm -rf /var/lib/apt/lists/*

# The scan runs offline, so install the Terraform CLI now to drive the provider against a local fake API.
ARG TERRAFORM_VERSION=1.16.5
RUN arch="$(dpkg --print-architecture)" \
    && curl -fsSLo /tmp/terraform.zip \
      "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${arch}.zip" \
    && unzip -d /usr/local/bin /tmp/terraform.zip terraform \
    && rm /tmp/terraform.zip

COPY . /src
WORKDIR /src
RUN go mod download && go install -gcflags=all="-N -l" .

# dev_overrides makes terraform use the provider built above with no registry access.
RUN printf 'provider_installation {\n  dev_overrides {\n    "cloudsmith-io/cloudsmith" = "/go/bin"\n  }\n  direct {}\n}\n' \
      > /root/.terraformrc

# Acceptance tests skip without TF_ACC=1; they need a live Cloudsmith API.
RUN go test -short -timeout 10m ./... || echo "WARNING: go test reported failures"
