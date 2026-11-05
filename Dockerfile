FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

WORKDIR /netlib

RUN apk --no-cache add --upgrade ca-certificates ca-certificates-bundle git

COPY go* ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o signaling -a -ldflags '-extldflags "-static"' cmd/signaling/main.go

FROM scratch
EXPOSE 8080
ENTRYPOINT ["/netlib-signaling"]

COPY --from=builder /netlib/signaling /netlib-signaling
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
