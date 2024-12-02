FROM golang:1.23.3-alpine@sha256:09742590377387b931261cbeb72ce56da1b0d750a27379f7385245b2b058b63a AS builder

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
