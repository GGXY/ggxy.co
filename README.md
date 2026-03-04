# ggxy.co

Link shortener and redirect service with regex support, hit counting, and admin UI — built with Go.

Successor to the original NodeJS version.

## Building and Running

Build with `make build`, then run `./redirector`. The server listens on port 8000 by default.

You can change the listen address via:
- `-listen` flag: `./redirector -listen :9000`
- `PORT` env var: `PORT=9000 ./redirector`
- `LISTEN_ADDR` env var: `LISTEN_ADDR=127.0.0.1:9000 ./redirector`

Precedence: `-listen` flag > `LISTEN_ADDR` > `PORT` > default (`:8000`).

## Running as a systemd service

```bash
sudo cp srv.service /etc/systemd/system/srv.service
sudo systemctl daemon-reload
sudo systemctl enable --now srv
```

To restart after code changes:

```bash
make build
sudo systemctl restart srv
```

## Authorization

Admin routes are protected with HTTP Basic Auth via `BASIC_AUTH_USER` / `BASIC_AUTH_PASS` environment variables.

## Database

Uses SQLite (`db.sqlite3`). SQL queries managed with sqlc.

## Code layout

- `cmd/srv`: main package (binary entrypoint)
- `srv`: HTTP server logic (handlers)
- `srv/templates`: Go HTML templates
- `db`: SQLite open + migrations

## Docker

```bash
docker compose up --build
```

---

:kiss: ![debot](http://i.imgur.com/9mswB2S.png)
