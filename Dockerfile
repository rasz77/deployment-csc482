FROM golang:1.27.1-alpine AS builder

WORKDIR /app

COPY ../go.mod ./
COPY main.go ./

RUN go build -o weather-agent main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/weather-agent .

CMD ["./weather-agent"]
