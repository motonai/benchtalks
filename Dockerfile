# Stage 1: L E G O
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o benchtalks cmd/benchtalks/main.go

# Stage 2: F1 
FROM scratch
COPY --from=builder /app/benchtalks /benchtalks
EXPOSE 3000

ENTRYPOINT ["/benchtalks"]
