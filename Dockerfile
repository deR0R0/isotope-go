FROM golang:1.26.2-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /isotope
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o bot .
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./webserver

FROM alpine:latest
RUN apk add --no-cache ca-certificates

WORKDIR /isotope
COPY --from=builder /isotope/bot .
COPY --from=builder /isotope/server ./webserver

EXPOSE 8080