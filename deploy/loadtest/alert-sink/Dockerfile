FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY main.go .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /alert-sink main.go

FROM alpine:3.21
COPY --from=builder /alert-sink /alert-sink
EXPOSE 8080
ENTRYPOINT ["/alert-sink"]
