FROM golang:1.26.1-alpine AS builder
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o /app/cmd/pinn/exe /app/cmd/pinn

FROM alpine:3.23.3
WORKDIR /app
COPY --from=builder ["/app/cmd/pinn/exe", "/app/"]
CMD ["/app/exe"]