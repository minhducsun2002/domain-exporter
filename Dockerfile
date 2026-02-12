FROM golang:1.25.6-alpine3.23 as build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /app/exporter ./...

FROM alpine:3.23.3 as certs
RUN apk add --no-cache ca-certificates-bundle

FROM busybox:1.37.0-musl as run
WORKDIR /app
# just copy both for binaries expecting other platforms too
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certs /etc/ssl/cert.pem /etc/ssl/cert.pem

COPY --from=build /app/exporter /app/exporter
ENTRYPOINT /app/exporter