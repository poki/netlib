FROM golang:1.26.3-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /netlib

RUN apk --no-cache add ca-certificates git

COPY go* ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o signaling -a -ldflags '-extldflags "-static"' cmd/signaling/main.go

FROM scratch
EXPOSE 8080
ENTRYPOINT ["/netlib-signaling"]

COPY --from=builder /netlib/signaling /netlib-signaling
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
