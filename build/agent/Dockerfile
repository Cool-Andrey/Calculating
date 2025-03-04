FROM golang:1.23.1-alpine AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agent ./cmd/agent

FROM alpine:3.21

COPY --from=build /app/agent /app/agent
ENV WRITE_FILE=FALSE
ENTRYPOINT ["/app/agent"]
