# Keep the tag in step with `go` in mise.toml and the go directive in go.mod;
# the digest makes the base immutable (Dependabot bumps both).
FROM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS builder
WORKDIR /src

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/transients ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/healthcheck ./cmd/healthcheck

FROM scratch

# scratch has no root CA bundle; without this, any outbound TLS call
# (R2, Cloudflare Queue) fails with "certificate signed by unknown authority".
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /out/transients /transients
COPY --from=builder /out/healthcheck /healthcheck

# Numeric UID with no matching /etc/passwd entry — fine for scratch, and
# keeps the container off root without needing an nss/passwd file.
USER 65532:65532

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD ["/healthcheck"]

ENTRYPOINT ["/transients"]
