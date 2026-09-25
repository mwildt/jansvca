# syntax=docker/dockerfile:1
#
# Image for the IdP (/idp). A minimal OAuth2 provider for local development of
# the BFF: users and clients from a YAML file, in-memory code/token store.

# ---- Build stage ---------------------------------------------------------
FROM golang:1.27 AS builder
WORKDIR /src
COPY idp/go.mod idp/go.sum* ./idp/
RUN cd idp && go mod download
COPY idp/ ./idp/
RUN cd idp && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" \
    -o /out/jansvca-idp ./cmd/idp

# ---- Runtime stage -------------------------------------------------------
FROM alpine:3.24
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 idp
COPY --from=builder /out/jansvca-idp /usr/local/bin/jansvca-idp
COPY idp-config.yaml /etc/jansvca/idp-config.yaml

ENV IDP_ADDR=:8080 \
    IDP_CONFIG=/etc/jansvca/idp-config.yaml
EXPOSE 8080
USER idp
ENTRYPOINT ["/usr/local/bin/jansvca-idp"]
