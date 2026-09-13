# syntax=docker/dockerfile:1
#
# Image for the backend service (/service). Builds a static, distroless binary
# and runs it from scratch. The app container (/app) uses this service as its
# upstream.

# ---- Build stage ---------------------------------------------------------
FROM golang:1.27 AS builder
WORKDIR /src
COPY service/go.mod ./service/go.mod
RUN cd service && go mod download
COPY service/ ./service/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" \
    -o /out/jansvca ./service/cmd/jansvca

# ---- Runtime stage -------------------------------------------------------
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/jansvca /jansvca

VOLUME ["/data"]
ENV JANSVCA_ADDR=:8080 \
    JANSVCA_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/jansvca"]
