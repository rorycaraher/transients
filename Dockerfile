FROM golang:1.25 AS builder
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
