FROM golang:1.20.4-alpine@sha256:4ee203ff3933e7a6f18d3574fd6661a73b58c60f028d2927274400f4774aaa41 AS builder

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
