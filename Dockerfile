FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go mod init auth-service && go mod tidy && go build -o auth-service

FROM gcr.io/distroless/base
COPY --from=builder /app/auth-service /
CMD ["/auth-service"]
