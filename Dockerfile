# --- Build stage ---
FROM golang:1.26-bookworm AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/redirector ./cmd/srv

# --- Runtime stage ---
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --shell /bin/bash app

# runtime.Caller(0) in the binary records /src/srv/server.go from the build.
# Templates and static files are resolved relative to that path, so we must
# place them at the same location in the runtime image.
COPY --from=builder /src/srv/templates /src/srv/templates
COPY --from=builder /src/srv/static /src/srv/static
RUN chmod -R a+rX /src/srv

WORKDIR /data

COPY --from=builder /app/redirector /usr/local/bin/redirector

RUN chown app:app /data

USER app

EXPOSE 8000

# SQLite DB lives in /data (the working directory). Mount a volume here.
ENTRYPOINT ["redirector"]
