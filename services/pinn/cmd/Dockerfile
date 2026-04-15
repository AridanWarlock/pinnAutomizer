FROM golang:1.26.2-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

ENV GOWORK=off

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/pinn/go.mod services/pinn/go.sum ./services/pinn/

WORKDIR /app/services/pinn
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

WORKDIR /app
COPY pkg/ ./pkg/
COPY services/pinn/ ./services/pinn

WORKDIR /app/services/pinn
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build \
    GOMODCACHE=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -ldflags="-s -w" \
    -o /app/exe ./cmd/main.go

FROM scratch
WORKDIR /app
COPY --from=builder ["/app/exe", "."]
ENTRYPOINT ["./exe"]