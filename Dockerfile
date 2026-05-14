FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o scrapboard .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/scrapboard .
RUN mkdir -p data/images data/boards
EXPOSE 8080
CMD ["./scrapboard"]
