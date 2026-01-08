FROM golang:1.25-alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/app;
EXPOSE $HTTP_PORT

CMD ["./main"]