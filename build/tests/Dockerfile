FROM golang:1.23.1-alpine AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download && go mod verify

COPY . .
CMD go test ./...