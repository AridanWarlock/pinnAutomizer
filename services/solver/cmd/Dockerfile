FROM golang:1.26.1-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

ENV GOWORK=off

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/solver/go.mod services/solver/go.sum ./services/solver/

WORKDIR /app/services/solver
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

WORKDIR /app
COPY pkg/ ./pkg/
COPY services/solver/ ./services/solver

WORKDIR /app/services/solver
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