# Bazooka

A modern MUD client: a Go backend owns the MUD connection, a detachable
frontend (TUI today, web later) shows output and takes input, and Python
automations attach over gRPC.

```
 ┌────────────────┐   ┌────────────────────┐
 │ TUI (Bubbletea)│   │ Python bots        │
 │ frontend/tui   │   │ automation/        │
 └───────┬────────┘   └─────────┬──────────┘
         │ gRPC (proto/bazooka/v1)│
 ┌───────▼───────────────────────▼──────────┐
 │ Go backend  backend/                     │
 │  server  → gRPC API                      │
 │  session → state, scrollback, lifecycle  │
 │  router  → fan-out of output to clients  │
 │  mud     → TCP + telnet filtering        │
 └───────────────────┬──────────────────────┘
                     │ TCP
                 MUD server
```

Any number of clients can attach to the same session at once: output is
broadcast to all, and commands from any client are echoed to all (bots tag
their commands, e.g. `[bot:healer] quaff healing`).

## Layout

| Path | Purpose |
|------|---------|
| `proto/bazooka/v1/bazooka.proto` | gRPC service + messages |
| `gen/bazookapb/` | generated Go stubs |
| `backend/cmd/bazooka-server` | backend binary |
| `backend/internal/{mud,router,session,server}` | backend packages |
| `backend/examples/connect` | connect to a MUD without gRPC |
| `frontend/tui` | Bubbletea terminal UI |
| `automation/bazooka` | Python client + `Bot` framework (and generated stubs in `v1/`) |
| `automation/examples` | example bots |

## Quick start

Requires Go 1.24+ and Python 3.9+.

```sh
# 1. backend (connects a session called "main" at startup)
go run ./backend/cmd/bazooka-server -connect mud.example.org:4000

# 2. TUI (another terminal). Esc/Ctrl+C quits, Up/Down = history, PgUp/PgDn = scroll
go run ./frontend/tui -session main
#    or let the TUI create the session:  -connect mud.example.org:4000

# 3. Python automation (another terminal)
pip install -r requirements.txt
python automation/examples/healer.py --session main
```

Quitting the TUI does not disconnect from the MUD; the session keeps running
in the backend and bots keep working. Reattaching replays recent scrollback.

### Writing a bot

```python
from bazooka import Bot, BazookaClient, trigger

class Healer(Bot):
    name = "healer"

    @trigger(r"HP:\s*(\d+)/(\d+)")
    def on_hp(self, m):
        if int(m.group(1)) < 30:
            self.send("quaff healing")

with BazookaClient("127.0.0.1:50051") as c:
    Healer(c, "main").run()
```

## Security note

The gRPC API is unauthenticated and plaintext. The server defaults to
`127.0.0.1`; do not expose it on untrusted networks.

## Docker

```sh
docker build -t bazooka .
docker run -p 127.0.0.1:50051:50051 bazooka -connect mud.example.org:4000
```

## Development

```sh
make test    # go test ./...
make build   # binaries into bin/
make proto   # regenerate Go + Python stubs (needs protoc-gen-go, protoc-gen-go-grpc, grpcio-tools)
```
