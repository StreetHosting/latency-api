# Optional container image (production uses bare-metal binary + nginx on VPS).
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /latency-probe ./cmd/probe

FROM alpine:3.21
RUN adduser -D -H -s /sbin/nologin probe
USER probe
COPY --from=builder /latency-probe /usr/local/bin/latency-probe
ENV LISTEN_ADDR=0.0.0.0:8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/latency-probe"]
