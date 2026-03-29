# ---------- BUILD STAGE ----------
FROM golang:latest AS builder

WORKDIR /app

# Копируем зависимости отдельно — кэш
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинари
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o server ./cmd/server && \
    go build -o consumer ./cmd/consumer


# ---------- MIGRATOR STAGE ----------
FROM golang:latest AS migrator

WORKDIR /app

COPY migrations ./migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.1

ENTRYPOINT ["goose", "-dir", "/app/migrations", "postgres"]


# ---------- RUNTIME STAGE ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Копируем бинари
COPY --from=builder /app/server /app/server
COPY --from=builder /app/consumer /app/consumer

EXPOSE 5000

ENTRYPOINT ["/app/server"]