# syntax=docker/dockerfile:1

# ---- Build stage ------------------------------------------------------------
FROM golang:1.24 AS builder

WORKDIR /src

# Cache module downloads: copy manifests first.
COPY go.mod ./
RUN go mod download

# Copy the rest of the source.
COPY . .

# Build a fully static, stripped binary with CGO disabled.
# -trimpath removes filesystem paths from the binary; -ldflags "-s -w" strips
# debug info and symbol tables to shrink the scratch image.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64}

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags "-s -w" -o /out/jansvca ./cmd/jansvca

# ---- Runtime stage ----------------------------------------------------------
# From scratch for a minimal, distroless image. The binary is fully static,
# so no libc or base filesystem is required.
FROM scratch

# Copy the CA certificates so outbound TLS (e.g. OAuth2 introspection) works
# even from scratch. Sourced from the builder image.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the static binary.
COPY --from=builder /out/jansvca /jansvca

# Data directory for the WAL event stores. Mounted as a volume at runtime.
VOLUME ["/data"]

# Runtime configuration via environment.
ENV JANSVCA_ADDR=:8080 \
    JANSVCA_DATA_DIR=/data

EXPOSE 8080

ENTRYPOINT ["/jansvca"]
