FROM golang:1.22-alpine

WORKDIR /app

RUN apk add --no-cache ca-certificates bash

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/server ./cmd/server && \
    go build -o /usr/local/bin/migrate ./cmd/migrate

EXPOSE 8000

CMD ["server"]
