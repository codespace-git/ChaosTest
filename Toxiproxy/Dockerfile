
FROM golang:1.24-alpine AS builder


WORKDIR /app

RUN apk add --no-cache gcc musl-dev


COPY . .
RUN go mod tidy
RUN go mod download
RUN go build -o main .

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache wget

COPY --from=builder /app/main  .



HEALTHCHECK --interval=30s --retries=3 --timeout=5s \
CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health || exit 1

CMD ["./main"]
