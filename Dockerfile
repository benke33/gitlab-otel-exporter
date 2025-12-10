FROM armdocker.rnd.ericsson.se/dockerhub-ericsson-remote/golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o otel-collector main.go

FROM armdocker.rnd.ericsson.se/dockerhub-ericsson-remote/alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/otel-collector .
CMD ["./otel-collector"]
