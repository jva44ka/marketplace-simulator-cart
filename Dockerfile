# ---------- BUILD STAGE ----------
FROM golang:latest AS builder

WORKDIR /app

# Копируем зависимости отдельно — кэш
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинарь
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o app ./cmd/server


# ---------- RUNTIME STAGE ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Копируем только бинарник
COPY --from=builder /app/app /app/app

EXPOSE 5000

ENTRYPOINT ["/app/app"]