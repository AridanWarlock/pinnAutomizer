FROM golang:1.26.1-alpine AS builder
RUN apk add --no-cache git

WORKDIR /app
ENV GOMODCACHE=/go/pkg/mod

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build GOMODCACHE=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/exe ./cmd/pinn

FROM scratch
WORKDIR /app
COPY --from=builder ["/app/exe", "."]
ENTRYPOINT ["./exe"]