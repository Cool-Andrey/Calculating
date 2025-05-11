FROM golang:1.23.1-alpine AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator ./cmd/orchestrator

FROM alpine:3.21

COPY --from=build /app/ .
COPY --from=build /app/db/migrations/ ./db/migrations/
ENV WRITE_FILE=FALSE
EXPOSE 8080
ENTRYPOINT ["./orchestrator"]

