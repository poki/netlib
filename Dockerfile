FROM golang:1.23.4-alpine@sha256:c23339199a08b0e12032856908589a6d41a0dab141b8b3b21f156fc571a3f1d3 AS builder

WORKDIR /netlib

RUN apk --no-cache add ca-certificates git

COPY go* ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o signaling -a -ldflags '-extldflags "-static"' cmd/signaling/main.go

FROM scratch
EXPOSE 8080
ENTRYPOINT ["/netlib-signaling"]

ENV CLOUDFLARE_ZONE=
ENV CLOUDFLARE_APP_ID=
ENV CLOUDFLARE_AUTH_USER=
ENV CLOUDFLARE_AUTH_KEY=

COPY --from=builder /netlib/signaling /netlib-signaling
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
