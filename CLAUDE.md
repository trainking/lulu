# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test/Lint

```bash
go build ./...                  # build all packages
go test -v ./...                # run all tests
go test -v -run TestName ./...  # run a single test
go vet ./...                    # lint with vet
go mod tidy                     # tidy dependencies
```

## Architecture

lulu is a Go game server framework supporting TCP, KCP, and WebSocket. It uses protobuf for message serialization. The protocol is a fixed 4-byte big-endian header (2 bytes body length + 2 bytes opcode) followed by a variable-length protobuf body. Max packet size is 64MB (`network.MaxPacketSize`).

### Core types

- **`App`** (`lulu.go`) — The central orchestrator. Owns the network listener, session manager, and router manager. Created via `lulu.New(config)`, started via `app.Run(modules...)`.
- **`Module`** (`module.go`) — Interface with `Name()`, `OnInit(app)`, `OnDestroy()`, `Route(app)`. All game logic is organized into modules. Modules are initialized in registration order, destroyed in reverse order.
- **`RouterManager`** (`router_manager.go`) — Holds three routing tables:
  - **External** (`handleRouter map[uint16]Router`) — Client-to-server messages, keyed by opcode.
  - **Internal** (`innerRouter map[FullName]Router`) — Server-internal messages (e.g. cross-player actions), keyed by protobuf message full name.
  - **Return** (`outSendMap map[FullName]interface{}`) — Server-to-client messages. No handler, just an opcode mapping so the server knows how to serialize outgoing messages.

### Session lifecycle (`session/`)

- **`Session`** wraps a `network.Conn`. Created on accept, assigned a nanosecond-unique ID.
- A session is "valid" once `SetUserID(id)` is called (typically after auth). A `ValidTimeout` timer fires if the client doesn't authenticate in time — the session is destroyed.
- Valid sessions are added to `SessionManager`, which maintains a `map[userID]*Session`. New sessions for the same userID replace and destroy old ones.
- `Session.Run()` is a read loop that calls `callback.OnMessage` for each incoming packet.
- `CheckFlood(limit)` provides per-minute message rate limiting using atomic counters.

### Network abstraction (`network/`)

- **`Listener`** interface — `Accept() (Conn, error)`, `Close()`. Implemented for TCP, KCP, WebSocket.
- **`Conn`** interface — `ReadPacket()`, `WritePacket()`, `GetRealIP()`, `Close()`.
- **`Packet`** interface (`packet.go`) — `OpCode()`, `BodyLen()`, `Body()`, `Serialize()`, `Free()`. Uses `sync.Pool` for `DefaultPacket` to reduce GC pressure.
- `ListenerFactory` builds the appropriate listener from `Config.NetWork` string (`"tcp"`, `"kcp"`, `"websocket"`).

### Request flow

1. `App.run()` accepts connections in a loop, respecting `ConnMax` limit.
2. Each connection spawns a goroutine that creates a `Session`, fires `app.OnConnect`, starts `s.Run()` (read loop), then waits for either valid timeout or `SetUserID` signal via `s.WaitValid()`.
3. When a packet arrives, `app.OnMessage` is called → flood check → `RouterManager.GetHandleRouter(opcode)` → `asyncHandleMessage`.
4. `asyncHandleMessage` wraps middleware chain around the handler, creates a `Context`, and invokes the handler.
5. `Context.Bind(msg)` unmarshals the protobuf body. `Context.Session()` returns the player's session.

### Middleware

`Middleware` is `func(next Handler) Handler`. Built-in `MiddlewareValidSession()` checks `session.IsValid()` and is automatically applied to all routes unless `WithRegisterIsNoValid(true)` is set (used for auth/login routes).

### Pushing messages to clients

- `app.Action(userID, msg)` — Looks up the session by userID, then calls `app.Call`.
- `app.Call(session, msg)` — Checks if the message has an internal router registered; if yes, executes it. Otherwise, looks up the return opcode and sends via `session.Send()`.
- `session.Send(msg)` — Looks up opcode via `callback.GetMsgOpCode`, marshals the protobuf, packs it, writes to the connection.

### Client (`client.go`)

Client-side helper that dials TCP/KCP/WebSocket connections. `Client.Send(opcode, msg)` packs and sends. `Client.Receive()` returns a channel of incoming packets.

### Key dependencies

- `github.com/gorilla/websocket` — WebSocket support
- `github.com/xtaci/kcp-go` — KCP support
- `google.golang.org/protobuf` — Protobuf serialization
- `gopkg.in/yaml.v3` — Config file parsing
- `github.com/pkg/errors` — Error wrapping
