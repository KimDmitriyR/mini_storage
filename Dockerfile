FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mini_storage ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

RUN useradd --create-home --shell /usr/sbin/nologin appuser \
    && mkdir -p /app/storage \
    && chown -R appuser:appuser /app

COPY --from=builder /out/mini_storage /app/mini_storage

ENV PORT=8080
ENV STORAGE_DIR=/app/storage
ENV MAX_UPLOAD_SIZE_MB=10
ENV DATABASE_PATH=/app/storage/metadata.db

EXPOSE 8080

USER appuser

CMD ["/app/mini_storage"]
