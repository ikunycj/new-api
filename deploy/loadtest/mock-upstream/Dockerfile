FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY main.go .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mock-upstream main.go

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /mock-upstream /mock-upstream
EXPOSE 8080
ENTRYPOINT ["/mock-upstream"]
