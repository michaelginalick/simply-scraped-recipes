FROM golang:1.24.4-alpine3.22 AS builder

ARG CMD=recipe-api

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/${CMD}

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/app .

# /data is where the SQLite file is stored; mount a volume here for persistence
VOLUME ["/data"]

ENTRYPOINT ["./app"]
