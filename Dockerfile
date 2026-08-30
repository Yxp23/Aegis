FROM golang:1.27 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /aegis ./cmd/aegis


FROM alpine:3.22
COPY --from=builder /aegis /aegis
EXPOSE 8080
CMD ["/aegis"]