FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:latest
COPY --from=builder /app/main /app/main
COPY ./tiles /tiles
CMD ["/app/main"]